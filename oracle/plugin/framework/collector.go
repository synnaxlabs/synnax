// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framework

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
)

// OutputPathFunc extracts the output path for a type.
// This allows different plugins to use different output path logic.
type OutputPathFunc func(typ resolution.Type) string

// SkipFunc determines whether to skip processing a type.
type SkipFunc func(typ resolution.Type) bool

// Collector collects types by output path for code generation.
// It maintains insertion order and handles path validation.
type Collector struct {
	// byPath maps output paths to their collected types.
	byPath map[string][]resolution.Type
	// order maintains the order in which paths were first seen.
	order []string
	// pathFunc extracts the output path from a type.
	pathFunc OutputPathFunc
	// skipFunc determines whether to skip a type.
	skipFunc SkipFunc
	// repoRoot is used for output path validation.
	repoRoot string
	// validatePaths enables/disables path validation.
	validatePaths bool
	// request is used for path validation.
	request *plugin.Request
}

// NewCollector creates a collector for the given domain.
// domain is the language domain (e.g., "go", "py", "ts", "cpp", "pb").
func NewCollector(domain string, req *plugin.Request) *Collector {
	return &Collector{
		byPath:        make(map[string][]resolution.Type),
		order:         make([]string, 0),
		pathFunc:      func(typ resolution.Type) string { return output.GetPath(typ, domain) },
		skipFunc:      func(typ resolution.Type) bool { return omit.IsType(typ, domain) },
		repoRoot:      req.RepoRoot,
		validatePaths: req.RepoRoot != "",
		request:       req,
	}
}

// WithPathFunc sets a custom output path extraction function.
func (c *Collector) WithPathFunc(fn OutputPathFunc) *Collector {
	c.pathFunc = fn
	return c
}

// WithSkipFunc sets a custom skip function.
func (c *Collector) WithSkipFunc(fn SkipFunc) *Collector {
	c.skipFunc = fn
	return c
}

// Add adds a type to the collector if it has an output path and isn't skipped.
// Returns an error if path validation fails.
func (c *Collector) Add(typ resolution.Type) error {
	outputPath := c.pathFunc(typ)
	if outputPath == "" {
		return nil
	}
	if c.skipFunc != nil && c.skipFunc(typ) {
		return nil
	}
	if c.validatePaths {
		if err := c.request.ValidateOutputPath(outputPath); err != nil {
			return errors.Wrapf(err, "invalid output path for %s", typ.Name)
		}
	}
	if _, exists := c.byPath[outputPath]; !exists {
		c.order = append(c.order, outputPath)
	}
	c.byPath[outputPath] = append(c.byPath[outputPath], typ)
	return nil
}

// AddAll adds all types from a slice.
func (c *Collector) AddAll(types []resolution.Type) error {
	for _, typ := range types {
		if err := c.Add(typ); err != nil {
			return err
		}
	}
	return nil
}

// Paths returns the output paths in the order they were first seen.
func (c *Collector) Paths() []string {
	return c.order
}

// Get returns the types collected for a given output path.
func (c *Collector) Get(path string) []resolution.Type {
	return c.byPath[path]
}

// Remove removes a path from the collector and returns its types.
// This is useful for consuming paths during generation (e.g., merging enums).
func (c *Collector) Remove(path string) []resolution.Type {
	types := c.byPath[path]
	delete(c.byPath, path)
	return types
}

// Has checks if a path exists in the collector.
func (c *Collector) Has(path string) bool {
	_, ok := c.byPath[path]
	return ok
}

// Count returns the total number of types collected.
func (c *Collector) Count() int {
	count := 0
	for _, types := range c.byPath {
		count += len(types)
	}
	return count
}

// Empty returns true if no types have been collected.
func (c *Collector) Empty() bool {
	return len(c.order) == 0
}

// ForEach iterates over each output path and its types in order.
func (c *Collector) ForEach(fn func(path string, types []resolution.Type) error) error {
	for _, path := range c.order {
		types := c.byPath[path]
		if len(types) == 0 {
			continue // Skip paths that were removed
		}
		if err := fn(path, types); err != nil {
			return err
		}
	}
	return nil
}

// CollectStructs creates a collector and populates it with struct types.
func CollectStructs(domain string, req *plugin.Request) (*Collector, error) {
	c := NewCollector(domain, req)
	if err := c.AddAll(req.Resolutions.StructTypes()); err != nil {
		return nil, err
	}
	return c, nil
}

// CollectEnums creates a collector and populates it with enum types.
func CollectEnums(domain string, req *plugin.Request) (*Collector, error) {
	c := NewCollector(domain, req)
	if err := c.AddAll(req.Resolutions.EnumTypes()); err != nil {
		return nil, err
	}
	return c, nil
}

// CollectDistinct creates a collector and populates it with distinct types.
func CollectDistinct(domain string, req *plugin.Request) (*Collector, error) {
	c := NewCollector(domain, req)
	if err := c.AddAll(req.Resolutions.DistinctTypes()); err != nil {
		return nil, err
	}
	return c, nil
}

// CollectAliases creates a collector and populates it with alias types.
func CollectAliases(domain string, req *plugin.Request) (*Collector, error) {
	c := NewCollector(domain, req)
	if err := c.AddAll(req.Resolutions.AliasTypes()); err != nil {
		return nil, err
	}
	return c, nil
}
