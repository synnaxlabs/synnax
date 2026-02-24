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
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	gomarshal "github.com/synnaxlabs/oracle/plugin/go/marshal"
	gomigrate "github.com/synnaxlabs/oracle/plugin/go/migrate"
	gopb "github.com/synnaxlabs/oracle/plugin/go/pb"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	pbtypes "github.com/synnaxlabs/oracle/plugin/pb/types"
	"github.com/synnaxlabs/oracle/snapshot"
	"github.com/synnaxlabs/x/errors"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migration management commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var migrateGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate migration files for schema changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runMigrateGenerate(cmd); err != nil {
			printError(err.Error())
			return err
		}
		return nil
	},
}

var migrateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check that schemas match the latest snapshot",
	Long: `Compares current .oracle schema files against the latest snapshot.

If any schema file has changed since the last snapshot and no new migration
was generated, exits with code 1 and prints an actionable error message.

This command is intended for CI pipelines to enforce that every schema change
is accompanied by a migration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runMigrateCheck(); err != nil {
			printError(err.Error())
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateGenerateCmd)
	migrateCmd.AddCommand(migrateCheckCmd)
}

func runMigrateGenerate(cmd *cobra.Command) error {
	ctx := cmd.Context()
	verbose := viper.GetBool(verboseFlag)
	printBanner()
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "migrate generate must be run within a git repository")
	}

	schemaFiles, err := expandGlobs([]string{"schemas/*.oracle"}, repoRoot)
	if err != nil {
		return err
	}

	if len(schemaFiles) == 0 {
		return errors.New("no schema files found")
	}

	normalizedFiles := make([]string, 0, len(schemaFiles))
	for _, f := range schemaFiles {
		relPath, err := paths.Normalize(f, repoRoot)
		if err != nil {
			return errors.Wrapf(err, "failed to normalize schema path %q", f)
		}
		normalizedFiles = append(normalizedFiles, relPath)
	}

	printSchemaCount(len(normalizedFiles))

	registry := buildMigrateRegistry()

	result, diag := generate(ctx, normalizedFiles, repoRoot, registry)
	if diag != nil {
		printDiagnostics(diag.String())
		if !diag.Ok() {
			return errors.New("generation failed")
		}
	}

	// Only write files from the go/migrate plugin
	filteredResult := filterByPlugin(result, "go/migrate")

	syncResult, err := filteredResult.syncFiles(repoRoot)
	if err != nil {
		return errors.Wrap(err, "failed to sync files")
	}
	if len(syncResult.Written) > 0 {
		absPaths := make([]string, len(syncResult.Written))
		for i, f := range syncResult.Written {
			absPaths[i] = filepath.Join(repoRoot, f)
		}
		if err = updateLicenseHeaders(repoRoot, absPaths); err != nil {
			return errors.Wrapf(err, "failed to update license headers")
		}
	}
	if verbose && len(syncResult.Written) > 0 {
		for pluginName, files := range syncResult.ByPlugin {
			for _, f := range files {
				printFileWritten(pluginName, f)
			}
		}
	}
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
	printSyncedCount(len(syncResult.Written), len(syncResult.Unchanged))

	schemasDir := filepath.Join(repoRoot, "schemas")
	snapshotsDir := filepath.Join(schemasDir, ".snapshots")
	latestVersion, err := snapshot.LatestVersion(snapshotsDir)
	if err != nil {
		return errors.Wrap(err, "failed to read snapshot version")
	}
	nextVersion := latestVersion + 1
	if err := snapshot.Create(schemasDir, snapshotsDir, nextVersion); err != nil {
		return errors.Wrap(err, "failed to create schema snapshot")
	}
	printSnapshotCreated(nextVersion)
	return nil
}

func runMigrateCheck() error {
	printBanner()
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "migrate check must be run within a git repository")
	}
	schemasDir := filepath.Join(repoRoot, "schemas")
	snapshotsDir := filepath.Join(schemasDir, ".snapshots")
	if err := snapshot.Check(schemasDir, snapshotsDir); err != nil {
		return err
	}
	printSuccess("schemas match latest snapshot")
	return nil
}

func buildMigrateRegistry() *plugin.Registry {
	registry := plugin.NewRegistry()
	_ = registry.Register(gotypes.New(gotypes.DefaultOptions()))
	_ = registry.Register(pbtypes.New(pbtypes.DefaultOptions()))
	_ = registry.Register(gopb.New(gopb.DefaultOptions()))
	_ = registry.Register(gomarshal.New(gomarshal.DefaultOptions()))
	_ = registry.Register(gomigrate.New(gomigrate.DefaultOptions()))
	return registry
}

func filterByPlugin(result *generateResult, pluginName string) *generateResult {
	filtered := &generateResult{
		Resolutions: result.Resolutions,
		Files:       make(map[string][]plugin.File),
	}
	if files, ok := result.Files[pluginName]; ok {
		filtered.Files[pluginName] = files
	}
	return filtered
}
