// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/x/jerky"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print jerky version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("jerky version %s\n", jerky.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
