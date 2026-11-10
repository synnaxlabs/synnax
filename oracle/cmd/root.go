// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package cmd provides the command-line interface for Oracle.
package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newRootCmd creates a fresh command tree. This is the primary entry point for both
// production use and testing. Each call returns an isolated tree with its own flag
// state, so tests can run in parallel without interference.
func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "oracle",
		Short: "Schema-first code generation for Synnax",
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
		// Execute prints each error once, so cobra prints nothing.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.Version = "dev"
	configureRootFlags(rootCmd)
	bindFlags(rootCmd)
	cobra.OnInitialize(initConfig)

	migrateCmd := newMigrateCmd()

	rootCmd.AddCommand(
		newCheckCmd(),
		newFmtCmd(),
		newLSPCmd(),
		migrateCmd,
		newSurfaceCmd(),
		newSyncCmd(),
	)
	return rootCmd
}

// Execute runs the root command with args and returns the process exit code. A command
// that fails with a gate failure returns that gate's code (e.g. 12 for generated drift
// in `oracle check`). Every other failure prints its error to stderr and returns 1.
func Execute(args []string) int {
	root := newRootCmd()
	root.SetArgs(args)
	err := root.Execute()
	if err == nil {
		return 0
	}
	// A gate failure's report is already on stdout.
	if ec, ok := err.(*exitCodeError); ok {
		return ec.code
	}
	printError(err.Error())
	return 1
}

func initConfig() {
	viper.SetEnvPrefix("oracle")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
}
