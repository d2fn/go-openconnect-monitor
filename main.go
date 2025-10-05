package main

import (
	"bufio"
	"context"
	"net"
	"fmt"
	"io"
	"log"
	"errors"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Child struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	name    string
	args    []string
	env     []string
	running bool
}

const (
	healthCheckHost = "8.8.8.8"
	healthCheckPort = "53"
	healthCheckInterval = 5 * time.Second
)

func NewChild(name string, args []string) *Child {
	return &Child{name: name, args: args, env: os.Environ()}
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

func (c *Child) Start(ctx context.Context) error {

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return errors.New("child already running")
	}

	cmd := exec.CommandContext(ctx, c.name, c.args...)
	cmd.Env = c.env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	} else {
		log.Printf("setup stdout")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	} else {
		log.Printf("setup stderr")
	}

	go stream("STDOUT", stdout)
	go stream("STDERR", stderr)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting process: %w", err)
	} else {
		log.Printf("[child] monitoring process with pid %d", cmd.Process.Pid)
	}

	c.cmd = cmd
	c.running = true

	go func() {
		err := cmd.Wait()
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		if err != nil {
			log.Printf("[child] exited with error: %v", err)
		} else {
			log.Printf("[child] exited")
		}
	}()

	return nil
}

func (c *Child) Stop(grace time.Duration) {
	c.mu.Lock()
	cmd := c.cmd
	running := c.running
	c.mu.Unlock()
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
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		return
	case <-time.After(grace):
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
	}
}

// Start child initially
func mustStart(child *Child, ctx context.Context) {
}

func healthCheck() bool {
	address := net.JoinHostPort(healthCheckHost, healthCheckPort)
	log.Printf("[parent] [%s] checking health", address)
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	log.Printf("[parent] [%s] health check OK", address)
	return true
}

func main() {

	c := NewChild("ping", []string { healthCheckHost })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mustStart := func() {
		if err := c.Start(ctx); err != nil {
			log.Printf("[parent] failed to start: %v", err)
		} else {
			log.Printf("[parent] started child")
		}
	}
	mustStart()

	// do a health check periodically
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <- ticker.C:
			alive := healthCheck()
			if alive {
				log.Printf("[parent] monitoring pid %d", c.cmd.Process.Pid)
			} else {
				log.Printf("[parent] connection dropped, restarting")
				c.Stop(1 * time.Second);
				mustStart()
			}
		}
	}
}

