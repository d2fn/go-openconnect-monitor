package main

import (
	"context"
)

/**

Types:

OpenConnectProcess:
Provides an interface to the core openconnect process. Can start, stop, and set the DSID.
On connect, reply with metadata about the current run as ConnectionStatus

ConnectionAttemptState:
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

func main() {

	config, err := LoadConfig()
	if err != nil {
		return
	}

	dsidPoller := NewDSIDPoller(config.DsidPoller)
	healthChecker := NewHealthChecker(config.HealthCheck)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	openConnectProcess := NewOpenConnectProcess(config.Vpn, config.OpenConnect, ctx)

	controller := NewController(config.Controller, dsidPoller, healthChecker, openConnectProcess)
	controller.Start()
}
