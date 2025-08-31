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
	"os"

	"github.com/spf13/cobra"
)

var Root = &cobra.Command{
	Use:   "slate",
	Short: "A reactive telemetry and control language",
	Long: `Slate is a domain-specific language for telemetry processing and hardware control.
It provides a reactive programming model optimized for real-time systems with
compile-time safety guarantees and predictable execution behavior.`,
	Version: "0.1.0",
}

// Execute is the entrypoint for the CLI.
func Execute() {
	if err := Root.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	Root.AddCommand(compileCmd)
	Root.AddCommand(lspCmd)
}
