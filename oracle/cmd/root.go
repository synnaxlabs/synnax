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

// Execute runs the root command.
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetEnvPrefix("oracle")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
}
