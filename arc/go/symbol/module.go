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
	"context"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// ErrModuleNotFound is returned when no module exists at an import path.
var ErrModuleNotFound = errors.New("module not found")

// ModuleResolver supplies modules that are not in the ambient prelude, such as user
// modules stored in a Core, by their import path.
type ModuleResolver interface {
	// ResolveModule returns the module at path with its declarations as children.
	// Returns ErrModuleNotFound when there is none.
	ResolveModule(ctx context.Context, path string) (*Symbol, error)
	// SearchModules returns the import paths that start with prefix, sorted.
	SearchModules(ctx context.Context, prefix string) ([]string, error)
}

// SetModuleResolver makes the root of s resolve import paths through r.
func (s *Symbol) SetModuleResolver(r ModuleResolver) { s.Root().moduleResolver = r }

// ResolveModule returns the module at path through the root's ModuleResolver. Returns
// ErrModuleNotFound without one.
func (s *Symbol) ResolveModule(ctx context.Context, path string) (*Symbol, error) {
	r := s.Root().moduleResolver
	if r == nil {
		return nil, errors.Wrapf(ErrModuleNotFound, "%q", path)
	}
	return r.ResolveModule(ctx, path)
}

// SearchModules returns the import paths that start with prefix, none without a
// ModuleResolver.
func (s *Symbol) SearchModules(ctx context.Context, prefix string) ([]string, error) {
	r := s.Root().moduleResolver
	if r == nil {
		return nil, nil
	}
	return r.SearchModules(ctx, prefix)
}

// ImportedModules returns every module that s imports, directly or through another
// module, each once in import order.
func (s *Symbol) ImportedModules() []*Symbol {
	var (
		modules []*Symbol
		seen    = set.New[*Symbol]()
		walk    func(scope *Symbol)
	)
	walk = func(scope *Symbol) {
		for _, child := range scope.children {
			if child.Kind != KindModuleAlias || child.Target == nil ||
				seen.Contains(child.Target) {
				continue
			}
			seen.Add(child.Target)
			modules = append(modules, child.Target)
			walk(child.Target)
		}
	}
	walk(s)
	return modules
}
