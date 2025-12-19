// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build windows

package cmd

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/svc"
)

func disablePermissionBits() {}

// RunMain is the entry point for the Synnax CLI on Windows.
// It detects whether the process is running as a Windows Service or as a console application
// and routes to the appropriate startup path.
func RunMain() {
	isService, err := svc.IsWindowsService()
	if err != nil {
		// If we can't determine, assume console mode
		fmt.Fprintf(os.Stderr, "Warning: failed to detect service mode: %v\n", err)
		Execute()
		return
	}

	if isService {
		// Running as a Windows Service
		if err := runAsWindowsService(); err != nil {
			fmt.Fprintf(os.Stderr, "Service error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Running as a console application
		Execute()
	}
}
