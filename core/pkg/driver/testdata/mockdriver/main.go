// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	var (
		debug      bool
		configPath string
	)
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "start", "--standalone", "--disable-sig-stop", "--no-color":
			continue
		case "--debug":
			debug = true
		case "--config":
			if i+1 < len(args) {
				i++
				configPath = args[i]
			}
		}
	}

	if configPath != "" {
		if _, err := os.Stat(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "E [mock] [main.go] config file not found: %s\n", configPath)
			os.Exit(1)
		}
	}

	if os.Getenv("MOCK_FAIL_START") == "1" {
		fmt.Fprintln(os.Stdout, "E [mock] [main.go] startup failure")
		os.Exit(1)
	}

	if delayStr := os.Getenv("MOCK_DELAY_MS"); delayStr != "" {
		if ms, err := strconv.Atoi(delayStr); err == nil && ms > 0 {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}

	if debug {
		fmt.Fprintln(os.Stdout, "D [mock] [main.go] debug mode enabled")
	}

	fmt.Fprintln(os.Stdout, "I [mock] [main.go] started successfully")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if scanner.Text() == "STOP" {
			fmt.Fprintln(os.Stdout, "I [mock] [main.go] received stop signal")
			os.Exit(0)
		}
	}
}
