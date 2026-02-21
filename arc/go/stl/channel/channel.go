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

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"
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

var nodeResolver = symbol.MapResolver{
	"on": {
		Name: "on",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.Variable("T", nil)}},
			Config:  types.Params{{Name: "channel", Type: types.ReadChan(types.Variable("T", nil))}},
		}),
	},
	"write": {
		Name: "write",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: ir.DefaultInputParam, Type: types.Variable("T", nil)}},
			Config: types.Params{{Name: "channel", Type: types.WriteChan(types.Variable("T", nil))}},
		}),
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
	if sym, err := symResolver.Resolve(ctx, name); err == nil {
		return sym, nil
	}
	return nodeResolver.Resolve(ctx, name)
}

func (m *Module) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	r1, _ := symResolver.Search(ctx, term)
	r2, _ := nodeResolver.Search(ctx, term)
	return append(r1, r2...), nil
}

func (m *Module) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	isSource := cfg.Node.Type == "on"
	isSink := cfg.Node.Type == "write"
	if !isSource && !isSink {
		return nil, query.ErrNotFound
	}
	var nodeCfg config
	if err := schema.Parse(cfg.Node.Config.ValueMap(), &nodeCfg); err != nil {
		return nil, err
	}
	if isSource {
		return &source{Node: cfg.State, key: nodeCfg.Channel}, nil
	}
	return &sink{Node: cfg.State, key: nodeCfg.Channel}, nil
}

func (m *Module) BindTo(rt stl.HostRuntime) error {
	if m.channel == nil {
		return nil
	}
	cs := m.channel
	bindI32[uint8](rt, cs, "u8")
	bindI32[uint16](rt, cs, "u16")
	bindI32[uint32](rt, cs, "u32")
	bindI32[int8](rt, cs, "i8")
	bindI32[int16](rt, cs, "i16")
	bindI32[int32](rt, cs, "i32")
	bindI64[uint64](rt, cs, "u64")
	bindI64[int64](rt, cs, "i64")
	bindF32(rt, cs)
	bindF64(rt, cs)
	bindStr(rt, cs, m.strings)
	return nil
}

var schema = zyn.Object(map[string]zyn.Schema{
	"channel": zyn.Uint32().Coerce(),
})

type config struct {
	Channel uint32 `json:"channel"`
}

type source struct {
	*state.Node
	key           uint32
	highWaterMark telem.Alignment
}

func (s *source) Init(node.Context) {}

func (s *source) Next(ctx node.Context) {
	data, indexData, ok := s.ReadSeries(s.key)
	if !ok {
		return
	}
	for i, ser := range data.Series {
		ab := ser.AlignmentBounds()
		if ab.Lower >= s.highWaterMark {
			var timeSeries telem.Series
			if indexData.DataType() == telem.UnknownT {
				timeSeries = telem.Arrange(
					telem.Now(),
					int(data.Len()),
					1*telem.NanosecondTS,
				)
				timeSeries.Alignment = ser.Alignment
			} else if len(indexData.Series) > i {
				timeSeries = indexData.Series[i]
			} else {
				return
			}
			if timeSeries.Alignment != ser.Alignment {
				return
			}
			*s.Output(0) = ser
			*s.OutputTime(0) = timeSeries
			s.highWaterMark = ab.Upper
			ctx.MarkChanged(ir.DefaultOutputParam)
			return
		}
	}
}

type sink struct {
	*state.Node
	key uint32
}

func (s *sink) Next(node.Context) {
	if !s.RefreshInputs() {
		return
	}
	data := s.Input(0)
	time := s.InputTime(0)
	if data.Len() == 0 {
		return
	}
	s.WriteSeries(s.key, data, time)
}

type i32Compatible interface {
	uint8 | uint16 | uint32 | int8 | int16 | int32
}

func bindI32[T i32Compatible](
	rt stl.HostRuntime,
	cs *state.ChannelState,
	suffix string,
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
			strings := telem.UnmarshalSeries[string](series)
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
			cs.WriteValue(chID, telem.NewSeriesV[string](str))
		})
}

var _ stl.Module = (*Module)(nil)
