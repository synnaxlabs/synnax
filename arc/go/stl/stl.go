// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stl defines the standard library module interfaces for Arc. A Module is the
// unit of STL organization: it provides symbols for the analyzer, node factories for
// the scheduler, and host function implementations for the WASM runtime.
package stl

import (
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/stl/constant"
	"github.com/synnaxlabs/arc/stl/control"
	"github.com/synnaxlabs/arc/stl/errors"
	"github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/op"
	"github.com/synnaxlabs/arc/stl/selector"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stable"
	"github.com/synnaxlabs/arc/stl/stateful"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/symbol"
)

// SymbolResolver is the legacy resolver chain that mixes all STL modules into a
// single fan-out of MapResolver and ModuleResolver entries. Kept for callers
// that have not yet migrated to BuildRoot.
var SymbolResolver = symbol.CompoundResolver{
	channels.SymbolResolver,
	constant.SymbolResolver,
	control.SymbolResolver,
	errors.SymbolResolver,
	math.SymbolResolver,
	op.SymbolResolver,
	selector.SymbolResolver,
	series.SymbolResolver,
	stable.SymbolResolver,
	stateful.SymbolResolver,
	strings.SymbolResolver,
	time.SymbolResolver,
}

// BuildRoot returns a user-program root whose Parent is an ambient prelude
// scope containing the STL modules and bare globals. Iterating the returned
// root's Children sees only user-defined entities; STL is reached by the
// normal lexical parent walk.
//
// Callers that need cluster-backed lookups (channels, status types) should
// set GlobalResolver on the returned root before passing it to the analyzer.
// Use NewRoot when both STL builtins and a dynamic resolver are required.
func BuildRoot() *symbol.Symbol {
	ambient := &symbol.Symbol{Kind: symbol.KindAmbient}
	for _, mod := range stlModules() {
		ambient.AddChild(mod)
	}
	for _, sym := range stlBareGlobals() {
		s := sym
		ambient.AddChild(&s)
	}
	root := symbol.CreateRoot(nil)
	ambient.AddChild(root)
	return root
}

// NewRoot is BuildRoot with dynamicResolver attached as the root's
// GlobalResolver. The dynamic resolver handles lookups for symbols that
// live outside the program (e.g., cluster channels) when local children
// do not match. Pass nil when no external lookups are needed.
func NewRoot(dynamicResolver symbol.Resolver) *symbol.Symbol {
	root := BuildRoot()
	root.GlobalResolver = dynamicResolver
	return root
}

// NewAutoImportRoot returns a root with every STL module pre-bound as a
// KindModuleAlias child. Used for graph mode and other entry points that
// reference module-qualified names (`control.set_authority`,
// `time.interval`) without producing `import` statements. Text-mode
// callers should use NewRoot and let the analyzer install aliases from
// the source-level import declarations.
func NewAutoImportRoot(dynamicResolver symbol.Resolver) *symbol.Symbol {
	root := NewRoot(dynamicResolver)
	ambient := root.Parent
	if ambient == nil {
		return root
	}
	for _, child := range ambient.Children {
		if child.Kind != symbol.KindModule {
			continue
		}
		alias := &symbol.Symbol{
			Name:   child.Name,
			Kind:   symbol.KindModuleAlias,
			Target: child,
			Parent: root,
		}
		root.Children = append(root.Children, alias)
	}
	return root
}

func stlModules() []*symbol.Symbol {
	return []*symbol.Symbol{
		channels.BuildModule(),
		control.BuildModule(),
		errors.BuildModule(),
		math.BuildModule(),
		series.BuildModule(),
		stable.BuildModule(),
		stateful.BuildModule(),
		strings.BuildModule(),
		time.BuildModule(),
	}
}

func stlBareGlobals() []symbol.Symbol {
	var out []symbol.Symbol
	out = append(out, channels.BareGlobals()...)
	out = append(out, constant.BareGlobals()...)
	out = append(out, control.BareGlobals()...)
	out = append(out, math.BareGlobals()...)
	out = append(out, op.BareGlobals()...)
	out = append(out, selector.BareGlobals()...)
	out = append(out, series.BareGlobals()...)
	out = append(out, stable.BareGlobals()...)
	out = append(out, time.BareGlobals()...)
	return out
}
