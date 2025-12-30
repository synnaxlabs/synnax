// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package oracle provides a schema-first code generation system for Synnax metadata structures.
//
// Oracle parses .oracle schema files, analyzes imports and type references, and invokes
// plugins to generate type-safe code across Go, TypeScript, and Python.
//
// Basic usage:
//
//	// Analyze schemas
//	loader := analyzer.NewStandardFileLoader("/path/to/repo")
//	table, diag := analyzer.Analyze(ctx, []string{"schema/core/ranger.oracle"}, loader)
//	if diag.HasErrors() {
//	    log.Fatal(diag)
//	}
//
//	// Generate code with plugins
//	registry := plugin.NewRegistry()
//	registry.Register(myGoPlugin)
//
//	for _, p := range registry.All() {
//	    resp, err := p.Generate(&plugin.Request{
//	        Resolutions: table,
//	        OutputDir:   "/path/to/output",
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    // Write generated files...
//	}
//
// Schema file example (.oracle):
//
//	import "schema/core/label"
//
//	struct Range {
//	    field key uuid {
//	        domain id
//	    }
//
//	    field name string {
//	        domain validate {
//	            required
//	            max_length 255
//	        }
//	        domain query {
//	            eq
//	            contains
//	        }
//	    }
//
//	    field labels uuid[] {
//	        domain relation {
//	            target label.Label
//	            cardinality many_to_many
//	        }
//	    }
//	}
//
// See the docs/tech/rfc/0026-251229-oracle-schema-system.md for the full specification.
package oracle

import (
	"context"
	"os"
	"path/filepath"

	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/resolution"
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

	schemaFiles := buildSchemaFiles(files, loader, table)
	result := &GenerateResult{
		Resolutions: table,
		Files:       make(map[string][]plugin.File),
	}

	for _, p := range registry.All() {
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
func (r *GenerateResult) WriteFiles(outputDir string) error {
	for _, files := range r.Files {
		for _, f := range files {
			fullPath := filepath.Join(outputDir, f.Path)
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			if err := os.WriteFile(fullPath, f.Content, 0644); err != nil {
				return err
			}
		}
	}
	return nil
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

// buildSchemaFiles creates SchemaFile entries for each analyzed file.
func buildSchemaFiles(
	files []string,
	loader analyzer.FileLoader,
	_ *resolution.Table,
) []plugin.SchemaFile {
	result := make([]plugin.SchemaFile, 0, len(files))

	for _, file := range files {
		source, filePath, err := loader.Load(file)
		if err != nil {
			continue
		}

		parsedAST, err := parseSource(source)
		if err != nil || parsedAST == nil {
			continue
		}

		result = append(result, plugin.SchemaFile{
			AST:       parsedAST,
			FilePath:  filePath,
			Namespace: analyzer.DeriveNamespace(filePath),
		})
	}

	return result
}

// parseSource is a helper to parse source without the full analyzer.
func parseSource(source string) (parser.ISchemaContext, error) {
	ast, diag := parser.Parse(source)
	if diag != nil && diag.HasErrors() {
		return nil, diag
	}
	return ast, nil
}
