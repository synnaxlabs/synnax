// Copyright 2026 Synnax Labs, Inc.
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
	"path"
	"path/filepath"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/output"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/diagnostics"
)

// Generate analyzes schema files and runs code generation with the given plugins.
func Generate(
	ctx context.Context,
	files []string,
	repoRoot string,
	registry *plugin.Registry,
) (*GenerateResult, *diagnostics.Diagnostics) {
	loader := analyzer.NewStandardFileLoader(repoRoot)
	table, diag := analyzer.Analyze(ctx, files, loader)
	if diag != nil && diag.HasErrors() {
		return nil, diag
	}

	result := &GenerateResult{
		Resolutions: table,
		Files:       make(map[string][]plugin.File),
	}
	for _, p := range registry.All() {
		output.PluginStart(p.Name())
		req := &plugin.Request{Resolutions: table, RepoRoot: repoRoot}
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
			existing, err := os.ReadFile(fullPath)
			if err == nil && string(existing) == string(f.Content) {
				result.Unchanged = append(result.Unchanged, f.Path)
				continue
			}
			dir := filepath.Dir(fullPath)
			if err = os.MkdirAll(dir, 0755); err != nil {
				return nil, err
			}
			if err = os.WriteFile(fullPath, f.Content, 0644); err != nil {
				return nil, err
			}
			result.Written = append(result.Written, f.Path)
			result.ByPlugin[pluginName] = append(result.ByPlugin[pluginName], f.Path)
		}
	}
	return result, nil
}

var updateLicenseHeadersCmd = []string{"./update_copyrights.sh"}

// UpdateLicenseHeaders runs the copyright update script on the given files and patterns.
// files can be absolute paths or glob patterns like "*.pb.go".
// repoRoot is the absolute path to the repository root.
func UpdateLicenseHeaders(repoRoot string, files []string) error {
	if len(files) == 0 {
		return nil
	}
	return exec.OnFiles(updateLicenseHeadersCmd, files, path.Join(repoRoot, "scripts"))
}
