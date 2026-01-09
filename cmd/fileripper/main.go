// This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
// If a copy of the MPL was not distributed with this file, You can obtain one at
// https://mozilla.org/MPL/2.0/.

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"fileripper/internal/core"
	"fileripper/internal/network"
	"fileripper/internal/pfte"
	"fileripper/internal/server"
)

func main() {
	fmt.Println("FileRipper v0.1.0 - Powered by PFTE (Go Edition)")

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "start-server":
		// Daemon mode.
		// Default port is now 9897 as requested.
		port := 9897
		if len(os.Args) > 2 {
			p, err := strconv.Atoi(os.Args[2])
			if err == nil {
				port = p
			}
		}
		server.StartDaemon(port)

	case "transfer":
		handleTransferCLI(os.Args)
		
	default:
		fmt.Printf("Error: %v: %s\n", core.ErrUnknownCommand, command)
		printUsage()
	}
}

func handleTransferCLI(args []string) {
	if len(args) < 6 {
		fmt.Println("Error: Missing arguments.")
		fmt.Println("Usage: fileripper transfer <host> <port> <user> <password> [--all]")
		return
	}

	host := args[2]
	portStr := args[3]
	user := args[4]
	password := args[5]
	
	downloadAll := false
	if len(args) > 6 && strings.ToLower(args[6]) == "--all" {
		downloadAll = true
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		fmt.Println("Error: Invalid port number.")
		return
	}

	fmt.Printf(">> CLI Transfer mode engaged. Target: %s@%s:%d\n", user, host, port)

	session := network.NewSession(host, port, user, password)
	defer session.Close()

	if err := session.Connect(); err != nil {
		os.Exit(1)
	}

	if err := session.OpenSFTP(); err != nil {
		os.Exit(1)
	}

	engine := pfte.NewEngine()
	if err := engine.StartTransfer(session, downloadAll); err != nil {
		fmt.Printf("Error during transfer: %v\n", core.ErrPipelineStalled)
	}
}

func printUsage() {
	fmt.Println(`
Usage: fileripper [command] [args]

Commands:
  start-server [port]   Start REST API Daemon (Default port: 9897)
  transfer              CLI mode (Debug/Scripts)
`)
}