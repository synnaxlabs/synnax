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

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/x/errors"
)

var fmtCmd = &cobra.Command{
	Use:   "fmt [files...]",
	Short: "Format Oracle schema files",
	Long: `Format Oracle schema files according to the canonical style.

If no files are specified, formats all .oracle files in the schemas/ directory.

Examples:
  oracle fmt                           # Format all schema files
  oracle fmt schemas/rack.oracle       # Format a specific file
  oracle fmt schemas/*.oracle          # Format matching files`,
	RunE: runFmt,
}

var (
	fmtCheck bool // Check mode - don't write, exit 1 if changes needed
	fmtDiff  bool // Show diff instead of writing
)

func init() {
	fmtCmd.Flags().BoolVarP(&fmtCheck, "check", "c", false,
		"Check if files are formatted (exit 1 if not)")
	fmtCmd.Flags().BoolVarP(&fmtDiff, "diff", "d", false,
		"Show diff instead of writing files")
	rootCmd.AddCommand(fmtCmd)
}

func runFmt(cmd *cobra.Command, args []string) error {
	printBanner()

	repoRoot, err := paths.RepoRoot()
	if err != nil {
		printError("must be run within a git repository")
		return err
	}

	// Default to schemas/*.oracle if no args
	patterns := args
	if len(patterns) == 0 {
		patterns = []string{"schemas/*.oracle"}
	}

	files, err := expandGlobs(patterns, repoRoot)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		printError("no schema files found")
		return errors.New("no schema files found")
	}

	printSchemaCount(len(files))

	var (
		formatted int
		unchanged int
		failed    int
	)

	for _, file := range files {
		result, err := formatFile(file)
		if err != nil {
			printError(fmt.Sprintf("failed to format %s: %v", file, err))
			failed++
			continue
		}

		switch result {
		case formatResultChanged:
			formatted++
		case formatResultUnchanged:
			unchanged++
		}
	}

	if failed > 0 {
		printError(fmt.Sprintf("%d file(s) failed to format", failed))
		return errors.New("formatting failed")
	}

	if fmtCheck && formatted > 0 {
		printError(fmt.Sprintf("%d file(s) need formatting", formatted))
		return errors.New("files need formatting")
	}

	printFormatResult(formatted, unchanged)
	return nil
}

type formatResult int

const (
	formatResultUnchanged formatResult = iota
	formatResultChanged
)

func formatFile(path string) (formatResult, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return formatResultUnchanged, err
	}

	formatted, err := formatter.Format(string(source))
	if err != nil {
		return formatResultUnchanged, err
	}

	if formatted == string(source) {
		return formatResultUnchanged, nil
	}

	if fmtCheck {
		// In check mode, just report that changes are needed
		printInfo(fmt.Sprintf("needs formatting: %s", path))
		return formatResultChanged, nil
	}

	if fmtDiff {
		// Show what would change
		printInfo(fmt.Sprintf("would format: %s", path))
		return formatResultChanged, nil
	}

	// Write the formatted file
	if err := os.WriteFile(path, []byte(formatted), 0644); err != nil {
		return formatResultUnchanged, err
	}

	printFileFormatted(path)
	return formatResultChanged, nil
}

func printFileFormatted(path string) {
	f := fileStyle.Render(path)
	fmt.Printf("  %s %s %s\n", dimStyle.Render(symbolFile), successStyle.Render("formatted"), f)
}

func printFormatResult(formatted, unchanged int) {
	if formatted == 0 {
		fmt.Printf("%s %s\n", dimStyle.Render(symbolDot), dimStyle.Render("all files already formatted"))
		return
	}
	f := countStyle.Render(fmt.Sprintf("%d", formatted))
	word := "file"
	if formatted != 1 {
		word = "files"
	}
	msg := fmt.Sprintf("%s %s formatted", f, word)
	if unchanged > 0 {
		msg += dimStyle.Render(fmt.Sprintf(" (%d unchanged)", unchanged))
	}
	printSuccess(msg)
}
