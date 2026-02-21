// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides shared test infrastructure for STL module unit tests.
package testutil

import (
	"fmt"

	"github.com/synnaxlabs/arc/stl"
)

// MockHostRuntime implements stl.HostRuntime by storing exported functions in a
// nested map keyed by [wasmModule][name].
type MockHostRuntime struct {
	Funcs map[string]map[string]any
}

// NewMockHostRuntime creates a new MockHostRuntime.
func NewMockHostRuntime() *MockHostRuntime {
	return &MockHostRuntime{Funcs: make(map[string]map[string]any)}
}

// Export registers a host function.
func (m *MockHostRuntime) Export(wasmModule, name string, impl any) error {
	inner, ok := m.Funcs[wasmModule]
	if !ok {
		inner = make(map[string]any)
		m.Funcs[wasmModule] = inner
	}
	if _, exists := inner[name]; exists {
		return fmt.Errorf("duplicate export: %s.%s", wasmModule, name)
	}
	inner[name] = impl
	return nil
}

var _ stl.HostRuntime = (*MockHostRuntime)(nil)

// Get retrieves and type-asserts a registered function. Panics on missing or
// wrong type for clear test failures.
func Get[F any](rt *MockHostRuntime, module, name string) F {
	inner, ok := rt.Funcs[module]
	if !ok {
		panic(fmt.Sprintf("module %q not found in mock runtime", module))
	}
	raw, ok := inner[name]
	if !ok {
		panic(fmt.Sprintf("function %s.%s not found in mock runtime", module, name))
	}
	fn, ok := raw.(F)
	if !ok {
		panic(fmt.Sprintf("function %s.%s has type %T, want %T", module, name, raw, fn))
	}
	return fn
}
