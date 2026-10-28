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

// New returns the root oracle command with every subcommand attached. Each call returns
// a separate command tree.
func New() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "oracle",
		Short: "Schema-first code generation for Synnax",
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
		// A failed RunE is a normal command outcome (e.g. `check` reporting drift,
		// `sync` hitting a generation error). The rendered output above the error
		// already says everything the user needs to know; cobra's default usage dump
		// and duplicate "Error: ..." line just add noise. Subcommands that want to
		// print their error explicitly do so via printError; the rest rely on the exit
		// code carried by exitCodeError.
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
// in `oracle check`). Every other failure returns 1.
func Execute(args []string) int {
	root := New()
	root.SetArgs(args)
	err := root.Execute()
	if err == nil {
		return 0
	}
	if ec, ok := err.(*exitCodeError); ok {
		return ec.code
	}
	return 1
}

func initConfig() {
	viper.SetEnvPrefix("oracle")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
}
