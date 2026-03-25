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
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	gomigrate "github.com/synnaxlabs/oracle/plugin/go/migrate"
	"github.com/synnaxlabs/oracle/snapshot"
	"github.com/synnaxlabs/x/errors"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Generate migration files for schema changes and take a schema snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runMigrate(cmd); err != nil {
			printError(err.Error())
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command) error {
	ctx := cmd.Context()
	verbose := viper.GetBool(verboseFlag)
	printBanner()
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "migrate must be run within a git repository")
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

	// Build a registry with only the migrate plugin.
	registry := plugin.NewRegistry()
	_ = registry.Register(gomigrate.New())

	// Load old snapshot if one exists.
	schemasDir := filepath.Join(repoRoot, "schemas")
	snapshotsDir := filepath.Join(schemasDir, ".snapshots")
	latestVersion, err := snapshot.LatestVersion(snapshotsDir)
	if err != nil {
		return errors.Wrap(err, "failed to read snapshot version")
	}

	// Analyze current schemas.
	loader := analyzer.NewStandardFileLoader(repoRoot)
	table, diag := analyzer.Analyze(ctx, normalizedFiles, loader)
	if diag != nil {
		printDiagnostics(diag.String())
		if !diag.Ok() {
			return errors.New("schema analysis failed")
		}
	}

	// Build the plugin request.
	req := &plugin.Request{
		Resolutions: table,
		RepoRoot:    repoRoot,
	}

	// If we have a previous snapshot, load it for diffing.
	if latestVersion > 0 {
		snapshotDir := filepath.Join(snapshotsDir, fmt.Sprintf("v%d", latestVersion))
		oldFiles, err := snapshot.Files(snapshotDir)
		if err != nil {
			return errors.Wrap(err, "failed to read snapshot files")
		}
		if len(oldFiles) > 0 {
			// Use a snapshot-specific loader that redirects imports to the
			// snapshot directory, preventing conflicts with current schemas.
			snapshotLoader := snapshot.NewFileLoader(snapshotDir, repoRoot)
			oldNormalized := make([]string, 0, len(oldFiles))
			for _, f := range oldFiles {
				// Convert absolute snapshot path to schemas/... import path
				// so the analyzer derives the same namespaces as the original.
				rel, err := filepath.Rel(snapshotDir, f)
				if err != nil {
					return errors.Wrapf(err, "failed to compute relative path for %s", f)
				}
				importPath := "schemas/" + strings.TrimSuffix(rel, ".oracle")
				oldNormalized = append(oldNormalized, importPath)
			}
			oldTable, oldDiag := analyzer.Analyze(ctx, oldNormalized, snapshotLoader)
			if oldDiag != nil && !oldDiag.Ok() {
				printDiagnostics(oldDiag.String())
			}
			if oldTable != nil {
				req.OldResolutions = oldTable
				req.SnapshotVersion = latestVersion
			}
		}
	}

	// Run the migrate plugin.
	resp, err := registry.Get("go/migrate").Generate(req)
	if err != nil {
		return errors.Wrap(err, "migration generation failed")
	}

	// Write generated files.
	written := 0
	for _, f := range resp.Files {
		fullPath := filepath.Join(repoRoot, f.Path)
		if err := writeFileIfChanged(fullPath, f.Content); err != nil {
			return errors.Wrapf(err, "failed to write %s", f.Path)
		}
		if verbose {
			printFileWritten("go/migrate", f.Path)
		}
		written++
	}

	// Run post-write hooks (gofmt).
	if written > 0 {
		absPaths := make([]string, 0, len(resp.Files))
		for _, f := range resp.Files {
			absPaths = append(absPaths, filepath.Join(repoRoot, f.Path))
		}
		p := registry.Get("go/migrate")
		if pw, ok := p.(plugin.PostWriter); ok {
			if err := pw.PostWrite(absPaths); err != nil {
				printDim(fmt.Sprintf("post-write hook failed: %v", err))
			}
		}
		if err := updateLicenseHeaders(repoRoot, absPaths); err != nil {
			return errors.Wrapf(err, "failed to update license headers")
		}
	}

	printSyncedCount(written, len(resp.Files)-written)

	// Take a new snapshot.
	nextVersion := latestVersion + 1
	if err := snapshot.Create(schemasDir, snapshotsDir, nextVersion); err != nil {
		return errors.Wrap(err, "failed to create schema snapshot")
	}
	printDim(fmt.Sprintf("snapshot v%d created", nextVersion))

	// Run oracle sync to update types/codecs.
	printDim("running sync...")
	if err := runSync(cmd); err != nil {
		return errors.Wrap(err, "sync failed after migration generation")
	}

	return nil
}

func writeFileIfChanged(path string, content []byte) error {
	existing, err := os.ReadFile(path)
	if err == nil && string(existing) == string(content) {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}
