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
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync generated code, only writing changed files",
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	verbose := viper.GetBool(verboseFlag)

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
		return fmt.Errorf("no schema files found")
	}

	normalizedFiles := make([]string, 0, len(schemaFiles))
	for _, f := range schemaFiles {
		relPath, err := paths.Normalize(f, repoRoot)
		if err != nil {
			return fmt.Errorf("failed to normalize schema path %q: %w", f, err)
		}
		normalizedFiles = append(normalizedFiles, relPath)
	}

	printSchemaCount(len(normalizedFiles))

	// Format schema files first
	printFormattingStart(len(schemaFiles))
	formatted := 0
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
			if err := os.WriteFile(f, []byte(result), 0644); err != nil {
				printError(fmt.Sprintf("failed to write %s: %v", f, err))
				return err
			}
			formatted++
		}
	}
	printFormattingDone(formatted)

	registry := buildPluginRegistry()

	result, diag := oracle.Generate(ctx, normalizedFiles, repoRoot, registry, repoRoot)
	if diag != nil && !diag.Empty() {
		printDiagnostics(diag.String())
	}

	if diag != nil && diag.HasErrors() {
		printError(fmt.Sprintf("generation failed with %d error(s)", len(diag.Errors())))
		return fmt.Errorf("generation failed")
	}

	if result != nil {
		syncResult, err := result.SyncFiles(repoRoot)
		if err != nil {
			printError("failed to sync files")
			return err
		}

		// Update license headers before running post-write hooks (e.g., eslint)
		if len(syncResult.Written) > 0 {
			absPaths := make([]string, len(syncResult.Written))
			for i, f := range syncResult.Written {
				absPaths[i] = filepath.Join(repoRoot, f)
			}
			if err := oracle.UpdateLicenseHeaders(repoRoot, absPaths); err != nil {
				printDim(fmt.Sprintf("license header update failed: %v", err))
			}
		}

		if verbose && len(syncResult.Written) > 0 {
			for pluginName, files := range syncResult.ByPlugin {
				for _, f := range files {
					printFileWritten(pluginName, f)
				}
			}
		}

		if len(syncResult.Written) > 0 {
			for pluginName, files := range syncResult.ByPlugin {
				p := registry.Get(pluginName)
				if pw, ok := p.(plugin.PostWriter); ok {
					absPaths := make([]string, len(files))
					for i, f := range files {
						absPaths[i] = filepath.Join(repoRoot, f)
					}
					if err := pw.PostWrite(absPaths); err != nil {
						printDim(fmt.Sprintf("post-write hook for %s failed: %v", pluginName, err))
					}
				}
			}
		}

		printSyncedCount(len(syncResult.Written), len(syncResult.Unchanged))
	}

	return nil
}
