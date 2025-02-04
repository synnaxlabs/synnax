// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/synnaxlabs/synnax/pkg/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of Synnax",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Synnax version %s\n", version.Get())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
