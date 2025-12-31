// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package oracle provides schema-first code generation for Synnax.
package oracle

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/output"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
)

// Generate analyzes schema files and runs code generation with the given plugins.
func Generate(
	ctx context.Context,
	files []string,
	baseDir string,
	registry *plugin.Registry,
	outputDir string,
) (*GenerateResult, *diagnostics.Diagnostics) {
	loader := analyzer.NewStandardFileLoader(baseDir)
	table, diag := analyzer.Analyze(ctx, files, loader)
	if diag != nil && diag.HasErrors() {
		return nil, diag
	}

	schemaFiles := buildSchemaFiles(files)
	result := &GenerateResult{
		Resolutions: table,
		Files:       make(map[string][]plugin.File),
	}

	for _, p := range registry.All() {
		output.PluginStart(p.Name())
		req := &plugin.Request{
			Schemas:     schemaFiles,
			Resolutions: table,
			Options:     make(map[string]interface{}),
			RepoRoot:    baseDir,
			OutputDir:   outputDir,
		}

		for _, depName := range p.Requires() {
			dep := registry.Get(depName)
			if dep == nil {
				diag.AddErrorf(nil, "", "plugin '%s' requires unknown plugin '%s'", p.Name(), depName)
				continue
			}

			if err := dep.Check(req); err != nil {
				return nil, diagnostics.FromError(&plugin.DependencyStaleError{
					Plugin:     p.Name(),
					Dependency: depName,
					Reason:     err,
				})
			}
		}

		resp, err := p.Generate(req)
		if err != nil {
			diag.AddErrorf(nil, "", "plugin %s failed: %v", p.Name(), err)
			continue
		}

		if resp != nil {
			result.Files[p.Name()] = resp.Files
			output.PluginDone(p.Name(), len(resp.Files))
		}
	}

	if diag != nil && diag.HasErrors() {
		return nil, diag
	}

	return result, diag
}

// GenerateResult contains the results of code generation.
type GenerateResult struct {
	// Resolutions is the resolution table from analysis
	Resolutions *resolution.Table

	// Files maps plugin names to their generated files
	Files map[string][]plugin.File
}

// WriteFiles writes all generated files to disk.
// Returns the absolute paths of all files that were written.
func (r *GenerateResult) WriteFiles(outputDir string) ([]string, error) {
	var written []string
	for _, files := range r.Files {
		for _, f := range files {
			fullPath := filepath.Join(outputDir, f.Path)
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(fullPath, f.Content, 0644); err != nil {
				return nil, err
			}
			written = append(written, fullPath)
		}
	}
	return written, nil
}

// SyncResult contains the results of a sync operation.
type SyncResult struct {
	// Written contains files that were written (new or changed)
	Written []string
	// Unchanged contains files that already had the correct content
	Unchanged []string
	// ByPlugin maps plugin names to their written files
	ByPlugin map[string][]string
}

// SyncFiles writes only files whose content has changed.
// Returns details about what was written vs unchanged.
func (r *GenerateResult) SyncFiles(outputDir string) (*SyncResult, error) {
	result := &SyncResult{
		Written:   make([]string, 0),
		Unchanged: make([]string, 0),
		ByPlugin:  make(map[string][]string),
	}

	for pluginName, files := range r.Files {
		for _, f := range files {
			fullPath := filepath.Join(outputDir, f.Path)

			// Check if file exists and has same content
			existing, err := os.ReadFile(fullPath)
			if err == nil && string(existing) == string(f.Content) {
				result.Unchanged = append(result.Unchanged, f.Path)
				continue
			}

			// Write the file (new or changed)
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(fullPath, f.Content, 0644); err != nil {
				return nil, err
			}

			result.Written = append(result.Written, f.Path)
			result.ByPlugin[pluginName] = append(result.ByPlugin[pluginName], f.Path)
		}
	}

	return result, nil
}

func buildSchemaFiles(files []string) []plugin.SchemaFile {
	result := make([]plugin.SchemaFile, len(files))
	for i, file := range files {
		result[i] = plugin.SchemaFile{FilePath: file}
	}
	return result
}

// UpdateLicenseHeaders runs the copyright update script on the given files.
// files should be absolute paths to the generated files.
// repoRoot is the absolute path to the repository root.
func UpdateLicenseHeaders(repoRoot string, files []string) error {
	if len(files) == 0 {
		return nil
	}

	scriptPath := filepath.Join(repoRoot, "scripts", "update_copyrights.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil // Script doesn't exist, skip silently
	}

	// Find common parent directory to limit script scope
	commonDir := findCommonDirectory(files, repoRoot)

	cmd := exec.Command(scriptPath, commonDir)
	cmd.Dir = repoRoot

	return cmd.Run()
}

// findCommonDirectory finds the common parent directory of all given files.
// Returns a path relative to repoRoot.
func findCommonDirectory(files []string, repoRoot string) string {
	if len(files) == 0 {
		return ""
	}

	// Convert to relative paths
	relativePaths := make([]string, 0, len(files))
	for _, f := range files {
		rel, err := filepath.Rel(repoRoot, f)
		if err != nil {
			continue
		}
		relativePaths = append(relativePaths, filepath.Dir(rel))
	}

	if len(relativePaths) == 0 {
		return ""
	}

	// Find common prefix of all directory paths
	common := relativePaths[0]
	for _, p := range relativePaths[1:] {
		common = commonPathPrefix(common, p)
		if common == "" || common == "." {
			return ""
		}
	}

	return common
}

// commonPathPrefix finds the common directory prefix of two paths.
func commonPathPrefix(a, b string) string {
	aParts := strings.Split(filepath.ToSlash(a), "/")
	bParts := strings.Split(filepath.ToSlash(b), "/")

	var common []string
	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if aParts[i] == bParts[i] {
			common = append(common, aParts[i])
		} else {
			break
		}
	}

	if len(common) == 0 {
		return ""
	}
	return strings.Join(common, "/")
}
