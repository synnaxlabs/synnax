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
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/oracle"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	cpptypes "github.com/synnaxlabs/oracle/plugin/cpp/types"
	goapi "github.com/synnaxlabs/oracle/plugin/go/api"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	pbtypes "github.com/synnaxlabs/oracle/plugin/pb/types"
	pytypes "github.com/synnaxlabs/oracle/plugin/py/types"
	tstypes "github.com/synnaxlabs/oracle/plugin/ts/types"
	"github.com/synnaxlabs/x/errors"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Parse schemas and generate code",
	RunE:  runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
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

	registry := buildPluginRegistry()

	result, diag := oracle.Generate(ctx, normalizedFiles, repoRoot, registry, repoRoot)
	if diag != nil && !diag.Empty() {
		printDiagnostics(diag.String())
	}

	if diag != nil && diag.HasErrors() {
		printError(fmt.Sprintf("generation failed with %d error(s)", len(diag.Errors())))
		return errors.New("generation failed")
	}

	if result != nil {
		writtenFiles, err := result.WriteFiles(repoRoot)
		if err != nil {
			printError("failed to write files")
			return err
		}

		// Update license headers before running post-write hooks (e.g., eslint)
		if err := oracle.UpdateLicenseHeaders(repoRoot, writtenFiles); err != nil {
			printDim(fmt.Sprintf("license header update failed: %v", err))
		}

		for pluginName, files := range result.Files {
			p := registry.Get(pluginName)
			if pw, ok := p.(plugin.PostWriter); ok {
				absPaths := make([]string, len(files))
				for i, f := range files {
					absPaths[i] = filepath.Join(repoRoot, f.Path)
				}
				if err := pw.PostWrite(absPaths); err != nil {
					printDim(fmt.Sprintf("post-write hook for %s failed: %v", pluginName, err))
				}
			}
		}

		if verbose {
			for pluginName, files := range result.Files {
				for _, f := range files {
					printFileWritten(pluginName, f.Path)
				}
			}
		}

		printGeneratedCount(countGeneratedFiles(result))
	}

	return nil
}

// expandGlobs expands glob patterns to actual file paths.
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

	return files, nil
}

func buildPluginRegistry() *plugin.Registry {
	registry := plugin.NewRegistry()
	_ = registry.Register(tstypes.New(tstypes.DefaultOptions()))
	_ = registry.Register(gotypes.New(gotypes.DefaultOptions()))
	_ = registry.Register(pytypes.New(pytypes.DefaultOptions()))
	_ = registry.Register(pbtypes.New(pbtypes.DefaultOptions()))
	_ = registry.Register(cpptypes.New(cpptypes.DefaultOptions()))
	_ = registry.Register(goapi.New(goapi.DefaultOptions()))
	return registry
}

func countGeneratedFiles(result *oracle.GenerateResult) int {
	count := 0
	for _, files := range result.Files {
		count += len(files)
	}
	return count
}
