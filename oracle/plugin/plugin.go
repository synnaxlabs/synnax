// Copyright 2025 Synnax Labs, Inc.
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
	Schemas     []SchemaFile
	Resolutions *resolution.Table
	Options     map[string]interface{}
	RepoRoot    string
	OutputDir   string
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

func (r *Registry) Names() []string {
	result := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		result = append(result, name)
	}
	return result
}

type DuplicatePluginError struct {
	Name string
}

func (e *DuplicatePluginError) Error() string {
	return "duplicate plugin: " + e.Name
}

// FilterTypesForPlugin filters types based on the plugin's domains.
func FilterTypesForPlugin(p Plugin, table *resolution.Table) []resolution.Type {
	domains := p.Domains()
	if len(domains) == 0 {
		return table.Types
	}

	domainSet := make(map[string]bool, len(domains))
	for _, d := range domains {
		domainSet[d] = true
	}

	var result []resolution.Type
	for _, typ := range table.Types {
		if typeHasDomain(typ, domainSet) {
			result = append(result, typ)
		}
	}
	return result
}

func typeHasDomain(typ resolution.Type, domains map[string]bool) bool {
	for domainName := range typ.Domains {
		if domains[domainName] {
			return true
		}
	}
	if form, ok := typ.Form.(resolution.StructForm); ok {
		for _, field := range form.Fields {
			for domainName := range field.Domains {
				if domains[domainName] {
					return true
				}
			}
		}
	}
	return false
}

// FilterFieldsWithDomain returns fields that have any of the specified domains.
func FilterFieldsWithDomain(typ resolution.Type, domains []string) []resolution.Field {
	form, ok := typ.Form.(resolution.StructForm)
	if !ok {
		return nil
	}
	if len(domains) == 0 {
		return form.Fields
	}

	domainSet := make(map[string]bool, len(domains))
	for _, d := range domains {
		domainSet[d] = true
	}

	var result []resolution.Field
	for _, field := range form.Fields {
		for domainName := range field.Domains {
			if domainSet[domainName] {
				result = append(result, field)
				break
			}
		}
	}
	return result
}

// GetFieldDomain returns a specific domain from a field.
func GetFieldDomain(field resolution.Field, domainName string) (resolution.Domain, bool) {
	d, ok := field.Domains[domainName]
	return d, ok
}

// GetTypeDomain returns a specific domain from a type.
func GetTypeDomain(typ resolution.Type, domainName string) (resolution.Domain, bool) {
	d, ok := typ.Domains[domainName]
	return d, ok
}

// HasDomain checks if a field has a specific domain.
func HasDomain(field resolution.Field, domainName string) bool {
	_, ok := field.Domains[domainName]
	return ok
}

// GetExpressionValue returns the first value of an expression.
func GetExpressionValue(expr resolution.Expression) resolution.ExpressionValue {
	if len(expr.Values) > 0 {
		return expr.Values[0]
	}
	return resolution.ExpressionValue{}
}

// FindExpression finds an expression by name in a domain.
func FindExpression(domain resolution.Domain, name string) (resolution.Expression, bool) {
	for _, expr := range domain.Expressions {
		if expr.Name == name {
			return expr, true
		}
	}
	return resolution.Expression{}, false
}

// HasExpression checks if a domain has an expression with the given name.
func HasExpression(domain resolution.Domain, name string) bool {
	_, ok := FindExpression(domain, name)
	return ok
}
