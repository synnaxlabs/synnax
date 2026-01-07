// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version

import "github.com/spf13/cobra"

var command = &cobra.Command{
	Use:   "version",
	Short: "Print the version of Synnax",
	Run:   func(*cobra.Command, []string) { Print() },
}

// AddCommand registers the version command to the given parent command.
func AddCommand(cmd *cobra.Command) { cmd.AddCommand(command) }
