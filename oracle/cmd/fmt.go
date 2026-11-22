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
	"sort"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/pipeline"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/errors"
)

func newFmtCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fmt [files...]",
		Short: "Format Oracle schema files",
		Long: `Format Oracle schema files according to the canonical style.

If no files are specified, formats all .oracle files in the schemas/ directory.

Examples:
  oracle fmt                           # Format all schema files
  oracle fmt schemas/rack.oracle       # Format a specific file
  oracle fmt schemas/*.oracle          # Format matching files`,
	}
	cmd.Flags().BoolVarP(&fmtCheck, "check", "c", false,
		"Check if files are formatted (exit 1 if not)")
	cmd.Flags().BoolVarP(&fmtDiff, "diff", "d", false,
		"Show diff instead of writing files")
	cmd.RunE = runFmt
	return cmd
}

var (
	fmtCheck bool
	fmtDiff  bool
)

func runFmt(cmd *cobra.Command, args []string) error {
	printBanner()

	repoRoot, err := paths.RepoRoot()
	if err != nil {
		return err
	}

	files, err := fmtFiles(args, repoRoot)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("no schema files found")
	}

	printSchemaCount(len(files))
	formatted, unchanged, failed := formatFiles(files, repoRoot)
	if failed > 0 {
		return errors.Newf("%d file(s) failed to format", failed)
	}
	if fmtCheck && formatted > 0 {
		return errors.Newf("%d file(s) need formatting", formatted)
	}

	printFormatResult(formatted, unchanged)
	return nil
}

// fmtFiles returns the absolute paths of the files to format: the files matching args,
// or every schema and version file when args is empty.
func fmtFiles(args []string, repoRoot string) ([]string, error) {
	if len(args) > 0 {
		return expandGlobs(args, repoRoot)
	}
	rel, err := pipeline.DiscoverSchemas(repoRoot)
	if err != nil {
		return nil, err
	}
	chains, err := versions.Discover(repoRoot)
	if err != nil {
		return nil, err
	}
	for _, chain := range chains {
		for _, n := range chain.Numbers {
			rel = append(rel, paths.EnsureOracleExtension(chain.FilePath(n)))
		}
	}
	sort.Strings(rel)
	files := make([]string, len(rel))
	for i, r := range rel {
		files[i] = paths.Resolve(r, repoRoot)
	}
	return files, nil
}

// formatFiles formats each file and counts the outcomes. It prints each failure and
// continues.
func formatFiles(files []string, repoRoot string) (formatted, unchanged, failed int) {
	for _, file := range files {
		result, err := formatFile(file, repoRoot)
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
	return formatted, unchanged, failed
}

type formatResult int

const (
	formatResultUnchanged formatResult = iota
	formatResultChanged
)

func formatFile(path, repoRoot string) (formatResult, error) {
	source, err := paths.ReadFile(path, repoRoot)
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
		printInfo("needs formatting: " + path)
		return formatResultChanged, nil
	}

	if fmtDiff {
		printInfo("would format: " + path)
		return formatResultChanged, nil
	}

	if err := paths.WriteFile(path, repoRoot, []byte(formatted), 0o644); err != nil {
		return formatResultUnchanged, err
	}

	printFileFormatted(path)
	return formatResultChanged, nil
}

func printFileFormatted(path string) {
	f := fileStyle.Render(path)
	fmt.Printf(
		"  %s %s %s\n",
		dimStyle.Render(symbolFile),
		successStyle.Render("formatted"),
		f,
	)
}

func printFormatResult(formatted, unchanged int) {
	if formatted == 0 {
		fmt.Printf(
			"%s %s\n",
			dimStyle.Render(symbolDot),
			dimStyle.Render("all files already formatted"),
		)
		return
	}
	f := countStyle.Render(strconv.Itoa(formatted))
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
