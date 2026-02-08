// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stateful

import (
	"context"

	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

type Module struct {
	series  *state.SeriesHandleStore
	strings *state.StringHandleStore

	stateU8     map[string]map[uint32]uint8
	stateU16    map[string]map[uint32]uint16
	stateU32    map[string]map[uint32]uint32
	stateU64    map[string]map[uint32]uint64
	stateI8     map[string]map[uint32]int8
	stateI16    map[string]map[uint32]int16
	stateI32    map[string]map[uint32]int32
	stateI64    map[string]map[uint32]int64
	stateF32    map[string]map[uint32]float32
	stateF64    map[string]map[uint32]float64
	stateString map[string]map[uint32]string
	stateSeries map[string]map[uint32]telem.Series
}

func NewModule(series *state.SeriesHandleStore, strings *state.StringHandleStore) *Module {
	return &Module{
		series:      series,
		strings:     strings,
		stateU8:     make(map[string]map[uint32]uint8),
		stateU16:    make(map[string]map[uint32]uint16),
		stateU32:    make(map[string]map[uint32]uint32),
		stateU64:    make(map[string]map[uint32]uint64),
		stateI8:     make(map[string]map[uint32]int8),
		stateI16:    make(map[string]map[uint32]int16),
		stateI32:    make(map[string]map[uint32]int32),
		stateI64:    make(map[string]map[uint32]int64),
		stateF32:    make(map[string]map[uint32]float32),
		stateF64:    make(map[string]map[uint32]float64),
		stateString: make(map[string]map[uint32]string),
		stateSeries: make(map[string]map[uint32]telem.Series),
	}
}

var numConstraint = types.NumericConstraint()

var symResolver = &symbol.ModuleResolver{
	Name: "state",
	Members: symbol.MapResolver{
		"load": {
			Name: "load",
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "id", Type: types.I32()}, {Name: "init", Type: types.Variable("T", &numConstraint)}},
				Outputs: types.Params{{Name: "value", Type: types.Variable("T", &numConstraint)}},
			}),
		},
		"store": {
			Name: "store",
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "id", Type: types.I32()}, {Name: "value", Type: types.Variable("T", &numConstraint)}},
			}),
		},
		"load_series": {
			Name: "load_series",
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "id", Type: types.I32()}, {Name: "init", Type: types.I32()}},
				Outputs: types.Params{{Name: "handle", Type: types.I32()}},
			}),
		},
		"store_series": {
			Name: "store_series",
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "id", Type: types.I32()}, {Name: "handle", Type: types.I32()}},
			}),
		},
	},
}

func (m *Module) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	return symResolver.Resolve(ctx, name)
}

func (m *Module) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	return symResolver.Search(ctx, term)
}

func (m *Module) Create(_ context.Context, _ node.Config) (node.Node, error) {
	return nil, query.ErrNotFound
}

func (m *Module) BindTo(_ context.Context, rt stl.HostRuntime) error {
	bindScalarI32[uint8](rt, m, m.stateU8, "u8")
	bindScalarI32[uint16](rt, m, m.stateU16, "u16")
	bindScalarI32[uint32](rt, m, m.stateU32, "u32")
	bindScalarI32[int8](rt, m, m.stateI8, "i8")
	bindScalarI32[int16](rt, m, m.stateI16, "i16")
	bindScalarI32[int32](rt, m, m.stateI32, "i32")
	bindScalarI64[uint64](rt, m, m.stateU64, "u64")
	bindScalarI64[int64](rt, m, m.stateI64, "i64")
	bindScalarF32(rt, m)
	bindScalarF64(rt, m)
	bindStr(rt, m)
	bindSeries(rt, m, "u8")
	bindSeries(rt, m, "u16")
	bindSeries(rt, m, "u32")
	bindSeries(rt, m, "u64")
	bindSeries(rt, m, "i8")
	bindSeries(rt, m, "i16")
	bindSeries(rt, m, "i32")
	bindSeries(rt, m, "i64")
	bindSeries(rt, m, "f32")
	bindSeries(rt, m, "f64")
	return nil
}

type i32Compatible interface {
	uint8 | uint16 | uint32 | int8 | int16 | int32
}

func bindScalarI32[T i32Compatible](
	rt stl.HostRuntime,
	m *Module,
	store map[string]map[uint32]T,
	suffix string,
) {
	stl.MustExport(rt, "state", "load_"+suffix,
		func(ctx context.Context, varID uint32, initValue uint32) uint32 {
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := store[key]
			if !ok {
				inner = make(map[uint32]T)
				store[key] = inner
			}
			if value, ok := inner[varID]; ok {
				return uint32(value)
			}
			inner[varID] = T(initValue)
			return initValue
		})
	stl.MustExport(rt, "state", "store_"+suffix,
		func(ctx context.Context, varID uint32, value uint32) {
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := store[key]
			if !ok {
				inner = make(map[uint32]T)
				store[key] = inner
			}
			inner[varID] = T(value)
		})
}

type i64Compatible interface {
	uint64 | int64
}

func bindScalarI64[T i64Compatible](
	rt stl.HostRuntime,
	m *Module,
	store map[string]map[uint32]T,
	suffix string,
) {
	stl.MustExport(rt, "state", "load_"+suffix,
		func(ctx context.Context, varID uint32, initValue uint64) uint64 {
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := store[key]
			if !ok {
				inner = make(map[uint32]T)
				store[key] = inner
			}
			if value, ok := inner[varID]; ok {
				return uint64(value)
			}
			inner[varID] = T(initValue)
			return initValue
		})
	stl.MustExport(rt, "state", "store_"+suffix,
		func(ctx context.Context, varID uint32, value uint64) {
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := store[key]
			if !ok {
				inner = make(map[uint32]T)
				store[key] = inner
			}
			inner[varID] = T(value)
		})
}

func bindScalarF32(rt stl.HostRuntime, m *Module) {
	stl.MustExport(rt, "state", "load_f32",
		func(ctx context.Context, varID uint32, initValue float32) float32 {
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := m.stateF32[key]
			if !ok {
				inner = make(map[uint32]float32)
				m.stateF32[key] = inner
			}
			if value, ok := inner[varID]; ok {
				return value
			}
			inner[varID] = initValue
			return initValue
		})
	stl.MustExport(rt, "state", "store_f32",
		func(ctx context.Context, varID uint32, value float32) {
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := m.stateF32[key]
			if !ok {
				inner = make(map[uint32]float32)
				m.stateF32[key] = inner
			}
			inner[varID] = value
		})
}

func bindScalarF64(rt stl.HostRuntime, m *Module) {
	stl.MustExport(rt, "state", "load_f64",
		func(ctx context.Context, varID uint32, initValue float64) float64 {
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := m.stateF64[key]
			if !ok {
				inner = make(map[uint32]float64)
				m.stateF64[key] = inner
			}
			if value, ok := inner[varID]; ok {
				return value
			}
			inner[varID] = initValue
			return initValue
		})
	stl.MustExport(rt, "state", "store_f64",
		func(ctx context.Context, varID uint32, value float64) {
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := m.stateF64[key]
			if !ok {
				inner = make(map[uint32]float64)
				m.stateF64[key] = inner
			}
			inner[varID] = value
		})
}

func bindStr(rt stl.HostRuntime, m *Module) {
	stl.MustExport(rt, "state", "load_str",
		func(ctx context.Context, varID uint32, initHandle uint32) uint32 {
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := m.stateString[key]
			if !ok {
				inner = make(map[uint32]string)
				m.stateString[key] = inner
			}
			if str, ok := inner[varID]; ok {
				return m.strings.Create(str)
			}
			if initStr, ok := m.strings.Get(initHandle); ok {
				inner[varID] = initStr
			}
			return initHandle
		})
	stl.MustExport(rt, "state", "store_str",
		func(ctx context.Context, varID uint32, handle uint32) {
			str, ok := m.strings.Get(handle)
			if !ok {
				return
			}
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := m.stateString[key]
			if !ok {
				inner = make(map[uint32]string)
				m.stateString[key] = inner
			}
			inner[varID] = str
		})
}

func bindSeries(rt stl.HostRuntime, m *Module, suffix string) {
	stl.MustExport(rt, "state", "load_series_"+suffix,
		func(ctx context.Context, varID uint32, initHandle uint32) uint32 {
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := m.stateSeries[key]
			if !ok {
				inner = make(map[uint32]telem.Series)
				m.stateSeries[key] = inner
			}
			if s, ok := inner[varID]; ok {
				return m.series.Store(s)
			}
			if initS, ok := m.series.Get(initHandle); ok {
				inner[varID] = initS
			}
			return initHandle
		})
	stl.MustExport(rt, "state", "store_series_"+suffix,
		func(ctx context.Context, varID uint32, handle uint32) {
			key := stl.NodeKeyFromContext(ctx)
			inner, ok := m.stateSeries[key]
			if !ok {
				inner = make(map[uint32]telem.Series)
				m.stateSeries[key] = inner
			}
			if s, ok := m.series.Get(handle); ok {
				inner[varID] = s
			}
		})
}

var _ stl.Module = (*Module)(nil)
