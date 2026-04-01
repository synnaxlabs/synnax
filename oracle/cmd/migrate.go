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
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	gomigrate "github.com/synnaxlabs/oracle/plugin/go/migrate"
	"github.com/synnaxlabs/oracle/snapshot"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
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

	// Read core version for migration numbering.
	coreVersion, err := readCoreVersion(repoRoot)
	if err != nil {
		return errors.Wrap(err, "failed to read core version")
	}

	// Build the plugin request.
	req := &plugin.Request{
		Resolutions:     table,
		RepoRoot:        repoRoot,
		SnapshotVersion: coreVersion,
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
			if oldDiag != nil {
				printDiagnostics(oldDiag.String())
				if !oldDiag.Ok() {
					return errors.New("failed to analyze old schema snapshot")
				}
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

	// Detect first-time migrate.gen.go files before writing.
	newMigrateGens := make(set.Set[string])
	for _, f := range resp.Files {
		if strings.HasSuffix(f.Path, "/migrate.gen.go") {
			fullPath := filepath.Join(repoRoot, f.Path)
			if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
				newMigrateGens.Add(f.Path)
			}
		}
	}

	// Write generated files and track what was generated.
	written := 0
	var templates []string
	for _, f := range resp.Files {
		fullPath := filepath.Join(repoRoot, f.Path)
		if err := writeFileIfChanged(fullPath, f.Content); err != nil {
			return errors.Wrapf(err, "failed to write %s", f.Path)
		}
		if strings.HasSuffix(f.Path, "/migrate.go") && !strings.Contains(f.Path, "/migrations/") {
			templates = append(templates, f.Path)
		}
		if verbose {
			printFileWritten("go/migrate", f.Path)
		}
		written++
	}
	if len(templates) > 0 {
		for _, t := range templates {
			printDim(fmt.Sprintf("  ✏️  %s ← edit this", t))
		}
	}
	for path := range newMigrateGens {
		pkg := filepath.Base(filepath.Dir(path))
		printDim(fmt.Sprintf("  🔌 Wire %sMigrations(cfg.Codec) into your gorp.OpenTable call", strings.ToUpper(pkg[:1])+pkg[1:]))
	}

	// Delete files that were retargeted and moved.
	for _, d := range resp.Deletions {
		fullPath := filepath.Join(repoRoot, d)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return errors.Wrapf(err, "failed to delete retargeted file %s", d)
		}
		if verbose {
			printDim(fmt.Sprintf("  moved %s", d))
		}
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

	// Run oracle sync to update types/codecs.
	printDim("running sync...")
	if err := runSync(cmd); err != nil {
		return errors.Wrap(err, "sync failed after migration generation")
	}

	return nil
}

// readCoreVersion reads core/pkg/version/VERSION and returns the migration
// version number (major*1000 + minor). For "0.53.4" this returns 53.
func readCoreVersion(repoRoot string) (int, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "core", "pkg", "version", "VERSION"))
	if err != nil {
		return 0, errors.Wrap(err, "failed to read core VERSION file")
	}
	version := strings.TrimSpace(string(data))
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return 0, errors.Newf("invalid version format: %s", version)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, errors.Wrapf(err, "invalid major version: %s", parts[0])
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, errors.Wrapf(err, "invalid minor version: %s", parts[1])
	}
	return major*1000 + minor, nil
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
