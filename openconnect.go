package main

import (
	"sync"

	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type OpenConnectProcess struct {

	// connection config
	url                 string
	dsid                string
	shutdownGracePeriod time.Duration

	// openconnect command settings
	extraArgs string
	verbose   bool
	dryRun    bool

	// process management
	mu      sync.Mutex
	env     []string
	ctx     context.Context
	cmd     *exec.Cmd
	running bool

	// connection attempt state
	attemptState *ConnectionAttemptState

	// logger
	log *log.Logger
}

/*
ConnectionAttemptStatus:
A pure data structure containing metadata about a given connection attempt. If the attempt
was successful it should report the server and client IP address. If not it should report
any error state, specifically if the DSID cookie was rejected by the server.
*/

type ConnectionAttemptState struct {
	success      bool
	hostAddr     string
	clientAddr   string
	rejectedDSID string
	needsRestart bool
}

func NewOpenConnectProcess(vpnConfig VPNConfig, openConnectConfig OpenConnectConfig, ctx context.Context) *OpenConnectProcess {
	return &OpenConnectProcess{
		env:                 os.Environ(),
		ctx:                 ctx,
		url:                 vpnConfig.Url,
		shutdownGracePeriod: time.Duration(openConnectConfig.ShutdownGracePeriodSeconds) * time.Second,
		extraArgs:           openConnectConfig.ExtraArgs,
		verbose:             openConnectConfig.Verbose,
		dryRun:              openConnectConfig.DryRun,
		attemptState:        &ConnectionAttemptState{},
		log:                 log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (p *OpenConnectProcess) parseStdout(in io.ReadCloser) {
	// continually read from stdin looking looking for updates to push into our ConnectionAttemptState
	defer in.Close()
	sc := bufio.NewScanner(in)
	for sc.Scan() {
		line := sc.Text()
		if p.verbose {
			p.log.Printf(line)
		}
		if strings.HasPrefix(line, "Connected to ") {
			// found ip address of vpn host
			parts := strings.Split(line, " ")
			if len(parts) == 3 {
				host := strings.Split(parts[2], ":")[0]
				p.attemptState.hostAddr = host
				p.log.Printf("Connected to remote %s", host)
			}
		} else if strings.HasPrefix(line, "Configured as ") {
			// found ip address of client
			host := strings.Split(line, " ")[2]
			p.attemptState.clientAddr = host
			p.log.Printf("Configured client as %s", host)
		} else if strings.HasPrefix(line, "Session authentication will expire at ") {
			if p.attemptState.hostAddr != "" && p.attemptState.clientAddr != "" && p.attemptState.rejectedDSID == "" {
				p.attemptState.success = true
				p.log.Printf("Successfully connected to remote %s as %s", p.attemptState.hostAddr, p.attemptState.clientAddr)
			}
		}
	}
	if err := sc.Err(); err != nil {
		p.log.Printf("stream error: %v", err)
	}
}

func (p *OpenConnectProcess) parseStderr(r io.ReadCloser) {
	defer r.Close()
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		pulsePacketSpam := strings.HasPrefix(line, "Unknown Pulse packet of ");
		if p.verbose && !pulsePacketSpam {
			p.log.Printf(line)
		}
		if strings.HasPrefix(line, "ESP detected dead peer") {
			// todo: signal that we need a restart
			p.attemptState.needsRestart = true
		} else if strings.HasPrefix(line, "Cookie was rejected by server") {
				p.attemptState.rejectedDSID = p.dsid
				// todo: imediately mark cookie as rejected
				p.log.Printf("DSID cookie rejected by server: %s", p.dsid)
		}
	}
	if err := sc.Err(); err != nil {
		p.log.Printf("stream error: %v", err)
	}
}

// get the current dsid and whether or not we saw it rejected
func (p *OpenConnectProcess) getDSIDStatus() (string, bool) {
	return p.dsid, p.dsid == p.attemptState.rejectedDSID
}

func (p *OpenConnectProcess) Start() error {

	if strings.TrimSpace(p.dsid) == "" {
		return errors.New("no_dsid")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return errors.New("VPN client already running")
	}

	name := "openconnect"
	args := []string{"-C", p.dsid, "--protocol=pulse"}
	if p.extraArgs != "" {
		for _, arg := range strings.Split(p.extraArgs, " ") {
			args = append(args, arg)
		}
	}
	args = append(args, p.url)

	if p.dryRun {
		log.Printf("[dry run] %s %s", name, strings.Join(args, " "))
		p.running = true
		return nil
	}

	cmd := exec.CommandContext(p.ctx, name, args...)
	cmd.Env = p.env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	// clear state
	p.attemptState = &ConnectionAttemptState{success: false}
	go p.parseStdout(stdout)
	go p.parseStderr(stderr)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting process: %w", err)
	} else {
		p.log.Printf("[child] openconnect pid = %d", cmd.Process.Pid)
	}

	p.cmd = cmd
	p.running = true

	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		if err != nil {
			p.log.Printf("[child] exited with error: %v", err)
		} else {
			p.log.Printf("[child] exited")
		}
	}()

	return nil
}

func (p *OpenConnectProcess) Restart() {
	p.Stop()
	p.Start()
}

func (p *OpenConnectProcess) Stop() {

	p.mu.Lock()
	cmd := p.cmd
	p.mu.Unlock()
	if cmd == nil { //|| !running {
		return
	}
	pgid, _ := syscall.Getpgid(cmd.Process.Pid)

	// Try graceful first.
	_ = syscall.Kill(-pgid, syscall.SIGTERM) // negative => process group
	waitCh := make(chan struct{})
	go func() {
		cmd.Wait() // already reaped in Start goroutine, but safe to wait again
		close(waitCh)
	}()

	// wait for shutdown or force SIGKILL
	select {
	case <-waitCh:
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		return
	case <-time.After(p.shutdownGracePeriod):
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		p.mu.Lock()
		p.running = false
		p.attemptState = &ConnectionAttemptState{success: false}
		p.mu.Unlock()
	}
}
