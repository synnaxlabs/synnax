// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/jerky/deps"
	"github.com/synnaxlabs/x/jerky/generate"
	"github.com/synnaxlabs/x/jerky/parse"
)

var (
	embedded bool
	file     string
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate proto and translation code for jerky-annotated structs",
	Long: `Generate protobuf definitions and Go translation code from structs
annotated with //go:generate jerky.

The source file is determined by:
  1. The --file flag if provided
  2. The GOFILE environment variable (set automatically by go generate)

Examples:
  jerky generate                    # Uses GOFILE env var
  jerky generate --file user.go     # Explicit file
  jerky generate --embedded         # Generate embedded-only type`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().BoolVar(&embedded, "embedded", false,
		"Generate as embedded-only type (no gorp methods or migrator)")
	generateCmd.Flags().StringVar(&file, "file", "",
		"Source file to process (defaults to GOFILE env var)")

	// Also add --embedded flag to root command for backward compatibility
	rootCmd.Flags().BoolVar(&embedded, "embedded", false,
		"Generate as embedded-only type (no gorp methods or migrator)")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Support legacy positional arg for backward compatibility
	// //go:generate jerky embedded
	if len(args) > 0 && args[0] == "embedded" {
		embedded = true
	}

	// Determine source file
	sourceFile := file
	if sourceFile == "" {
		sourceFile = os.Getenv("GOFILE")
	}
	if sourceFile == "" {
		return errors.Newf("source file not specified (use --file flag or run via go generate)")
	}

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return errors.Newf("failed to get working directory: %w", err)
	}

	sourcePath := filepath.Join(wd, sourceFile)

	typeKind := "storage"
	if embedded {
		typeKind = "embedded"
	}
	fmt.Printf("jerky: parsing %s (%s type)\n", sourceFile, typeKind)

	// Discover existing jerky-managed types for dependency tracking
	depRegistry := deps.NewRegistry()
	if moduleRoot := findModuleRoot(wd); moduleRoot != "" {
		if err := depRegistry.DiscoverJerkyTypes(moduleRoot); err != nil {
			fmt.Printf("jerky: warning: failed to discover jerky types: %v\n", err)
		}
	}

	// Parse the source file with dependency awareness
	parser := parse.NewParserWithDeps(depRegistry)
	structs, err := parser.ParseFile(sourcePath)
	if err != nil {
		return errors.Newf("failed to parse %s: %w", sourceFile, err)
	}

	if len(structs) == 0 {
		return errors.Newf("no jerky-annotated structs found in %s", sourceFile)
	}

	// Generate code for each struct with dependency registry
	gen, err := generate.NewGeneratorWithDeps(wd, nil, depRegistry)
	if err != nil {
		return errors.Newf("failed to create generator: %w", err)
	}

	// Track type directories for buf generate
	var typeDirs []string

	for i := range structs {
		structs[i].IsEmbedded = embedded
		fmt.Printf("jerky: generating code for %s\n", structs[i].Name)
		if err := gen.Generate(structs[i]); err != nil {
			return errors.Newf("failed to generate code for %s: %w", structs[i].Name, err)
		}
		// Track the type-specific directory for buf generate
		typePkgName := strings.ToLower(structs[i].Name)
		typeDirs = append(typeDirs, filepath.Join(wd, "types", typePkgName))
	}

	// Run buf generate for each type subdirectory
	repoRoot := findBufRoot(wd)
	if repoRoot == "" {
		repoRoot = wd // Fallback to working directory
	}
	for _, typeDir := range typeDirs {
		if err := runBufGenerate(repoRoot, typeDir); err != nil {
			fmt.Printf("jerky: warning: buf generate failed for %s: %v\n", typeDir, err)
			fmt.Println("jerky: please run 'buf generate --path <type_dir>' from the repo root")
		}
	}

	// Generate types/types.go re-export file
	typesDir := filepath.Join(wd, "types")
	packagePath := structs[0].PackagePath // Use package path from first struct
	if err := gen.GenerateTypesExport(typesDir, packagePath); err != nil {
		fmt.Printf("jerky: warning: failed to generate types export: %v\n", err)
	}

	fmt.Printf("jerky: generated code for %d struct(s)\n", len(structs))
	return nil
}

func runBufGenerate(repoRoot, typesDir string) error {
	// Check if buf is available
	_, err := exec.LookPath("buf")
	if err != nil {
		return errors.Newf("buf not found in PATH")
	}

	// Compute relative path from repo root to types directory
	relPath, err := filepath.Rel(repoRoot, typesDir)
	if err != nil {
		return errors.Newf("failed to compute relative path: %w", err)
	}

	// Run buf generate from the repo root with --path targeting the types directory
	cmd := exec.Command("buf", "generate", "--path", relPath)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// findBufRoot walks up the directory tree to find buf.yaml (the repo root for buf).
func findBufRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "buf.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// findModuleRoot walks up the directory tree to find the go.mod file.
func findModuleRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
