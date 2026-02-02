// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package plugin provides the interface and types for Oracle code generation plugins.
package plugin

import (
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/resolution"
)

// Plugin is the interface that code generators must implement.
type Plugin interface {
	Name() string
	Domains() []string
	Requires() []string
	Check(req *Request) error
	Generate(req *Request) (*Response, error)
}

// Request contains all data needed for code generation.
type Request struct {
	// Resolutions holds the resolved type table from Oracle schema files.
	Resolutions *resolution.Table
	// RepoRoot is the absolute path to the repository root directory.
	RepoRoot string
}

// ResolvePath resolves a repo-relative path to an absolute path.
func (r *Request) ResolvePath(repoRelative string) string {
	return paths.Resolve(repoRelative, r.RepoRoot)
}

// RelativeImport computes the relative import path from one path to another.
func (r *Request) RelativeImport(from, to string) (string, error) {
	return paths.RelativeImport(from, to)
}

// ValidateOutputPath validates that the output path is within the repository.
func (r *Request) ValidateOutputPath(path string) error {
	return paths.ValidateOutput(path, r.RepoRoot)
}

// Response contains the generated files from a plugin.
type Response struct {
	// Files holds the list of generated files.
	Files []File
}

// PostWriter is an optional interface for post-processing.
type PostWriter interface {
	PostWrite(files []string) error
}

// File represents a single generated file.
type File struct {
	// Path is the output file path relative to the repository root.
	Path string
	// Content is the generated file content.
	Content []byte
}

// Registry holds registered plugins.
type Registry struct {
	plugins map[string]Plugin
}

// NewRegistry creates a new empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]Plugin)}
}

// Register adds a plugin to the registry. Returns an error if a plugin
// with the same name is already registered.
func (r *Registry) Register(p Plugin) error {
	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		return &DuplicatePluginError{Name: name}
	}
	r.plugins[name] = p
	return nil
}

// Get retrieves a plugin by name, or nil if not found.
func (r *Registry) Get(name string) Plugin {
	return r.plugins[name]
}

// All returns all registered plugins.
func (r *Registry) All() []Plugin {
	result := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		result = append(result, p)
	}
	return result
}

// DuplicatePluginError is returned when attempting to register a plugin
// with a name that is already registered.
type DuplicatePluginError struct {
	// Name is the duplicate plugin name.
	Name string
}

func (e *DuplicatePluginError) Error() string {
	return "duplicate plugin: " + e.Name
}
