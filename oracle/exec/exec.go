// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package exec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/synnaxlabs/oracle/output"
	"github.com/synnaxlabs/x/errors"
)

// OnFiles runs a command on a list of files in a specific directory.
func OnFiles(
	command []string,
	files []string,
	dir string,
) error {
	output.PostWriteStep(strings.Join(command, " "), len(files), "running")
	eslintArgs := append(command[1:], files...)
	eslintCmd := exec.Command(command[0], eslintArgs...)
	eslintCmd.Dir = dir
	cmdOutput, err := eslintCmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to run %s: %s", command[0], string(cmdOutput))
	}
	return nil
}

// PostWriter provides a reusable PostWrite implementation for plugins.
// It groups files by project directory and runs commands in order.
type PostWriter struct {
	// ConfigFile is the file that marks a project directory (e.g., "package.json", "pyproject.toml").
	// If empty, files are run from their containing directory without grouping.
	ConfigFile string
	// Extensions filters files to only those matching the given extensions (e.g., []string{".go"}).
	// If empty, all files are processed.
	Extensions []string
	// Commands are the commands to run in order on each group of files.
	Commands [][]string
}

// PostWrite groups files by their project directory and runs all configured commands.
func (w *PostWriter) PostWrite(files []string) error {
	if len(files) == 0 {
		return nil
	}
	// Filter by extensions if specified
	if len(w.Extensions) > 0 {
		filtered := make([]string, 0, len(files))
		for _, f := range files {
			for _, ext := range w.Extensions {
				if strings.HasSuffix(f, ext) {
					filtered = append(filtered, f)
					break
				}
			}
		}
		files = filtered
		if len(files) == 0 {
			return nil
		}
	}
	// If no config file specified, run from current directory
	if w.ConfigFile == "" {
		for _, cmd := range w.Commands {
			if err := OnFiles(cmd, files, ""); err != nil {
				return err
			}
		}
		return nil
	}
	byProject := GroupByConfigDir(files, w.ConfigFile)
	for projDir, projFiles := range byProject {
		for _, cmd := range w.Commands {
			if err := OnFiles(cmd, projFiles, projDir); err != nil {
				return err
			}
		}
	}
	return nil
}

// GroupByConfigDir groups files by the nearest directory containing the config file.
func GroupByConfigDir(files []string, configFile string) map[string][]string {
	result := make(map[string][]string)
	for _, file := range files {
		if dir := FindConfigDir(file, configFile); dir != "" {
			result[dir] = append(result[dir], file)
		}
	}
	return result
}

// FindConfigDir finds the nearest ancestor directory containing the config file.
func FindConfigDir(file, configFile string) string {
	dir := filepath.Dir(file)
	for dir != "/" && dir != "." {
		if _, err := os.Stat(filepath.Join(dir, configFile)); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return ""
}
