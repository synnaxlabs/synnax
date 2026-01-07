// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package start

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:     "start",
	Short:   "Starts a Synnax Core",
	Long:    "Starts a Synnax Core using the data directory specified by the --data flag, and listening on the address specified by the --listen flag. If --peers is specified and no existing data is found, the Core will attempt to join the cluster formed by its peers. If no peers are specified and no existing data is found, the Core will bootstrap a new cluster.",
	Example: "synnax start --listen localhost:9091 --data /mnt/ssd1 --peers localhost:9092,localhost:9093 --insecure",
	Args:    cobra.NoArgs,
	Run:     func(cmd *cobra.Command, _ []string) { start(cmd) },
}

// AddCommand adds the start command to the given parent command.
func AddCommand(cmd *cobra.Command) error {
	BindFlags(startCmd)
	cmd.AddCommand(startCmd)
	return viper.BindPFlags(startCmd.Flags())
}
