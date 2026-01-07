// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package cli provides the command-line interface for Oracle.
package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// BuildTime is injected at build time via -ldflags.
var BuildTime = "dev"

var rootCmd = &cobra.Command{
	Use:   "oracle",
	Short: "Schema-first code generation for Synnax",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = BuildTime
	configureRootFlags()
	bindFlags(rootCmd)
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// Environment variable support with ORACLE_ prefix
	viper.SetEnvPrefix("oracle")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
}
