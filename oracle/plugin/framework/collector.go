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
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
)

type OutputPathFunc func(typ resolution.Type) string

type SkipFunc func(typ resolution.Type) bool

type PathEntry struct {
	Path  string
	Types []resolution.Type
}

type Collector struct {
	entries       []PathEntry
	pathFunc      OutputPathFunc
	skipFunc      SkipFunc
	validatePaths bool
	request       *plugin.Request
}

func NewCollector(domain string, req *plugin.Request) *Collector {
	return &Collector{
		entries:       make([]PathEntry, 0),
		pathFunc:      func(typ resolution.Type) string { return output.GetPath(typ, domain) },
		skipFunc:      func(typ resolution.Type) bool { return omit.IsType(typ, domain) },
		validatePaths: req.RepoRoot != "",
		request:       req,
	}
}

func (c *Collector) WithPathFunc(fn OutputPathFunc) *Collector {
	c.pathFunc = fn
	return c
}

func (c *Collector) WithSkipFunc(fn SkipFunc) *Collector {
	c.skipFunc = fn
	return c
}

func (c *Collector) findEntry(path string) int {
	for i, e := range c.entries {
		if e.Path == path {
			return i
		}
	}
	return -1
}

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
	idx := c.findEntry(outputPath)
	if idx == -1 {
		c.entries = append(c.entries, PathEntry{Path: outputPath, Types: []resolution.Type{typ}})
	} else {
		c.entries[idx].Types = append(c.entries[idx].Types, typ)
	}
	return nil
}

func (c *Collector) AddAll(types []resolution.Type) error {
	for _, typ := range types {
		if err := c.Add(typ); err != nil {
			return err
		}
	}
	return nil
}

func (c *Collector) Paths() []string {
	return lo.Map(c.entries, func(e PathEntry, _ int) string { return e.Path })
}

func (c *Collector) Get(path string) []resolution.Type {
	entry, found := lo.Find(c.entries, func(e PathEntry) bool { return e.Path == path })
	if !found {
		return nil
	}
	return entry.Types
}

func (c *Collector) Remove(path string) []resolution.Type {
	idx := c.findEntry(path)
	if idx == -1 {
		return nil
	}
	types := c.entries[idx].Types
	c.entries[idx].Types = nil
	return types
}

func (c *Collector) Has(path string) bool {
	entry, found := lo.Find(c.entries, func(e PathEntry) bool { return e.Path == path })
	return found && len(entry.Types) > 0
}

func (c *Collector) Count() int {
	return lo.SumBy(c.entries, func(e PathEntry) int { return len(e.Types) })
}

func (c *Collector) Empty() bool {
	return len(c.entries) == 0
}

func (c *Collector) ForEach(fn func(path string, types []resolution.Type) error) error {
	for _, entry := range c.entries {
		if len(entry.Types) == 0 {
			continue
		}
		if err := fn(entry.Path, entry.Types); err != nil {
			return err
		}
	}
	return nil
}

func CollectStructs(domain string, req *plugin.Request) (*Collector, error) {
	c := NewCollector(domain, req)
	if err := c.AddAll(req.Resolutions.StructTypes()); err != nil {
		return nil, err
	}
	return c, nil
}

func CollectEnums(domain string, req *plugin.Request) (*Collector, error) {
	c := NewCollector(domain, req)
	if err := c.AddAll(req.Resolutions.EnumTypes()); err != nil {
		return nil, err
	}
	return c, nil
}

func CollectDistinct(domain string, req *plugin.Request) (*Collector, error) {
	c := NewCollector(domain, req)
	if err := c.AddAll(req.Resolutions.DistinctTypes()); err != nil {
		return nil, err
	}
	return c, nil
}

func CollectAliases(domain string, req *plugin.Request) (*Collector, error) {
	c := NewCollector(domain, req)
	if err := c.AddAll(req.Resolutions.AliasTypes()); err != nil {
		return nil, err
	}
	return c, nil
}
