// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/x/set"
)

// ImportRecord is one imported module. Path is the source path
// ("math", "math.trig"); Alias is the user-visible qualifier.
type ImportRecord struct {
	Path  string
	Alias string
	AST   antlr.ParserRuleContext
	Used  bool
}

// ImportSet gates dotted-name lookups on the root Scope. AutoAll skips
// the gate (graph mode, REPLs, anywhere without source-level imports).
type ImportSet struct {
	aliases map[string]*ImportRecord
	autoAll bool
}

// NewImportSet returns an empty set that gates dotted lookups.
func NewImportSet() *ImportSet {
	return &ImportSet{aliases: make(map[string]*ImportRecord)}
}

// NewAutoImportSet returns a set that auto-imports every module.
func NewAutoImportSet() *ImportSet {
	return &ImportSet{aliases: make(map[string]*ImportRecord), autoAll: true}
}

// AutoAll reports whether the set treats every module as imported.
func (s *ImportSet) AutoAll() bool { return s.autoAll }

// Lookup returns the record bound to alias, or false.
func (s *ImportSet) Lookup(alias string) (*ImportRecord, bool) {
	if s == nil {
		return nil, false
	}
	rec, ok := s.aliases[alias]
	return rec, ok
}

// CanonicalName rewrites a qualified name's alias prefix to the underlying
// module path (`t.now` → `time.now` when `time as t` is imported). Returns
// the input unchanged when the name is unqualified or the alias is unbound.
func (s *ImportSet) CanonicalName(name string) string {
	if s == nil {
		return name
	}
	dot := strings.IndexByte(name, '.')
	if dot < 0 {
		return name
	}
	rec, ok := s.aliases[name[:dot]]
	if !ok || rec.Path == name[:dot] {
		return name
	}
	return rec.Path + name[dot:]
}

// MarkUsed flags the alias as consumed. No-op for unknown aliases.
func (s *ImportSet) MarkUsed(alias string) {
	if s == nil {
		return
	}
	if rec, ok := s.aliases[alias]; ok {
		rec.Used = true
	}
}

// Add inserts rec. Returns true if its alias was already bound.
func (s *ImportSet) Add(rec *ImportRecord) bool {
	if s == nil || rec == nil {
		return false
	}
	if _, exists := s.aliases[rec.Alias]; exists {
		return true
	}
	s.aliases[rec.Alias] = rec
	return false
}

// All returns every record (unordered).
func (s *ImportSet) All() []*ImportRecord {
	if s == nil {
		return nil
	}
	out := make([]*ImportRecord, 0, len(s.aliases))
	for _, rec := range s.aliases {
		out = append(out, rec)
	}
	return out
}

// Unused returns records that were added but never marked used.
// Returns nil for auto-import sets.
func (s *ImportSet) Unused() []*ImportRecord {
	if s == nil || s.autoAll {
		return nil
	}
	var out []*ImportRecord
	for _, rec := range s.aliases {
		if !rec.Used {
			out = append(out, rec)
		}
	}
	return out
}

// ListModules returns every module name exposed by r via ModuleResolver
// entries (unordered). Used to diagnose imports of unknown modules.
func ListModules(r Resolver) []string {
	seen := make(set.Set[string])
	collectModuleNames(r, seen)
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	return out
}

func collectModuleNames(r Resolver, seen set.Set[string]) {
	switch v := r.(type) {
	case *ModuleResolver:
		seen.Add(v.Name)
	case CompoundResolver:
		for _, sub := range v {
			collectModuleNames(sub, seen)
		}
	}
}
