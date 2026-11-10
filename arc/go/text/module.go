// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// LoaderConfig configures a Loader.
type LoaderConfig struct {
	// Source returns the module text at path, or an error wrapping ErrModuleNotFound.
	Source func(ctx context.Context, path string) (Text, error)
	// Paths returns the import path of each module the root can read. May be nil.
	Paths func(ctx context.Context) ([]string, error)
	// Channels resolves channel names for each module's root. May be nil.
	Channels symbol.Resolver
	// Ambient returns the prelude symbols for each module's root. May be nil.
	Ambient func() []*symbol.Symbol
	// Config is the parser configuration for each module.
	Config parser.Config
}

// unit is a loaded module, holding its lowered program and the variable nodes an
// importer reads through.
type unit struct {
	ir       ir.IR
	varNodes map[*symbol.Symbol]varEntry
}

// Loader resolves import paths to analyzed modules for one root. Each path loads
// once, whether it succeeds or fails. It is not safe for concurrent use.
type Loader struct {
	cfg    LoaderConfig
	units  map[string]*unit
	failed map[string]error
	// stack holds the paths under analysis, outermost first.
	stack []string
	// cycle holds the chain of the first import cycle found. Every module on it
	// reports the cycle instead of its own failure.
	cycle []string
}

var _ symbol.Resolver = (*Loader)(nil)

// NewLoader returns a Loader that reads modules through cfg.
func NewLoader(cfg LoaderConfig) *Loader {
	return &Loader{
		cfg:    cfg,
		units:  make(map[string]*unit),
		failed: make(map[string]error),
	}
}

// Resolve implements symbol.Resolver. A module's own imports resolve through the
// same Loader.
func (l *Loader) Resolve(ctx context.Context, path string) (*symbol.Symbol, error) {
	if u, ok := l.units[path]; ok {
		return u.ir.Symbols, nil
	}
	if err, ok := l.failed[path]; ok {
		return nil, err
	}
	if slices.Contains(l.stack, path) {
		l.cycle = append(slices.Clone(l.stack), path)
		return nil, l.cycleError()
	}
	u, err := l.load(ctx, path)
	if err != nil {
		l.failed[path] = err
		return nil, err
	}
	l.units[path] = u
	return u.ir.Symbols, nil
}

func (l *Loader) load(ctx context.Context, path string) (*unit, error) {
	t, err := l.cfg.Source(ctx, path)
	if err != nil {
		return nil, err
	}
	l.stack = append(l.stack, path)
	defer func() { l.stack = l.stack[:len(l.stack)-1] }()
	parsed, diag := Parse(t, l.cfg.Config)
	if diag != nil && !diag.Ok() {
		return nil, compileError(path, diag)
	}
	var ambient []*symbol.Symbol
	if l.cfg.Ambient != nil {
		ambient = l.cfg.Ambient()
	}
	root := symbol.NewRoot(l.cfg.Channels, ambient, symbol.WithModules(l))
	root.Name, root.UserDefined = path, true
	u, diag := analyzeUnit(ctx, parsed, root, true, []parser.Config{l.cfg.Config})
	if diag != nil && !diag.Ok() {
		if slices.Contains(l.cycle, path) {
			return nil, l.cycleError()
		}
		return nil, compileError(path, diag)
	}
	u.ir.Symbols.Kind, u.ir.Symbols.AST = symbol.KindModule, parsed.AST
	return &u, nil
}

// loaded returns every unit the Loader holds, ordered by path.
func (l *Loader) loaded() []*unit {
	units := make([]*unit, 0, len(l.units))
	for _, path := range slices.Sorted(maps.Keys(l.units)) {
		units = append(units, l.units[path])
	}
	return units
}

// loaderOf returns the Loader installed on root, nil when root resolves no modules.
func loaderOf(root *symbol.Symbol) *Loader {
	l, _ := root.ModuleResolver().(*Loader)
	return l
}

// LoadedPaths returns the import path of every module loaded through root, at any
// depth, sorted. It is nil when root resolves no modules.
func LoadedPaths(root *symbol.Symbol) []string {
	l := loaderOf(root)
	if l == nil {
		return nil
	}
	return slices.Sorted(maps.Keys(l.units))
}

// link appends each unit's program to i. A module's top-level members join the
// root stratum, and a pending activation binds to the module scope it names.
func link(i *ir.IR, units []*unit, shell *shellBuilder, bound set.Set[string]) {
	for _, u := range units {
		i.Functions = append(i.Functions, u.ir.Functions...)
		i.Nodes = append(i.Nodes, u.ir.Nodes...)
		i.Edges = append(i.Edges, u.ir.Edges...)
		maps.Copy(i.TypeMap, u.ir.TypeMap)
		maps.Copy(i.FmtStrings, u.ir.FmtStrings)
		for _, stratum := range u.ir.Root.Strata {
			for _, m := range stratum {
				if m.Scope != nil {
					if a, ok := shell.activations[m.Scope.Key]; ok {
						m.Scope.Activations = a
						bound.Add(m.Scope.Key)
					}
				}
				if len(i.Root.Strata) == 0 {
					i.Root.Strata = []ir.Members{nil}
				}
				i.Root.Strata[0] = append(i.Root.Strata[0], m)
			}
		}
	}
}

// compileError reports the first error of the module at path, with its position in
// the module.
func compileError(path string, diag *diagnostics.Diagnostics) error {
	first := diag.Errors()[0]
	return errors.Newf(
		"module %q does not compile: %d:%d %s",
		path,
		first.Range.Start.Line+1,
		first.Range.Start.Character,
		first.Message,
	)
}

func (l *Loader) cycleError() error {
	return errors.Newf("import cycle: %s", strings.Join(l.cycle, " -> "))
}

// Search implements symbol.Resolver. It returns the import paths that start with term.
func (l *Loader) Search(ctx context.Context, term string) ([]*symbol.Symbol, error) {
	if l.cfg.Paths == nil {
		return nil, nil
	}
	paths, err := l.cfg.Paths(ctx)
	if err != nil {
		return nil, err
	}
	var modules []*symbol.Symbol
	for _, path := range paths {
		if strings.HasPrefix(path, term) {
			modules = append(
				modules,
				&symbol.Symbol{Name: path, Kind: symbol.KindModule},
			)
		}
	}
	return modules, nil
}
