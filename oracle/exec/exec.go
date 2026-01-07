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
	if err := eslintCmd.Run(); err != nil {
		return errors.Wrapf(err, "failed to run %s", command[0])
	}
	return nil
}

// PostWriter provides a reusable PostWrite implementation for plugins.
// It groups files by project directory and runs commands in order.
type PostWriter struct {
	// ConfigFile is the file that marks a project directory (e.g., "package.json", "pyproject.toml").
	ConfigFile string
	// Commands are the commands to run in order on each group of files.
	Commands [][]string
}

// PostWrite groups files by their project directory and runs all configured commands.
func (w *PostWriter) PostWrite(files []string) error {
	if len(files) == 0 {
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
