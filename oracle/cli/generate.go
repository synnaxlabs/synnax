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
	"github.com/synnaxlabs/oracle/plugin"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/zod"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Parse schemas and generate code via plugins",
	Long: `Parse .oracle schema files, analyze types and imports, and generate
code using configured plugins.

Output locations are declared per-struct in schema files using struct-level domains.
For example:

    struct Range {
        field key uuid { domain id }

        domain go { output "core/ranger" }
        domain ts { output "console/src/ranger" }
    }`,
	Example: `  oracle generate --schemas "schema/*.oracle"
  oracle generate -s "schema/*.oracle" -p zod
  oracle generate -s "schema/*.oracle" -p go -p ts -v`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	configureGenerateFlags()
	bindFlags(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	verbose := viper.GetBool(verboseFlag)

	// Get schema patterns
	schemaPatterns := viper.GetStringSlice(schemasFlag)
	if len(schemaPatterns) == 0 {
		return fmt.Errorf("no schemas specified (use --schemas)")
	}

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Expand schema globs
	schemaFiles, err := expandGlobs(schemaPatterns, wd)
	if err != nil {
		return err
	}

	if len(schemaFiles) == 0 {
		return fmt.Errorf("no schema files found matching patterns: %v", schemaPatterns)
	}

	if verbose {
		fmt.Printf("Found %d schema file(s):\n", len(schemaFiles))
		for _, f := range schemaFiles {
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

	// Run generation
	result, diag := oracle.Generate(ctx, schemaFiles, wd, registry, wd)

	// Print diagnostics
	if diag != nil && !diag.Empty() {
		fmt.Fprintln(os.Stderr, diag.String())
	}

	if diag != nil && diag.HasErrors() {
		return fmt.Errorf("generation failed with %d error(s)", len(diag.Errors()))
	}

	// Write generated files
	if result != nil {
		if err := result.WriteFiles(wd); err != nil {
			return fmt.Errorf("failed to write generated files: %w", err)
		}

		totalFiles := countGeneratedFiles(result)
		if verbose {
			printGeneratedFiles(result)
		}
		fmt.Printf("Successfully generated %d file(s)\n", totalFiles)
	}

	return nil
}

// expandGlobs expands glob patterns to actual file paths.
func expandGlobs(patterns []string, baseDir string) ([]string, error) {
	var files []string

	for _, pattern := range patterns {
		// Make pattern absolute if relative
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(baseDir, pattern)
		}

		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}

		files = append(files, matches...)
	}

	return files, nil
}

// buildPluginRegistry creates a registry with the specified plugins.
// If no plugins are specified, all available plugins are loaded.
func buildPluginRegistry(pluginNames []string) (*plugin.Registry, error) {
	registry := plugin.NewRegistry()

	// If no plugins specified, load all available
	if len(pluginNames) == 0 {
		pluginNames = []string{"zod"}
	}

	for _, name := range pluginNames {
		p, err := createPlugin(name)
		if err != nil {
			return nil, err
		}
		if err := registry.Register(p); err != nil {
			return nil, err
		}
	}

	return registry, nil
}

// createPlugin instantiates a plugin by name.
func createPlugin(name string) (plugin.Plugin, error) {
	switch name {
	case "zod":
		return zod.New(zod.DefaultOptions()), nil
	case "go/types":
		return gotypes.New(gotypes.DefaultOptions()), nil
	default:
		return nil, fmt.Errorf("unknown plugin: %s (available: zod, go/types)", name)
	}
}

func countGeneratedFiles(result *oracle.GenerateResult) int {
	count := 0
	for _, files := range result.Files {
		count += len(files)
	}
	return count
}

func printGeneratedFiles(result *oracle.GenerateResult) {
	for pluginName, files := range result.Files {
		for _, f := range files {
			fmt.Printf("  [%s] %s\n", pluginName, f.Path)
		}
	}
}
