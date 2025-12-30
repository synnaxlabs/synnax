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
	"github.com/synnaxlabs/oracle"
	"github.com/synnaxlabs/oracle/paths"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync generated code with schemas, only writing changed files",
	Long: `Parse .oracle schema files and generate code, but only write files
whose content has actually changed. This is useful for incremental builds
and avoiding unnecessary file modifications.

By default, looks for schemas in schemas/*.oracle and runs all plugins.
Each plugin only generates output for structs that have its domain declared
with an output path.`,
	Example: `  oracle sync
  oracle sync -s "other/*.oracle"
  oracle sync -p ts/types -v`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	configureSyncFlags()
	bindFlags(syncCmd)
}

func configureSyncFlags() {
	syncCmd.Flags().StringSliceP(
		pluginsFlag,
		"p",
		nil,
		"Plugins to run (default: all)",
	)
}

func runSync(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	verbose := viper.GetBool(verboseFlag)

	schemaPatterns := viper.GetStringSlice(schemasFlag)
	if len(schemaPatterns) == 0 {
		schemaPatterns = []string{"schemas/*.oracle"}
	}

	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return fmt.Errorf("oracle must be run within a git repository: %w", err)
	}

	if verbose {
		fmt.Printf("Repository root: %s\n", repoRoot)
	}

	schemaFiles, err := expandGlobs(schemaPatterns, repoRoot)
	if err != nil {
		return err
	}

	if len(schemaFiles) == 0 {
		return fmt.Errorf("no schema files found matching patterns: %v", schemaPatterns)
	}

	normalizedFiles := make([]string, 0, len(schemaFiles))
	for _, f := range schemaFiles {
		relPath, err := paths.Normalize(f, repoRoot)
		if err != nil {
			return fmt.Errorf("failed to normalize schema path %q: %w", f, err)
		}
		normalizedFiles = append(normalizedFiles, relPath)
	}

	if verbose {
		fmt.Printf("Found %d schema file(s):\n", len(normalizedFiles))
		for _, f := range normalizedFiles {
			fmt.Printf("  - %s\n", f)
		}
	}

	// Build plugin registry
	pluginNames := viper.GetStringSlice(pluginsFlag)
	registry, err := buildPluginRegistry(pluginNames)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("Using %d plugin(s): %v\n", len(registry.Names()), registry.Names())
	}

	result, diag := oracle.Generate(ctx, normalizedFiles, repoRoot, registry, repoRoot)
	if diag != nil && !diag.Empty() {
		fmt.Fprintln(os.Stderr, diag.String())
	}

	if diag != nil && diag.HasErrors() {
		return fmt.Errorf("generation failed with %d error(s)", len(diag.Errors()))
	}

	if result != nil {
		syncResult, err := result.SyncFiles(repoRoot)
		if err != nil {
			return fmt.Errorf("failed to sync files: %w", err)
		}

		if verbose {
			if len(syncResult.Written) > 0 {
				fmt.Println("Written files:")
				for plugin, files := range syncResult.ByPlugin {
					for _, f := range files {
						fmt.Printf("  [%s] %s\n", plugin, f)
					}
				}
			}
			if len(syncResult.Unchanged) > 0 {
				fmt.Printf("Unchanged: %d file(s)\n", len(syncResult.Unchanged))
			}
		}

		if len(syncResult.Written) == 0 {
			fmt.Println("Already up to date")
		} else {
			fmt.Printf("Synced %d file(s)\n", len(syncResult.Written))
		}
	}

	return nil
}
