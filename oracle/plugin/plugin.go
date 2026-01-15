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
	Resolutions *resolution.Table
	RepoRoot    string
}

func (r *Request) ResolvePath(repoRelative string) string {
	return paths.Resolve(repoRelative, r.RepoRoot)
}

func (r *Request) RelativeImport(from, to string) (string, error) {
	return paths.RelativeImport(from, to)
}

func (r *Request) ValidateOutputPath(path string) error {
	return paths.ValidateOutput(path, r.RepoRoot)
}

// SchemaFile represents a single schema file.
type SchemaFile struct {
	FilePath string
}

// Response contains the generated files from a plugin.
type Response struct {
	Files []File
}

// PostWriter is an optional interface for post-processing.
type PostWriter interface {
	PostWrite(files []string) error
}

// File represents a single generated file.
type File struct {
	Path    string
	Content []byte
}

// Registry holds registered plugins.
type Registry struct {
	plugins map[string]Plugin
}

func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]Plugin)}
}

func (r *Registry) Register(p Plugin) error {
	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		return &DuplicatePluginError{Name: name}
	}
	r.plugins[name] = p
	return nil
}

func (r *Registry) Get(name string) Plugin {
	return r.plugins[name]
}

func (r *Registry) All() []Plugin {
	result := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		result = append(result, p)
	}
	return result
}

type DuplicatePluginError struct {
	Name string
}

func (e *DuplicatePluginError) Error() string {
	return "duplicate plugin: " + e.Name
}
