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
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jerky",
	Short: "Generate protobuf definitions from Go structs",
	Long: `Jerky generates protobuf definitions and translation functions
from Go struct definitions annotated with //go:generate jerky.

When invoked via go generate, jerky reads the GOFILE environment
variable to determine which file to process.

Examples:
  //go:generate jerky
  type User struct { ... }     // Storage type: generates gorp + migrator

  //go:generate jerky embedded
  type Address struct { ... }  // Embedded type: translation functions only

Then run: go generate ./...`,
	// Allow arbitrary args for backward compatibility with "jerky embedded"
	Args: cobra.ArbitraryArgs,
	// Default behavior when no subcommand is specified: run generate
	// This allows backward compatibility with //go:generate jerky
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGenerate(cmd, args)
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
