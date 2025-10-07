package main

import (
	"sync"

	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"strings"
	"time"

)

type OpenConnectProcess struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	env     []string
	running bool
	dsid    string
}

func NewOpenConnectProcess() *OpenConnectProcess {
	return &OpenConnectProcess{env: os.Environ()}
}

func stream(tag string, r io.ReadCloser) {
	defer r.Close()
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		log.Printf("[%s] %s", tag, sc.Text())
	}
	if err := sc.Err(); err != nil {
		log.Printf("[%s] stream error: %v", tag, err)
	}
}

func (p *OpenConnectProcess) Start(ctx context.Context) error {

	if strings.TrimSpace(p.dsid) == "" {
		return errors.New("no DSID set")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return errors.New("vpn client already running")
	}

	name := "openconnect"
	args := []string { "-C", p.dsid, "--protocol=pulse", "https://pcs.flxvpn.net/emp" }

	cmd := exec.CommandContext(ctx, name, args...)
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

	go stream("STDOUT", stdout)
	go stream("STDERR", stderr)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting process: %w", err)
	} else {
		log.Printf("[child] openconnect pid = %d", cmd.Process.Pid)
	}

	p.cmd = cmd
	p.running = true

	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		if err != nil {
			log.Printf("[child] exited with error: %v", err)
		} else {
			log.Printf("[child] exited")
		}
	}()

	return nil
}

func (p *OpenConnectProcess) Stop(grace time.Duration) {
	p.mu.Lock()
	cmd := p.cmd
	running := p.running
	p.mu.Unlock()
	if cmd == nil || !running {
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

	select {
	case <-waitCh:
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		return
	case <-time.After(grace):
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}
}


/**
❯ go run .
2025/10/07 12:17:06 [parent] Attempting to start openconnect with DSID = abs123XXXX
2025/10/07 12:17:06 [child] openconnect pid = 8461
2025/10/07 12:17:06 [parent] started child
// when we see the following line that starts with "Connected to $IP", set the VPN host in the connection status
2025/10/07 12:17:07 [STDOUT] Connected to 192.000.000.000:443
2025/10/07 12:17:07 [STDOUT] SSL negotiation with pcs.flxvpn.net
2025/10/07 12:17:07 [STDOUT] Connected to HTTPS on pcs.flxvpn.net with ciphersuite (TLS1.2)-(RSA)-(AES-128-GCM)
2025/10/07 12:17:07 [STDOUT] Got HTTP response: HTTP/1.1 101 Switching Protocols
// not sure what to do with this, log it maybe
2025/10/07 12:17:07 [STDOUT] Unexpected Pulse configuration packet: wrong type field (!= 1)
// when we see the line starting "Configured as $IP" set the client IP in the connection status
2025/10/07 12:17:07 [STDOUT] Configured as 172.000.000.000 + xxxxxxxxxxxxxxxxxxxxxxxxx, with SSL connected and ESP in progress
// when we see "Session authentication will expire at $date", parse date and set as expiry time in connection status
2025/10/07 12:17:07 [STDOUT] Session authentication will expire at Wed Oct  8 09:54:32 2025
2025/10/07 12:17:07 [STDOUT] 
2025/10/07 12:17:07 [STDERR] mkdir: cannot create directory ‘/var/run/vpnc’: Permission denied
2025/10/07 12:17:07 [STDERR] Failed to bind local tun device (TUNSETIFF): Operation not permitted
2025/10/07 12:17:07 [STDERR] To configure local networking, openconnect must be running as root
2025/10/07 12:17:07 [STDERR] See https://www.infradead.org/openconnect/nonroot.html for more information
2025/10/07 12:17:07 [STDERR] Set up tun device failed
2025/10/07 12:17:07 [STDOUT] Unrecoverable I/O error; exiting.
2025/10/07 12:17:07 [child] exited with error: exit status 1
2025/10/07 12:17:07 [parent] Attempting to start openconnect with DSID = abc123XXXXX
2025/10/07 12:17:07 [child] openconnect pid = 8477
2025/10/07 12:17:07 [parent] started child
2025/10/07 12:17:07 [STDOUT] Connected to 192.173.91.18:443
2025/10/07 12:17:08 [STDOUT] SSL negotiation with pcs.flxvpn.net
2025/10/07 12:17:08 [STDOUT] Connected to HTTPS on pcs.flxvpn.net with ciphersuite (TLS1.2)-(RSA)-(AES-128-GCM)
2025/10/07 12:17:08 [STDOUT] Got HTTP response: HTTP/1.1 101 Switching Protocols
2025/10/07 12:17:08 [STDERR] Authentication failure: Code 0x00
2025/10/07 12:17:08 [STDERR] Creating SSL connection failed
2025/10/07 12:17:08 [STDERR] Cookie was rejected by server; exiting.
2025/10/07 12:17:08 [child] exited with error: exit status 2
^Csignal: interrupt
**/
