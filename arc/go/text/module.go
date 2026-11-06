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
	"slices"
	"strings"

	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/errors"
)

// LoaderConfig configures a Loader.
type LoaderConfig struct {
	// Source returns the module text at path, or an error wrapping ErrModuleNotFound.
	Source func(ctx context.Context, path string) (Text, error)
	// Channels resolves channel names for each module's root. May be nil.
	Channels symbol.Resolver
	// Ambient returns the prelude symbols for each module's root. May be nil.
	Ambient func() []*symbol.Symbol
	// Config is the parser configuration for each module.
	Config parser.Config
}

// Loader resolves import paths to analyzed modules for one root, each once. It is
// not safe for concurrent use.
type Loader struct {
	cfg     LoaderConfig
	modules map[string]*symbol.Symbol
	// stack holds the paths under analysis, outermost first.
	stack []string
	// cycle is the first import cycle found, reported by every module on its chain.
	cycle error
}

var _ symbol.Resolver = (*Loader)(nil)

// NewLoader returns a Loader that reads modules through cfg.
func NewLoader(cfg LoaderConfig) *Loader {
	return &Loader{cfg: cfg, modules: make(map[string]*symbol.Symbol)}
}

// Resolve implements symbol.Resolver. A module's own imports resolve through the
// same Loader.
func (l *Loader) Resolve(ctx context.Context, path string) (*symbol.Symbol, error) {
	if mod, ok := l.modules[path]; ok {
		return mod, nil
	}
	if slices.Contains(l.stack, path) {
		chain := strings.Join(append(slices.Clone(l.stack), path), " -> ")
		l.cycle = errors.Newf("import cycle: %s", chain)
		return nil, l.cycle
	}
	t, err := l.cfg.Source(ctx, path)
	if err != nil {
		return nil, err
	}
	l.stack = append(l.stack, path)
	defer func() { l.stack = l.stack[:len(l.stack)-1] }()
	parsed, diag := Parse(t, l.cfg.Config)
	if diag != nil && !diag.Ok() {
		return nil, errors.Newf("module %q does not compile", path)
	}
	var ambient []*symbol.Symbol
	if l.cfg.Ambient != nil {
		ambient = l.cfg.Ambient()
	}
	root := symbol.NewRoot(l.cfg.Channels, ambient, symbol.WithModules(l))
	i, diag := AnalyzeModule(ctx, parsed, root, l.cfg.Config)
	if diag != nil && !diag.Ok() {
		if l.cycle != nil {
			return nil, l.cycle
		}
		return nil, errors.Newf("module %q does not compile", path)
	}
	mod := i.Symbols
	mod.Kind, mod.Name, mod.AST = symbol.KindModule, path, parsed.AST
	l.modules[path] = mod
	return mod, nil
}

// Search implements symbol.Resolver. Import paths are not searched.
func (l *Loader) Search(context.Context, string) ([]*symbol.Symbol, error) {
	return nil, nil
}
