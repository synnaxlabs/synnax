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

func generate(
	ctx context.Context,
	files []string,
	repoRoot string,
	registry *plugin.Registry,
) (*generateResult, *diagnostics.Diagnostics) {
	loader := analyzer.NewStandardFileLoader(repoRoot)
	table, diag := analyzer.Analyze(ctx, files, loader)
	if diag != nil && !diag.Ok() {
		return nil, diag
	}

	result := &generateResult{
		Resolutions: table,
		Files:       make(map[string][]plugin.File),
	}
	for _, p := range registry.All() {
		output.PluginStart(p.Name())
		req := &plugin.Request{Resolutions: table, RepoRoot: repoRoot}
		for _, depName := range p.Requires() {
			dep := registry.Get(depName)
			if dep == nil {
				diag.Add(diagnostics.Errorf(nil, "plugin '%s' requires unknown plugin '%s'", p.Name(), depName))
				continue
			}
			if err := dep.Check(req); err != nil {
				staleErr := &plugin.DependencyStaleError{
					Plugin:     p.Name(),
					Dependency: depName,
					Reason:     err,
				}
				errDiag := &diagnostics.Diagnostics{}
				errDiag.Add(diagnostics.Error(staleErr, nil))
				return nil, errDiag
			}
		}
		resp, err := p.Generate(req)
		if err != nil {
			diag.Add(diagnostics.Errorf(nil, "plugin %s failed: %v", p.Name(), err))
			continue
		}
		if resp != nil {
			result.Files[p.Name()] = resp.Files
			output.PluginDone(p.Name(), len(resp.Files))
		}
	}
	return result, diag
}

type generateResult struct {
	Resolutions *resolution.Table
	Files       map[string][]plugin.File
}

type syncResult struct {
	ByPlugin  map[string][]string
	Written   []string
	Unchanged []string
}

func (r *generateResult) syncFiles(outputDir string) (*syncResult, error) {
	result := &syncResult{
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

func updateLicenseHeaders(repoRoot string, files []string) error {
	if len(files) == 0 {
		return nil
	}
	return exec.OnFiles(updateLicenseHeadersCmd, files, path.Join(repoRoot, "scripts"))
}
