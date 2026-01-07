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

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/synnax/cmd/service"
)

// RunMain is the entry point for the Synnax CLI on Windows. It detects whether the
// process is running as a Windows Service or as an application and routes to the
// appropriate startup path.
func RunMain() {
	isService, err := service.Is()
	cobra.CheckErr(err)
	if isService {
		fmt.Println("Running as service")
		cobra.CheckErr(service.Run())
		return
	}
	Execute()
}
