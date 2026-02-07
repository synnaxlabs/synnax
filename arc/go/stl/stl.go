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
	"context"

	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/tetratelabs/wazero/api"
)

// Module is the unit of STL organization. It provides symbols for the analyzer,
// node factories for the scheduler, and host function implementations for the
// WASM runtime.
type Module interface {
	symbol.Resolver
	node.Factory
	// BindTo registers this module's host function implementations with the
	// WASM runtime. Called once during runtime setup.
	BindTo(ctx context.Context, rt HostRuntime) error
}

// MemorySetter is implemented by modules that need access to the WASM linear
// memory (e.g., for reading string literals or panic messages). After the WASM
// module is instantiated, the runtime calls SetMemory on every module that
// implements this interface.
type MemorySetter interface {
	SetMemory(memory api.Memory)
}

// HostRuntime abstracts the WASM runtime engine so modules don't import
// engine-specific packages.
type HostRuntime interface {
	// Export registers a host function callable from WASM. wasmModule is the
	// WASM import module name, name is the function name within it.
	Export(wasmModule, name string, impl any) error
}

// MustExport calls rt.Export and panics on error. Export only fails on
// programming errors (duplicate name, invalid signature), so panicking is
// appropriate — this follows the template.Must / regexp.MustCompile pattern.
func MustExport(rt HostRuntime, wasmModule, name string, impl any) {
	if err := rt.Export(wasmModule, name, impl); err != nil {
		panic("stl: Export(" + wasmModule + "." + name + "): " + err.Error())
	}
}

// CompoundResolver derives a symbol.CompoundResolver from a slice of modules.
func CompoundResolver(modules ...Module) symbol.CompoundResolver {
	resolvers := make(symbol.CompoundResolver, len(modules))
	for i, m := range modules {
		resolvers[i] = m
	}
	return resolvers
}

// MultiFactory derives a node.MultiFactory from a slice of modules.
func MultiFactory(modules ...Module) node.MultiFactory {
	factories := make(node.MultiFactory, len(modules))
	for i, m := range modules {
		factories[i] = m
	}
	return factories
}

type nodeKeyCtxKey struct{}

// WithNodeKey returns a new context with the given node key attached.
func WithNodeKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, nodeKeyCtxKey{}, key)
}

// NodeKeyFromContext retrieves the node key from the context.
func NodeKeyFromContext(ctx context.Context) string {
	if key, ok := ctx.Value(nodeKeyCtxKey{}).(string); ok {
		return key
	}
	return ""
}
