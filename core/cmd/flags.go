// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import "github.com/spf13/cobra"

const (
	flagVersion = "version"
	flagConfig  = "config"
)

func addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP(
		flagConfig,
		"c",
		"/usr/local/synnax/config.yaml",
		"The path to the configuration file",
	)
	cmd.Flags().Bool(flagVersion, false, "Print the version of Synnax")
}
