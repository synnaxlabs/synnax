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
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/tetratelabs/wazero"
)

// Host is the runtime host-side support for the state module: it registers
// the WASM host bindings (load_*, store_*, load_series_*, store_series_*)
// and holds the per-program state maps. Stateful depends on the series and
// strings ProgramStates because some state operations allocate values
// through them (e.g. store_str stores a string handle from the strings
// ProgramState).
type Host struct {
	series  *series.ProgramState
	strings *strings.ProgramState

	currentNodeKey string

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

// SetNodeKey sets the active node key so subsequent state.load / state.store
// operations scope to the right node's slots. Called by the scheduler when
// entering a node.
func (h *Host) SetNodeKey(key string) { h.currentNodeKey = key }

// ClearNode discards all stateful variable slots held for the node with the
// given key, so its variables re-initialize on the next load. Called by the
// runtime when the stage containing the node is activated.
func (h *Host) ClearNode(key string) {
	delete(h.stateU8, key)
	delete(h.stateU16, key)
	delete(h.stateU32, key)
	delete(h.stateU64, key)
	delete(h.stateI8, key)
	delete(h.stateI16, key)
	delete(h.stateI32, key)
	delete(h.stateI64, key)
	delete(h.stateF32, key)
	delete(h.stateF64, key)
	delete(h.stateString, key)
	delete(h.stateSeries, key)
}

// NewHost registers the state module's WASM host bindings with rt. The
// stateful module's host functions allocate values through the series and
// strings ProgramStates, so both must be supplied.
func NewHost(
	ctx context.Context,
	rt wazero.Runtime,
	seriesState *series.ProgramState,
	stringsState *strings.ProgramState,
) (*Host, error) {
	h := &Host{
		series:      seriesState,
		strings:     stringsState,
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
	if rt == nil {
		return h, nil
	}
	builder := rt.NewHostModuleBuilder(Name)
	bindScalarI32[uint8](builder, h, h.stateU8, "u8")
	bindScalarI32[uint16](builder, h, h.stateU16, "u16")
	bindScalarI32[uint32](builder, h, h.stateU32, "u32")
	bindScalarI32[int8](builder, h, h.stateI8, "i8")
	bindScalarI32[int16](builder, h, h.stateI16, "i16")
	bindScalarI32[int32](builder, h, h.stateI32, "i32")
	bindScalarI64[uint64](builder, h, h.stateU64, "u64")
	bindScalarI64[int64](builder, h, h.stateI64, "i64")
	bindScalarF32(builder, h)
	bindScalarF64(builder, h)
	bindStr(builder, h)
	bindSeries(builder, h, "u8")
	bindSeries(builder, h, "u16")
	bindSeries(builder, h, "u32")
	bindSeries(builder, h, "u64")
	bindSeries(builder, h, "i8")
	bindSeries(builder, h, "i16")
	bindSeries(builder, h, "i32")
	bindSeries(builder, h, "i64")
	bindSeries(builder, h, "f32")
	bindSeries(builder, h, "f64")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return h, nil
}

var numConstraint = types.NumericConstraint()

// Name is the module name.
const Name = "stateful"

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the stateful module only (no bare globals).
func NewSymbols() []*symbol.Symbol {
	mod := &symbol.Symbol{Name: Name, Kind: symbol.KindModule, Internal: true}
	mod.AddChild(
		symbol.InternalHostFunc(
			"load",
			types.Params{{Name: "id", Type: types.I32()}, {Name: "init", Type: types.Variable("T", &numConstraint)}},
			types.Params{{Name: "value", Type: types.Variable("T", &numConstraint)}},
		),
		symbol.InternalHostFunc(
			"store",
			types.Params{{Name: "id", Type: types.I32()}, {Name: "value", Type: types.Variable("T", &numConstraint)}},
			nil,
		),
		symbol.InternalHostFunc(
			"load_series",
			types.Params{{Name: "id", Type: types.I32()}, {Name: "init", Type: types.I32()}},
			types.Params{{Name: "handle", Type: types.I32()}},
		),
		symbol.InternalHostFunc(
			"store_series",
			types.Params{{Name: "id", Type: types.I32()}, {Name: "handle", Type: types.I32()}},
			nil,
		),
	)
	return []*symbol.Symbol{mod}
}

func (h *Host) Create(_ context.Context, _ node.Config) (node.Node, error) {
	return nil, query.ErrNotFound
}

type i32Compatible interface {
	uint8 | uint16 | uint32 | int8 | int16 | int32
}

func bindScalarI32[T i32Compatible](
	builder wazero.HostModuleBuilder,
	h *Host,
	store map[string]map[uint32]T,
	suffix string,
) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, initValue uint32) uint32 {
			key := h.currentNodeKey
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
		}).Export("load_" + suffix)
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, value uint32) {
			key := h.currentNodeKey
			inner, ok := store[key]
			if !ok {
				inner = make(map[uint32]T)
				store[key] = inner
			}
			inner[varID] = T(value)
		}).Export("store_" + suffix)
}

type i64Compatible interface {
	uint64 | int64
}

func bindScalarI64[T i64Compatible](
	builder wazero.HostModuleBuilder,
	h *Host,
	store map[string]map[uint32]T,
	suffix string,
) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, initValue uint64) uint64 {
			key := h.currentNodeKey
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
		}).Export("load_" + suffix)
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, value uint64) {
			key := h.currentNodeKey
			inner, ok := store[key]
			if !ok {
				inner = make(map[uint32]T)
				store[key] = inner
			}
			inner[varID] = T(value)
		}).Export("store_" + suffix)
}

func bindScalarF32(builder wazero.HostModuleBuilder, h *Host) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, initValue float32) float32 {
			key := h.currentNodeKey
			inner, ok := h.stateF32[key]
			if !ok {
				inner = make(map[uint32]float32)
				h.stateF32[key] = inner
			}
			if value, ok := inner[varID]; ok {
				return value
			}
			inner[varID] = initValue
			return initValue
		}).Export("load_f32")
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, value float32) {
			key := h.currentNodeKey
			inner, ok := h.stateF32[key]
			if !ok {
				inner = make(map[uint32]float32)
				h.stateF32[key] = inner
			}
			inner[varID] = value
		}).Export("store_f32")
}

func bindScalarF64(builder wazero.HostModuleBuilder, h *Host) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, initValue float64) float64 {
			key := h.currentNodeKey
			inner, ok := h.stateF64[key]
			if !ok {
				inner = make(map[uint32]float64)
				h.stateF64[key] = inner
			}
			if value, ok := inner[varID]; ok {
				return value
			}
			inner[varID] = initValue
			return initValue
		}).Export("load_f64")
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, value float64) {
			key := h.currentNodeKey
			inner, ok := h.stateF64[key]
			if !ok {
				inner = make(map[uint32]float64)
				h.stateF64[key] = inner
			}
			inner[varID] = value
		}).Export("store_f64")
}

func bindStr(builder wazero.HostModuleBuilder, h *Host) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, initHandle uint32) uint32 {
			key := h.currentNodeKey
			inner, ok := h.stateString[key]
			if !ok {
				inner = make(map[uint32]string)
				h.stateString[key] = inner
			}
			if str, ok := inner[varID]; ok {
				return h.strings.Create(str)
			}
			if initStr, ok := h.strings.Get(initHandle); ok {
				inner[varID] = initStr
			}
			return initHandle
		}).Export("load_str")
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, handle uint32) {
			str, ok := h.strings.Get(handle)
			if !ok {
				return
			}
			key := h.currentNodeKey
			inner, ok := h.stateString[key]
			if !ok {
				inner = make(map[uint32]string)
				h.stateString[key] = inner
			}
			inner[varID] = str
		}).Export("store_str")
}

func bindSeries(builder wazero.HostModuleBuilder, h *Host, suffix string) {
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, initHandle uint32) uint32 {
			key := h.currentNodeKey
			inner, ok := h.stateSeries[key]
			if !ok {
				inner = make(map[uint32]telem.Series)
				h.stateSeries[key] = inner
			}
			if s, ok := inner[varID]; ok {
				return h.series.Store(s)
			}
			if initS, ok := h.series.Get(initHandle); ok {
				inner[varID] = initS.DeepCopy()
			}
			return initHandle
		}).Export("load_series_" + suffix)
	builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, varID uint32, handle uint32) {
			key := h.currentNodeKey
			inner, ok := h.stateSeries[key]
			if !ok {
				inner = make(map[uint32]telem.Series)
				h.stateSeries[key] = inner
			}
			if s, ok := h.series.Get(handle); ok {
				inner[varID] = s.DeepCopy()
			}
		}).Export("store_series_" + suffix)
}
