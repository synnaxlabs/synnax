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
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/plugin"
	gomigrate "github.com/synnaxlabs/oracle/plugin/go/migrate"
	"github.com/synnaxlabs/oracle/snapshot"
	"github.com/synnaxlabs/x/errors"
)

func newMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Generate migration files for schema changes and take a schema snapshot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := runMigrate(cmd); err != nil {
				printError(err.Error())
				return err
			}
			return nil
		},
	}
}

func runMigrate(cmd *cobra.Command) error {
	ctx := cmd.Context()
	verbose := viper.GetBool(verboseFlag)
	printBanner()
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "migrate must be run within a git repository")
	}

	normalizedFiles, err := pipeline.DiscoverSchemas(repoRoot)
	if err != nil {
		return err
	}
	if len(normalizedFiles) == 0 {
		return errors.New("no schema files found")
	}

	printSchemaCount(len(normalizedFiles))

	// Build a registry with only the migrate plugin.
	registry := plugin.NewRegistry()
	_ = registry.Register(gomigrate.New())

	// Load old snapshot if one exists.
	latestVersion, loadSnapshot, err := snapshot.TableLoader(ctx, repoRoot)
	if err != nil {
		return err
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
		LoadSnapshot:    loadSnapshot,
	}

	// If we have a previous snapshot, load it for diffing. A snapshot the
	// current grammar can no longer analyze is historical text, not a
	// baseline — migrate proceeds without diffing rather than failing.
	if latestVersion > 0 {
		oldTable, err := loadSnapshot(latestVersion)
		if err != nil && !errors.Is(err, snapshot.ErrAnalysis) {
			return errors.Wrap(err, "failed to load latest snapshot")
		}
		switch {
		case err != nil:
			printDim(fmt.Sprintf(
				"snapshot v%d no longer parses under the current grammar; "+
					"migration diffing resumes at the next snapshot",
				latestVersion,
			))
		case oldTable != nil:
			req.OldResolutions = oldTable
			req.SnapshotVersion = latestVersion
			if gomigrate.SnapshotPreVersioning(oldTable) {
				printDim(fmt.Sprintf(
					"snapshot v%d predates @go version; migration diffing resumes at the next snapshot",
					latestVersion,
				))
			}
		}
	}

	// Run the migrate plugin.
	resp, err := registry.Get("go/migrate").Generate(req)
	if err != nil {
		return errors.Wrap(err, "migration generation failed")
	}

	formatters, err := format.Default(repoRoot)
	if err != nil {
		return errors.Wrap(err, "build formatter registry")
	}

	written := 0
	var templates []string
	for _, f := range resp.Files {
		fullPath := filepath.Join(repoRoot, f.Path)
		canonical, err := formatters.Format(ctx, f.Content, fullPath)
		if err != nil {
			return err
		}
		if err := writeFileIfChanged(fullPath, canonical); err != nil {
			return errors.Wrapf(err, "failed to write %s", f.Path)
		}
		if strings.HasSuffix(f.Path, "/migrate.go") {
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
	data, err := os.ReadFile(
		filepath.Join(repoRoot, "core", "pkg", "version", "VERSION"),
	)
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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}
