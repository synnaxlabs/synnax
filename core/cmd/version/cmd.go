// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of Synnax",
	Long:  "Print the version of Synnax.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return FPrint(cmd.OutOrStdout())
	},
}
