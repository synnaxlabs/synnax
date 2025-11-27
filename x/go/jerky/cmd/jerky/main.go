// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package main provides the jerky CLI for code generation.
//
// Usage:
//
//	//go:generate jerky
//	type MyStruct struct { ... }
//
// Then run: go generate ./...
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/synnaxlabs/x/jerky"
	"github.com/synnaxlabs/x/jerky/deps"
	"github.com/synnaxlabs/x/jerky/generate"
	"github.com/synnaxlabs/x/jerky/parse"
)

var (
	version = flag.Bool("version", false, "Print version and exit")
	help    = flag.Bool("help", false, "Print help and exit")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("jerky version %s\n", jerky.Version)
		os.Exit(0)
	}

	if *help {
		printUsage()
		os.Exit(0)
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "jerky: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`jerky - Generate protobuf definitions and translation functions from Go structs

Usage:
  jerky [flags]

Flags:
  -help       Print this help message
  -version    Print version and exit

Environment Variables:
  GOFILE      Source file (set automatically by go generate)

Example:
  //go:generate jerky
  type MyStruct struct {
      Key      uuid.UUID ` + "`json:\"key\"`" + `
      Name     string    ` + "`json:\"name\"`" + `
      Duration int       ` + "`json:\"duration\"`" + `
  }

Then run: go generate ./...`)
}

func run() error {
	// Get source file from GOFILE env var (set by go generate)
	sourceFile := os.Getenv("GOFILE")
	if sourceFile == "" {
		return fmt.Errorf("GOFILE environment variable not set (run via go generate)")
	}

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	sourcePath := filepath.Join(wd, sourceFile)

	fmt.Printf("jerky: parsing %s\n", sourceFile)

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
		return fmt.Errorf("failed to parse %s: %w", sourceFile, err)
	}

	if len(structs) == 0 {
		return fmt.Errorf("no jerky-annotated structs found in %s", sourceFile)
	}

	// Generate code for each struct with dependency registry
	gen, err := generate.NewGeneratorWithDeps(wd, nil, depRegistry)
	if err != nil {
		return fmt.Errorf("failed to create generator: %w", err)
	}

	for _, s := range structs {
		fmt.Printf("jerky: generating code for %s\n", s.Name)
		if err := gen.Generate(s); err != nil {
			return fmt.Errorf("failed to generate code for %s: %w", s.Name, err)
		}
	}

	// Run buf generate for proto compilation
	typesDir := filepath.Join(wd, "types")
	if err := runBufGenerate(typesDir); err != nil {
		fmt.Printf("jerky: warning: buf generate failed: %v\n", err)
		fmt.Println("jerky: please run 'buf generate' manually in the types directory")
	}

	fmt.Printf("jerky: generated code for %d struct(s)\n", len(structs))
	return nil
}

func runBufGenerate(typesDir string) error {
	// Check if buf is available
	_, err := exec.LookPath("buf")
	if err != nil {
		return fmt.Errorf("buf not found in PATH")
	}

	// Run buf generate from the types directory
	cmd := exec.Command("buf", "generate")
	cmd.Dir = typesDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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
