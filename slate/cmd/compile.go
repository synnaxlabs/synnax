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
)

var compileCmd = &cobra.Command{
	Use:   "compile [file]",
	Short: "Compile a Slate program",
	Long:  `Compile a Slate source file (not yet implemented)`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("compile command not yet implemented")
	},
}
