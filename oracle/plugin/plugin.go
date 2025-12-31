// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package plugin provides the interface and types for Oracle code generation plugins.
// Plugins receive parsed and analyzed schemas and generate language-specific code.
package plugin

import (
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/resolution"
)

// Plugin is the interface that code generators must implement.
// Each plugin generates code for a specific target (Go, TypeScript, Python, etc.)
// and may handle specific domains (validation, query, index, etc.).
type Plugin interface {
	// Name returns the plugin identifier (e.g., "go", "ts", "py")
	Name() string

	// Domains returns the domain names this plugin handles.
	// An empty slice means the plugin processes all structs regardless of domains.
	// If non-empty, the plugin will only receive structs/fields that have
	// at least one of the specified domains.
	Domains() []string

	// Requires returns the names of plugins that must run before this one.
	// Returns nil if this plugin has no dependencies.
	// The orchestrator will call Check() on each required plugin before
	// running this plugin's Generate().
	Requires() []string

	// Check verifies this plugin's output is up-to-date with the schemas.
	// Uses mtime comparison: schema file mtime vs generated file mtime.
	// Returns nil if all generated files are fresh, or an error describing
	// which files are stale and why.
	Check(req *Request) error

	// Generate produces output files from the analyzed schemas.
	// It receives all schemas and the resolution table with resolved types.
	Generate(req *Request) (*Response, error)
}

// Request contains all data needed for code generation.
type Request struct {
	// Schemas contains all parsed schema files with repo-relative paths
	Schemas []SchemaFile

	// Resolutions contains the resolution table with all resolved types
	Resolutions *resolution.Table

	// Options contains plugin-specific configuration from oracle.yaml
	Options map[string]interface{}

	// RepoRoot is the absolute path to the git repository root.
	// All paths in Schemas and outputs should be relative to this.
	RepoRoot string

	// OutputDir is the base output directory for generated files.
	// Deprecated: This is always equal to RepoRoot. Use RepoRoot instead.
	OutputDir string
}

// ResolvePath converts a repo-relative path to an absolute path.
func (r *Request) ResolvePath(repoRelative string) string {
	return paths.Resolve(repoRelative, r.RepoRoot)
}

// RelativeImport calculates the relative import path between two repo-relative directories.
// Both from and to should be repo-relative directory paths.
// Returns a path suitable for use in import statements.
func (r *Request) RelativeImport(from, to string) (string, error) {
	return paths.RelativeImport(from, to)
}

// ValidateOutputPath validates that an output path is valid and within repo bounds.
func (r *Request) ValidateOutputPath(path string) error {
	return paths.ValidateOutput(path, r.RepoRoot)
}

// SchemaFile represents a single schema file.
type SchemaFile struct {
	// FilePath is the repo-relative path to the source file (e.g., "schema/core/user.oracle")
	FilePath string
}

// Response contains the generated files from a plugin.
type Response struct {
	// Files contains all generated files
	Files []File
}

// PostWriter is an optional interface plugins can implement to run post-processing
// after files have been written to disk (e.g., formatting, linting).
type PostWriter interface {
	// PostWrite is called after all files have been written to disk.
	// It receives the absolute paths of the files that were written.
	PostWrite(files []string) error
}

// File represents a single generated file.
type File struct {
	// Path is the file path relative to the output directory
	Path string

	// Content is the file contents
	Content []byte
}

// Registry holds registered plugins and provides plugin management.
type Registry struct {
	plugins map[string]Plugin
}

// NewRegistry creates a new empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
	}
}

// Register adds a plugin to the registry.
// Returns an error if a plugin with the same name is already registered.
func (r *Registry) Register(p Plugin) error {
	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		return &DuplicatePluginError{Name: name}
	}
	r.plugins[name] = p
	return nil
}

// Get returns a plugin by name, or nil if not found.
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

// Names returns the names of all registered plugins.
func (r *Registry) Names() []string {
	result := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		result = append(result, name)
	}
	return result
}

// DuplicatePluginError is returned when registering a plugin with a duplicate name.
type DuplicatePluginError struct {
	Name string
}

func (e *DuplicatePluginError) Error() string {
	return "duplicate plugin: " + e.Name
}

// FilterStructsForPlugin filters structs based on the plugin's domains.
// If the plugin has no domain filter, all structs are returned.
// Otherwise, only structs that have fields with matching domains are returned.
func FilterStructsForPlugin(p Plugin, table *resolution.Table) []*resolution.Struct {
	domains := p.Domains()
	if len(domains) == 0 {
		// No filter - return all structs
		return table.AllStructs()
	}

	domainSet := make(map[string]bool, len(domains))
	for _, d := range domains {
		domainSet[d] = true
	}

	var result []*resolution.Struct
	for _, entry := range table.AllStructs() {
		if structHasDomain(entry, domainSet) {
			result = append(result, entry)
		}
	}
	return result
}

// structHasDomain checks if a struct has any of the specified domains.
func structHasDomain(entry *resolution.Struct, domains map[string]bool) bool {
	// Check struct-level domains
	for domainName := range entry.Domains {
		if domains[domainName] {
			return true
		}
	}

	// Check field-level domains
	for _, field := range entry.Fields {
		for domainName := range field.Domains {
			if domains[domainName] {
				return true
			}
		}
	}

	return false
}

// FilterFieldsWithDomain returns fields that have any of the specified domains.
func FilterFieldsWithDomain(entry *resolution.Struct, domains []string) []*resolution.Field {
	if len(domains) == 0 {
		// No filter - return all fields
		result := make([]*resolution.Field, 0, len(entry.Fields))
		for _, field := range entry.Fields {
			result = append(result, field)
		}
		return result
	}

	domainSet := make(map[string]bool, len(domains))
	for _, d := range domains {
		domainSet[d] = true
	}

	var result []*resolution.Field
	for _, field := range entry.Fields {
		for domainName := range field.Domains {
			if domainSet[domainName] {
				result = append(result, field)
				break
			}
		}
	}
	return result
}

// GetFieldDomain returns a specific domain from a field, or nil if not present.
func GetFieldDomain(field *resolution.Field, domainName string) *resolution.Domain {
	return field.Domains[domainName]
}

// GetStructDomain returns a specific domain from a struct, or nil if not present.
func GetStructDomain(entry *resolution.Struct, domainName string) *resolution.Domain {
	return entry.Domains[domainName]
}

// HasDomain checks if a field has a specific domain.
func HasDomain(field *resolution.Field, domainName string) bool {
	_, ok := field.Domains[domainName]
	return ok
}

// GetExpressionValue returns the first value of an expression, or a zero value if none.
func GetExpressionValue(expr *resolution.Expression) resolution.ExpressionValue {
	if len(expr.Values) > 0 {
		return expr.Values[0]
	}
	return resolution.ExpressionValue{}
}

// FindExpression finds an expression by name in a domain, or nil if not found.
func FindExpression(domain *resolution.Domain, name string) *resolution.Expression {
	if domain == nil {
		return nil
	}
	for _, expr := range domain.Expressions {
		if expr.Name == name {
			return expr
		}
	}
	return nil
}

// HasExpression checks if a domain has an expression with the given name.
func HasExpression(domain *resolution.Domain, name string) bool {
	return FindExpression(domain, name) != nil
}
