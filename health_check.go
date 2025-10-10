package main

import (
	"net"
	"time"
)

type HealthChecker struct {
	host    string
	port    string
	timeout time.Duration
}

func NewHealthChecker(config HealthCheckConfig) *HealthChecker {
	return &HealthChecker{host: config.Host, port: config.Port, timeout: time.Duration(config.TimeoutSeconds) * time.Second}
}

func (healthChecker *HealthChecker) check() bool {
	address := net.JoinHostPort(healthChecker.host, healthChecker.port)
	d := net.Dialer{Timeout: healthChecker.timeout}
	conn, err := d.Dial("tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
