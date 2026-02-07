// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

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

var numConstraint = types.NumericConstraint()

var symResolver = &symbol.ModuleResolver{
	Name: "channel",
	Members: symbol.MapResolver{
		"read": {
			Name: "read",
			Type: types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "ch", Type: types.I32()}},
				Outputs: types.Params{{Name: "value", Type: types.Variable("T", &numConstraint)}},
			}),
		},
		"write": {
			Name: "write",
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "ch", Type: types.I32()}, {Name: "value", Type: types.Variable("T", &numConstraint)}},
			}),
		},
	},
}

type Module struct {
	channel *state.ChannelState
	strings *state.StringHandleStore
}

func NewModule(cs *state.ChannelState, ss *state.StringHandleStore) *Module {
	return &Module{channel: cs, strings: ss}
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
	cs := m.channel
	bindI32[uint8](rt, cs, "u8", telem.Uint8T)
	bindI32[uint16](rt, cs, "u16", telem.Uint16T)
	bindI32[uint32](rt, cs, "u32", telem.Uint32T)
	bindI32[int8](rt, cs, "i8", telem.Int8T)
	bindI32[int16](rt, cs, "i16", telem.Int16T)
	bindI32[int32](rt, cs, "i32", telem.Int32T)
	bindI64[uint64](rt, cs, "u64")
	bindI64[int64](rt, cs, "i64")
	bindF32(rt, cs)
	bindF64(rt, cs)
	bindStr(rt, cs, m.strings)
	return nil
}

type i32Compatible interface {
	uint8 | uint16 | uint32 | int8 | int16 | int32
}

func bindI32[T i32Compatible](
	rt stl.HostRuntime,
	cs *state.ChannelState,
	suffix string,
	_ telem.DataType,
) {
	stl.MustExport(rt, "channel", "read_"+suffix,
		func(_ context.Context, chID uint32) uint32 {
			series, ok := cs.ReadValue(chID)
			if !ok || series.Len() == 0 {
				return 0
			}
			return uint32(telem.ValueAt[T](series, -1))
		})
	stl.MustExport(rt, "channel", "write_"+suffix,
		func(_ context.Context, chID uint32, val uint32) {
			cs.WriteValue(chID, telem.NewSeriesV[T](T(val)))
		})
}

type i64Compatible interface {
	uint64 | int64
}

func bindI64[T i64Compatible](
	rt stl.HostRuntime,
	cs *state.ChannelState,
	suffix string,
) {
	stl.MustExport(rt, "channel", "read_"+suffix,
		func(_ context.Context, chID uint32) uint64 {
			series, ok := cs.ReadValue(chID)
			if !ok || series.Len() == 0 {
				return 0
			}
			return uint64(telem.ValueAt[T](series, -1))
		})
	stl.MustExport(rt, "channel", "write_"+suffix,
		func(_ context.Context, chID uint32, val uint64) {
			cs.WriteValue(chID, telem.NewSeriesV[T](T(val)))
		})
}

func bindF32(rt stl.HostRuntime, cs *state.ChannelState) {
	stl.MustExport(rt, "channel", "read_f32",
		func(_ context.Context, chID uint32) float32 {
			series, ok := cs.ReadValue(chID)
			if !ok || series.Len() == 0 {
				return 0
			}
			return telem.ValueAt[float32](series, -1)
		})
	stl.MustExport(rt, "channel", "write_f32",
		func(_ context.Context, chID uint32, val float32) {
			cs.WriteValue(chID, telem.NewSeriesV[float32](val))
		})
}

func bindF64(rt stl.HostRuntime, cs *state.ChannelState) {
	stl.MustExport(rt, "channel", "read_f64",
		func(_ context.Context, chID uint32) float64 {
			series, ok := cs.ReadValue(chID)
			if !ok || series.Len() == 0 {
				return 0
			}
			return telem.ValueAt[float64](series, -1)
		})
	stl.MustExport(rt, "channel", "write_f64",
		func(_ context.Context, chID uint32, val float64) {
			cs.WriteValue(chID, telem.NewSeriesV[float64](val))
		})
}

func bindStr(rt stl.HostRuntime, cs *state.ChannelState, ss *state.StringHandleStore) {
	stl.MustExport(rt, "channel", "read_str",
		func(_ context.Context, chID uint32) uint32 {
			series, ok := cs.ReadValue(chID)
			if !ok || series.Len() == 0 {
				return 0
			}
			strings := telem.UnmarshalStrings(series.Data)
			if len(strings) == 0 {
				return 0
			}
			return ss.Create(strings[len(strings)-1])
		})
	stl.MustExport(rt, "channel", "write_str",
		func(_ context.Context, chID uint32, handle uint32) {
			str, ok := ss.Get(handle)
			if !ok {
				return
			}
			data := telem.MarshalStrings([]string{str}, telem.StringT)
			cs.WriteValue(chID, telem.Series{
				DataType: telem.StringT,
				Data:     data,
			})
		})
}

var _ stl.Module = (*Module)(nil)
