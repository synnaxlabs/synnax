// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/text"
)

var (
	outputFile string
	verbose    bool
)

var compileCmd = &cobra.Command{
	Use:   "compile <input.arc>",
	Short: "Compile Arc source code to WebAssembly",
	Long: `Compile Arc source code to WebAssembly bytecode.

The compiler performs these steps:
  1. Parse: Convert source text to Abstract Syntax Tree (AST)
  2. Analyze: Perform semantic analysis and build Intermediate Representation (IR)
  3. Compile: Generate WebAssembly bytecode from IR

Output is in JSON format containing the module with IR and WASM bytecode.`,
	Args: cobra.ExactArgs(1),
	RunE: runCompile,
}

func init() {
	compileCmd.Flags().StringVarP(
		&outputFile,
		"output",
		"o",
		"",
		"Output file (default: stdout)",
	)
	compileCmd.Flags().BoolVarP(
		&verbose,
		"verbose",
		"v",
		false,
		"Verbose diagnostic output",
	)
}

func runCompile(cmd *cobra.Command, args []string) error {
	inputPath := args[0]

	// Step 1: Read source file
	sourceBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}
	source := string(sourceBytes)

	if verbose {
		fmt.Fprintf(os.Stderr, "Compiling %s...\n", inputPath)
	}

	// Step 2: Parse
	if verbose {
		fmt.Fprintln(os.Stderr, "Parsing...")
	}
	parsed, diag := text.Parse(text.Text{Raw: source})
	if diag != nil && !diag.Ok() {
		return formatDiagnostics(inputPath, source, diag)
	}

	// Step 3: Analyze
	if verbose {
		fmt.Fprintln(os.Stderr, "Analyzing...")
	}
	ctx := context.Background()
	ir, diag := text.Analyze(ctx, parsed, nil)
	if diag != nil && !diag.Ok() {
		return formatDiagnostics(inputPath, source, diag)
	}

	// Step 4: Compile
	if verbose {
		fmt.Fprintln(os.Stderr, "Compiling to WebAssembly...")
	}
	mod, err := text.Compile(ctx, ir)
	if err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}

	// Step 5: Marshal to JSON
	outputJSON, err := marshalModule(mod)
	if err != nil {
		return fmt.Errorf("failed to serialize output: %w", err)
	}

	// Step 6: Write output
	if outputFile == "" {
		fmt.Println(string(outputJSON))
	} else {
		if err := os.WriteFile(outputFile, outputJSON, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "Output written to %s\n", outputFile)
		}
	}

	if verbose {
		fmt.Fprintln(os.Stderr, "Compilation successful!")
	}

	return nil
}

// marshalModule marshals a module to JSON with proper indentation
func marshalModule(mod module.Module) ([]byte, error) {
	return json.MarshalIndent(mod, "", "  ")
}

// formatDiagnostics formats and prints diagnostic messages with source context
func formatDiagnostics(
	filePath string,
	source string,
	diag *diagnostics.Diagnostics,
) error {
	fmt.Fprintf(os.Stderr, "Compilation failed for %s:\n\n", filePath)

	lines := splitLines(source)

	for _, d := range *diag {
		// Print location and severity
		fmt.Fprintf(
			os.Stderr,
			"%s:%d:%d: %s: %s\n",
			filepath.Base(filePath),
			d.Line,
			d.Column+1, // Convert 0-based to 1-based for display
			d.Severity.String(),
			d.Message,
		)

		// Print source line context if available
		if d.Line > 0 && d.Line <= len(lines) {
			sourceLine := lines[d.Line-1] // Line numbers are 1-based
			fmt.Fprintf(os.Stderr, "  %s\n", sourceLine)

			// Print caret pointer
			if d.Column >= 0 {
				pointer := fmt.Sprintf("  %*s^", d.Column, "")
				fmt.Fprintln(os.Stderr, pointer)
			}
		}

		fmt.Fprintln(os.Stderr)
	}

	return fmt.Errorf("compilation failed with %d error(s)", len(*diag))
}

// splitLines splits source into lines preserving original line endings
func splitLines(source string) []string {
	var lines []string
	var currentLine []rune

	for _, r := range source {
		if r == '\n' {
			lines = append(lines, string(currentLine))
			currentLine = nil
		} else if r != '\r' { // Skip \r to handle both \n and \r\n
			currentLine = append(currentLine, r)
		}
	}

	// Add final line if it doesn't end with newline
	if len(currentLine) > 0 {
		lines = append(lines, string(currentLine))
	}

	return lines
}
