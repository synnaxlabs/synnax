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
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "arc",
	Short: "Arc programming language compiler and tools",
	Long: `Arc is a domain-specific programming language for control systems.

The Arc compiler takes .arc source files and compiles them to WebAssembly
for execution in the Synnax runtime environment.

This CLI also provides a Language Server Protocol (LSP) implementation for
editor integration, providing features like code completion, hover information,
and diagnostics.`,
	Version: "0.1.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(lspCmd)
}
