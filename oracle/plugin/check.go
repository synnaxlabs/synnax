// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package plugin

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// StaleFile describes a generated file that is out of date relative to its source schema.
type StaleFile struct {
	// Generated is the path to the generated file
	Generated string
	// Schema is the path to the schema file that is newer
	Schema string
	// GenTime is the modification time of the generated file (zero if missing)
	GenTime time.Time
	// SchemaTime is the modification time of the schema file
	SchemaTime time.Time
}

// StaleError is returned when generated files are out of date with their source schemas.
type StaleError struct {
	// Plugin is the name of the plugin whose output is stale
	Plugin string
	// Files contains details about each stale file
	Files []StaleFile
}

func (e *StaleError) Error() string {
	if len(e.Files) == 0 {
		return fmt.Sprintf("plugin '%s' has stale output", e.Plugin)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("plugin '%s' has %d stale file(s):\n", e.Plugin, len(e.Files)))
	for _, f := range e.Files {
		if f.GenTime.IsZero() {
			sb.WriteString(fmt.Sprintf("  - %s (missing, source: %s)\n", f.Generated, f.Schema))
		} else {
			sb.WriteString(fmt.Sprintf("  - %s (modified: %s, source %s modified: %s)\n",
				f.Generated, f.GenTime.Format(time.RFC3339),
				f.Schema, f.SchemaTime.Format(time.RFC3339)))
		}
	}
	return sb.String()
}

// DependencyStaleError is returned when a plugin's required dependency has stale output.
type DependencyStaleError struct {
	// Plugin is the name of the plugin that was requested
	Plugin string
	// Dependency is the name of the required plugin that is stale
	Dependency string
	// Reason is the underlying error from the dependency's Check()
	Reason error
}

func (e *DependencyStaleError) Error() string {
	return fmt.Sprintf(
		"plugin '%s' requires '%s' to be up-to-date.\n\n  %s\n  Run: oracle generate -p %s",
		e.Plugin, e.Dependency, e.Reason, e.Dependency,
	)
}

// CheckFreshness compares generated files against schema file modification times.
// genFiles maps generated file paths to the schema files they depend on.
// Returns nil if all generated files are fresh (newer than their schemas),
// or a StaleError describing which files are stale.
func CheckFreshness(pluginName string, genFiles map[string][]string) error {
	var staleFiles []StaleFile

	for genPath, schemaPaths := range genFiles {
		genTime := FileModTime(genPath)

		for _, schemaPath := range schemaPaths {
			schemaTime := FileModTime(schemaPath)

			// If generated file is missing or older than schema, it's stale
			if genTime.IsZero() || schemaTime.After(genTime) {
				staleFiles = append(staleFiles, StaleFile{
					Generated:  genPath,
					Schema:     schemaPath,
					GenTime:    genTime,
					SchemaTime: schemaTime,
				})
				break // One stale schema is enough to mark the generated file as stale
			}
		}
	}

	if len(staleFiles) > 0 {
		return &StaleError{
			Plugin: pluginName,
			Files:  staleFiles,
		}
	}
	return nil
}

// FileModTime returns the modification time of a file.
// Returns zero time if the file doesn't exist or can't be accessed.
func FileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
