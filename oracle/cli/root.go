// Copyright 2025 Synnax Labs, Inc.
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
	Short: "Schema-first code generation for Synnax metadata structures",
	Long: `Oracle is a schema-first code generation system that parses .oracle schema
files, analyzes imports and type references, and invokes plugins to generate
type-safe code across Go, TypeScript, and Python.

Output locations are declared per-struct in schema files using struct-level domains:

    struct Range {
        field key uuid { domain id }
        field name string { domain validate { required } }

        domain go { output "core/ranger" }
        domain ts { output "console/src/ranger" }
    }

See docs/tech/rfc/0026-251229-oracle-schema-system.md for the full specification.`,
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
