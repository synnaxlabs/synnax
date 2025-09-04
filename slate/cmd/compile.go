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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/slate/analyzer"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/module"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/x/errors"
)

func init() {
	compileCmd.Flags().StringP("output", "o", "", "Output file path")
	viper.BindPFlag("compile.output", compileCmd.Flags().Lookup("output"))
}

var compileCmd = &cobra.Command{
	Use:   "compile [file]",
	Short: "Compile a Slate program",
	Long:  `Compile a Slate source file to WebAssembly and metadata`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath := args[0]
		outputPath := viper.GetString("compile.output")
		outputJSON := viper.GetBool("compile.json")
		source, err := os.ReadFile(inputPath)
		if err != nil {
			return errors.Newf("failed to read source file: %s", err)
		}
		prog, err := parser.Parse(string(source))
		if err != nil {
			return errors.Newf("parse error: %s", err)
		}
		analysis := analyzer.Analyze(prog, analyzer.Options{})
		if len(analysis.Diagnostics) > 0 {
			return errors.Newf("analysis failed %s", analysis.String())
		}
		wasmBytes, err := compiler.Compile(compiler.Config{
			Program:  prog,
			Analysis: &analysis,
		})
		if err != nil {
			return errors.Newf("compilation error: %s", err)
		}
		mod, err := module.Assemble(prog, analysis, wasmBytes)
		if err != nil {
			return errors.Newf("module assembly error: %s", err)
		}
		if outputPath == "" {
			base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
			outputPath = base + ".module.json"
		}
		// Write output
		if outputJSON {
			// Write as JSON with WASM embedded
			jsonData, err := json.MarshalIndent(mod, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal module: %w", err)
			}
			if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
			fmt.Printf("Compiled module written to %s\n", outputPath)
		} else {
			// Write raw WASM binary
			if err := os.WriteFile(outputPath, wasmBytes, 0644); err != nil {
				return fmt.Errorf("failed to write WASM: %w", err)
			}
			fmt.Printf("WASM binary written to %s\n", outputPath)

			// Also write metadata file
			metaPath := strings.TrimSuffix(outputPath, ".wasm") + ".meta.json"
			metaData, err := json.MarshalIndent(struct {
				Tasks     []module.Task     `json:"tasks"`
				Functions []module.Function `json:"functions"`
				Nodes     []module.Node     `json:"nodes"`
				Edges     []module.Edge     `json:"edges"`
			}{
				Tasks:     mod.Tasks,
				Functions: mod.Functions,
				Nodes:     mod.Nodes,
				Edges:     mod.Edges,
			}, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
			if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
				return fmt.Errorf("failed to write metadata: %w", err)
			}
			fmt.Printf("Metadata written to %s\n", metaPath)
		}

		// Print summary
		fmt.Printf("\nCompilation successful:\n")
		fmt.Printf("  %d tasks\n", len(mod.Tasks))
		fmt.Printf("  %d functions\n", len(mod.Functions))
		fmt.Printf("  %d nodes\n", len(mod.Nodes))
		fmt.Printf("  %d edges\n", len(mod.Edges))
		fmt.Printf("  WASM size: %d bytes\n", len(wasmBytes))

		return nil
	},
}
