// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/x/errors"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate schemas without generating code",
	RunE:  runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	_ = viper.GetBool(verboseFlag)

	printBanner()

	repoRoot, err := paths.RepoRoot()
	if err != nil {
		printError("must be run within a git repository")
		return err
	}

	schemaFiles, err := expandGlobs([]string{"schemas/*.oracle"}, repoRoot)
	if err != nil {
		return err
	}

	if len(schemaFiles) == 0 {
		printError("no schema files found")
		return errors.New("no schema files found")
	}

	printSchemaCount(len(schemaFiles))

	// Check formatting
	unformatted := 0
	for _, f := range schemaFiles {
		source, err := os.ReadFile(f)
		if err != nil {
			printError(fmt.Sprintf("failed to read %s: %v", f, err))
			return err
		}
		result, err := formatter.Format(string(source))
		if err != nil {
			printError(fmt.Sprintf("failed to format %s: %v", f, err))
			return err
		}
		if result != string(source) {
			printInfo(fmt.Sprintf("needs formatting: %s", f))
			unformatted++
		}
	}
	if unformatted > 0 {
		printError(fmt.Sprintf("%d file(s) need formatting (run 'oracle fmt')", unformatted))
		return errors.New("formatting check failed")
	}

	loader := analyzer.NewStandardFileLoader(repoRoot)
	table, diag := analyzer.Analyze(ctx, schemaFiles, loader)
	if diag != nil && !diag.Empty() {
		printDiagnostics(diag.String())
	}

	if diag != nil && !diag.Ok() {
		printError(fmt.Sprintf("validation failed with %d error(s)", len(diag.Errors())))
		return errors.New("validation failed")
	}

	if table != nil {
		printValidationPassed(len(table.StructTypes()), len(table.EnumTypes()))
	}
	return nil
}
