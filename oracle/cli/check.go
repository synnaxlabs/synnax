// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle/analyzer"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate schemas without generating code",
	Long: `Parse and analyze .oracle schema files to check for errors
without generating any code (dry run).

By default, looks for schemas in schemas/*.oracle.

This is useful for CI/CD pipelines or pre-commit hooks to validate
schema changes before code generation.`,
	Example: `  oracle check
  oracle check -s "other/*.oracle"
  oracle check -v`,
	RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
	configureCheckFlags()
	bindFlags(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	verbose := viper.GetBool(verboseFlag)
	schemaPatterns := viper.GetStringSlice(schemasFlag)
	if len(schemaPatterns) == 0 {
		schemaPatterns = []string{"schemas/*.oracle"}
	}
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	schemaFiles, err := expandGlobs(schemaPatterns, wd)
	if err != nil {
		return err
	}
	if len(schemaFiles) == 0 {
		return fmt.Errorf("no schema files found matching patterns: %v", schemaPatterns)
	}

	if verbose {
		fmt.Printf("Checking %d schema file(s)...\n", len(schemaFiles))
		for _, f := range schemaFiles {
			fmt.Printf("  - %s\n", f)
		}
	}

	loader := analyzer.NewStandardFileLoader(wd)
	table, diag := analyzer.Analyze(ctx, schemaFiles, loader)
	if diag != nil && !diag.Empty() {
		fmt.Fprintln(os.Stderr, diag.String())
	}

	if diag != nil && diag.HasErrors() {
		return fmt.Errorf("schema validation failed with %d error(s)", len(diag.Errors()))
	}
	if table != nil {
		structCount := len(table.AllStructs())
		enumCount := len(table.AllEnums())
		fmt.Printf("Validation passed: %d struct(s), %d enum(s)\n", structCount, enumCount)
	}
	return nil
}
