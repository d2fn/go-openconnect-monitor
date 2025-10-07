package main

import (
	"context"
	"net"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/browserutils/kooky/browser/chrome"
	_ "github.com/browserutils/kooky/browser/all" // register cookie store finders!
)


/**

Types:

OpenConnectProcess:
Provides an interface to the core openconnect process. Can start, stop, and set the DSID.
On connect, reply with metadata about the current run as ConnectionStatus

ConnectionAttemptStatus:
A pure data structure containing metadata about a given connection attempt. If the attempt
was successful it should report the server and client IP address. If not it should report
any error state, specifically if the DSID cookie was rejected by the server.

HealthChecker:
Checks the health of the network by establishing a tearing down a TCP connection
returning the status as OK or DOWN

DSIDPoller:
Polls the Cookie database for the DSID used to connect to Connect

DSIDTracker:
Keeps track of DSIDs and their state. DSID can be marked as 

Timer:
Wakes up every N seconds to run the main program loop, check connection status, poll for new
DSID cookie

Controller:
The monitor's OODA loop. Integrates signals from the main program loop and makes decisions about
when to to reconnect

Configuration:
Holds configurable data such as
- VPN host and endpoint
- Path to cookie file
- Name of DSID cookie
- Host and port to health check against
- Time between health checks. This is also the time at which we check for new cookies

 **/

const (
	healthCheckHost = "8.8.8.8"
	healthCheckPort = "53"
	healthCheckInterval = 1 * time.Second
)


func healthCheck() bool {
	address := net.JoinHostPort(healthCheckHost, healthCheckPort)
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func dsidFromProfile(cookiesPath string, domain string) (string, error) {
	// cookiesFile := "/home/d/.config/google-chrome/" + profile + "/Cookies"
	cookiesSeq := chrome.TraverseCookies(cookiesPath).OnlyCookies()
	for cookie := range cookiesSeq {
		//fmt.Println(cookie.Domain, cookie.Name, cookie.Value)
		if cookie.Domain == domain && cookie.Name == `DSID` {
			return cookie.Value, nil
		}
	}
	return "", fmt.Errorf("[parent] DSID not found for domain %q", domain)
}


func main() {

	config, err := LoadConfig()
	if err != nil {
		log.Printf("Error loading config: %v", err)
		return
	}

	log.Printf("Loaded config:\n%q\n", config)

	c := NewOpenConnectProcess()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loadNewDSID := func() string {
		dsid, err := dsidFromProfile(config.DsidPoller.CookiePath, `pcs.flxvpn.net`)
		if strings.TrimSpace(dsid) == "" {
			log.Printf("[parent] empty DSID")
		} else if err != nil {
			log.Printf("[parent] error getting DSID: %v", err)
		}
		return dsid
	}

	dsid := loadNewDSID()
	c.dsid = dsid

	openConnectStart := func() {
		log.Printf("[parent] Attempting to start openconnect with DSID = %s", c.dsid)
		if err := c.Start(ctx); err != nil {
			log.Printf("[parent] failed to start: %v", err)
		} else {
			log.Printf("[parent] started child")
		}
	}
	openConnectStart()

	openConnectRestart := func(grace time.Duration) {
		c.Stop(grace)
		openConnectStart()
	}
	

	// do a health check periodically
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <- ticker.C:
			if !c.running {
				openConnectStart()
			}
			alive := healthCheck()
			if !alive {
					log.Printf("[parent] connection dropped, restarting")
				// update cookie and restart
				dsid = loadNewDSID()
				if dsid != c.dsid && strings.TrimSpace(dsid) != "" {
					// cookie changed and it's non zero
					log.Printf("DSID changed from %s to %s, restarting openconnect", c.dsid, dsid)
					c.dsid = dsid
				}
				openConnectRestart(5 * time.Second)
			}
		}
	}
}

