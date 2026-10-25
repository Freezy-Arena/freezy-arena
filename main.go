// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package main

import (
	"flag"
	"github.com/Team254/cheesy-arena/field"
	"github.com/Team254/cheesy-arena/network"
	"github.com/Team254/cheesy-arena/web"
	"log"
	"net"
	"strconv"
)

const eventDbPath = "./event.db"
const httpPort = 8080

// Main entry point for the application.
func main() {
	flag.BoolVar(&network.DevMode, "dev", false, "Bind driver station listeners to all IP addresses for development")
	var kioskPort int
	var kioskBind string
	flag.IntVar(&kioskPort, "kiosk-port", 8081, "Dedicated kiosk HTTP port (0 disables kiosk listener)")
	flag.StringVar(&kioskBind, "kiosk-bind", "0.0.0.0", "Interface address on which to listen for kiosk requests")
	flag.Parse()
	if kioskPort < 0 || kioskPort > 65535 || kioskPort == httpPort {
		log.Fatalf("Invalid kiosk port %d: choose 0 to disable or a port other than %d", kioskPort, httpPort)
	}

	arena, err := field.NewArena(eventDbPath)
	if err != nil {
		log.Fatalln("Error during startup: ", err)
	}

	// Start the web server in a separate goroutine.
	web := web.NewWeb(arena)
	web.KioskPort = kioskPort
	go web.ServeWebInterface(httpPort)
	if kioskPort != 0 {
		go web.ServeKioskInterface(net.JoinHostPort(kioskBind, strconv.Itoa(kioskPort)))
	}

	// Run the arena state machine in the main thread.
	arena.Run()
}
