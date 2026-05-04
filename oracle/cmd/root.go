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
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// BuildTime is injected at build time via -ldflags.
var BuildTime = "dev"

// NewRootCmd creates a fresh command tree. This is the primary entry point for
// both production use and testing. Each call returns an isolated tree with its
// own flag state, so tests can run in parallel without interference.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "oracle",
		Short: "Schema-first code generation for Synnax",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		// A failed RunE is a normal command outcome (e.g. `check`
		// reporting drift, `sync` hitting a generation error). The
		// rendered output above the error already says everything the
		// user needs to know; cobra's default usage dump and duplicate
		// "Error: ..." line just add noise. Subcommands that want to
		// print their error explicitly do so via printError; the rest
		// rely on the exit code carried by exitCodeError.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.Version = BuildTime
	configureRootFlags(rootCmd)
	bindFlags(rootCmd)
	cobra.OnInitialize(initConfig)

	migrateCmd := newMigrateCmd()
	migrateCmd.AddCommand(newMigrateCreateCmd())

	rootCmd.AddCommand(
		newCheckCmd(),
		newFmtCmd(),
		newLSPCmd(),
		migrateCmd,
		newSnapshotCmd(),
		newSyncCmd(),
	)
	return rootCmd
}

// Execute runs the root command. Commands that fail with an
// exitCodeError are routed to that specific exit code so CI consumers
// can attribute failures (e.g. `oracle check` returns 12 on generated
// drift). Every other failure exits with 1.
func Execute() {
	err := NewRootCmd().Execute()
	if err == nil {
		return
	}
	if ec, ok := err.(*exitCodeError); ok {
		os.Exit(ec.code)
	}
	os.Exit(1)
}

func initConfig() {
	viper.SetEnvPrefix("oracle")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
}
