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
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	cpptypes "github.com/synnaxlabs/oracle/plugin/cpp/types"
	gopb "github.com/synnaxlabs/oracle/plugin/go/pb"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	pbtypes "github.com/synnaxlabs/oracle/plugin/pb/types"
	pytypes "github.com/synnaxlabs/oracle/plugin/py/types"
	tstypes "github.com/synnaxlabs/oracle/plugin/ts/types"
	"github.com/synnaxlabs/x/errors"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync generated code, only writing changed files",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runSync(cmd); err != nil {
			printError(err.Error())
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command) error {
	ctx := cmd.Context()
	verbose := viper.GetBool(verboseFlag)
	printBanner()
	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return errors.Wrap(err, "sync must be run within a git repository")
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
	printFormattingStart(len(schemaFiles))

	formatted := 0
	for _, f := range schemaFiles {
		source, err := os.ReadFile(f)
		if err != nil {
			return errors.Wrapf(err, "failed to read %s", f)
		}
		result, err := formatter.Format(string(source))
		if err != nil {
			return errors.Wrapf(err, "failed to format %s", err)
		}
		if result != string(source) {
			if err := os.WriteFile(f, []byte(result), 0644); err != nil {
				return errors.Wrapf(err, "failed to write %s", f)
			}
			formatted++
		}
	}
	printFormattingDone(formatted)

	registry := buildPluginRegistry()

	result, diag := oracle.Generate(ctx, normalizedFiles, repoRoot, registry)
	if diag != nil {
		printDiagnostics(diag.String())
		if diag.HasErrors() {
			return errors.New("generation failed")
		}
	}

	syncResult, err := result.SyncFiles(repoRoot)
	if err != nil {
		return errors.Wrap(err, "failed to sync files")
	}
	if len(syncResult.Written) > 0 {
		absPaths := make([]string, len(syncResult.Written))
		for i, f := range syncResult.Written {
			absPaths[i] = filepath.Join(repoRoot, f)
		}
		if err = oracle.UpdateLicenseHeaders(repoRoot, absPaths); err != nil {
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
	// Update copyright headers on protobuf-generated files
	if _, hasPB := syncResult.ByPlugin["pb/types"]; hasPB {
		if err = oracle.UpdateLicenseHeaders(repoRoot, []string{"*.pb.go"}); err != nil {
			return errors.Wrapf(err, "failed to update license headers on .pb.go files")
		}
	}
	printSyncedCount(len(syncResult.Written), len(syncResult.Unchanged))
	return nil
}

// expandGlobs expands glob patterns to actual file paths.
// Results are sorted to ensure deterministic ordering across runs.
func expandGlobs(patterns []string, baseDir string) ([]string, error) {
	var files []string
	for _, pattern := range patterns {
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(baseDir, pattern)
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid glob pattern %q", pattern)
		}

		files = append(files, matches...)
	}
	sort.Strings(files)
	return files, nil
}

func buildPluginRegistry() *plugin.Registry {
	registry := plugin.NewRegistry()
	_ = registry.Register(tstypes.New(tstypes.DefaultOptions()))
	_ = registry.Register(gotypes.New(gotypes.DefaultOptions()))
	_ = registry.Register(pytypes.New(pytypes.DefaultOptions()))
	_ = registry.Register(pbtypes.New(pbtypes.DefaultOptions()))
	_ = registry.Register(cpptypes.New(cpptypes.DefaultOptions()))
	_ = registry.Register(gopb.New(gopb.DefaultOptions()))
	return registry
}
