package main

import (
	"log"
	"os"
	"time"
)

type Controller struct {
	interval               time.Duration
	healthCheckGracePeriod time.Duration
	dsidFileReader         *DSIDFileReader
	healthChecker          *HealthChecker
	openConnectProcess     *OpenConnectProcess
	dsidTracker            *DSIDTracker
	log                    *log.Logger

	// state variables
	lastHealthyConnectionTime time.Time
}

func NewController(config ControllerConfig, dsidFileReader *DSIDFileReader, healthChecker *HealthChecker, openConnectProcess *OpenConnectProcess) *Controller {
	return &Controller{
		interval:                  time.Duration(config.IntervalSeconds) * time.Second,
		healthCheckGracePeriod:    time.Duration(config.HealthCheckGracePeriodSeconds) * time.Second,
		dsidFileReader:          	 dsidFileReader,
		healthChecker:             healthChecker,
		openConnectProcess:        openConnectProcess,
		dsidTracker:               NewDSIDTracker(),
		lastHealthyConnectionTime: time.Now(),
		log:                       log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (c *Controller) eventLoop() {

	// ask openconnect for the status of its current cookie
	// and mark as rejected if necessary
	// ifcookie is rejected, sigterm openconnect and wait
	currentDSID, rejected := c.openConnectProcess.getDSIDStatus()
	if rejected {
		// cookie rejected, mark as such and shutdown openconnect
		c.dsidTracker.reject(currentDSID)
		if c.openConnectProcess.running {
			c.log.Printf("DSID rejected, killing openconnect process (dsid=%s)", currentDSID)
			c.openConnectProcess.Stop()
		}
	}

	// check if the openconnect process itself has marked itself as unhealthy but is still running
	if c.openConnectProcess.running && c.openConnectProcess.attemptState.needsRestart {
		c.log.Printf("openconnect marked itself as unhealthy, stopping pid=%d", c.openConnectProcess.cmd.Process.Pid)
		c.openConnectProcess.Stop()
	}

	if c.openConnectProcess.running {
		alive := c.healthChecker.check()
		if alive {
			c.lastHealthyConnectionTime = time.Now()
		} else {
			if time.Since(c.lastHealthyConnectionTime) > c.healthCheckGracePeriod {
				// health checks are failing, kill openconnect
				c.log.Printf("Health checks failing for %s, killing current openconnect process %d", c.healthCheckGracePeriod, c.openConnectProcess.cmd.Process.Pid)
				c.openConnectProcess.Stop()
			}
		}
	}

	if dsid, err := c.dsidFileReader.ReadDSID(); err == nil {
		// new dsid cookie available, notify the cookie tracker
		status := c.dsidTracker.notify(dsid)
		switch status {
		case Accepted:
			{
				// dsid changed, kill openconnect
				c.log.Printf("DSID changed: %s", dsid)
				c.openConnectProcess.dsid = c.dsidTracker.current
				c.openConnectProcess.Stop()
			}
		}
	} else {
		c.log.Printf("Error getting DSID cookie: %v", err)
	}

	if !c.openConnectProcess.running && c.dsidTracker.current == c.openConnectProcess.dsid {
		c.log.Printf("Starting openconnect")
		c.openConnectProcess.Start()
	}
}

func (c *Controller) Start() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.eventLoop()
		}
	}
}

