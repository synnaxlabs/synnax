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
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"
	"github.com/tetratelabs/wazero"
)

var numConstraint = types.NumericConstraint()

var userSymbols = symbol.MapResolver{
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
			Inputs:  types.Params{{Name: ir.DefaultInputParam, Type: types.Variable("T", nil)}},
			Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			Config:  types.Params{{Name: "channel", Type: types.WriteChan(types.Variable("T", nil))}},
		}),
	},
}

var hostSymbols = symbol.MapResolver{
	"read": {
		Name:     "read",
		Kind:     symbol.KindFunction,
		Internal: true,
		Type: types.Function(types.FunctionProperties{
			Inputs:  types.Params{{Name: "ch", Type: types.I32()}},
			Outputs: types.Params{{Name: "value", Type: types.Variable("T", &numConstraint)}},
		}),
	},
	"write": {
		Name:     "write",
		Kind:     symbol.KindFunction,
		Internal: true,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: "ch", Type: types.I32()}, {Name: "value", Type: types.Variable("T", &numConstraint)}},
		}),
	},
}

var SymbolResolver = symbol.CompoundResolver{
	userSymbols,
	&symbol.ModuleResolver{Name: "channel", Members: hostSymbols},
}

type Module struct {
	state   *ProgramState
	strings *strings.ProgramState
}

func NewModule(
	ctx context.Context,
	cs *ProgramState,
	stringState *strings.ProgramState,
	rat wazero.Runtime,
) (*Module, error) {
	mod := &Module{state: cs, strings: stringState}
	if rat == nil {
		return mod, nil
	}
	builder := rat.NewHostModuleBuilder("channel")
	builder = bindI32[uint8](builder, cs, "u8")
	builder = bindI32[uint16](builder, cs, "u16")
	builder = bindI32[uint32](builder, cs, "u32")
	builder = bindI32[int8](builder, cs, "i8")
	builder = bindI32[int16](builder, cs, "i16")
	builder = bindI32[int32](builder, cs, "i32")
	builder = bindI64[uint64](builder, cs, "u64")
	builder = bindI64[int64](builder, cs, "i64")
	builder = bindF32(builder, cs)
	builder = bindF64(builder, cs)
	builder = bindStr(builder, cs, stringState)
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return mod, nil
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
		return &source{
			State: cfg.State,
			key:   nodeCfg.Channel,
			state: m.state,
		}, nil
	}
	return &sink{State: cfg.State, state: m.state, key: nodeCfg.Channel}, nil
}

var schema = zyn.Object(map[string]zyn.Schema{
	"channel": zyn.Uint32().Coerce(),
})

type config struct {
	Channel uint32 `json:"channel"`
}

type source struct {
	*node.State
	state         *ProgramState
	key           uint32
	highWaterMark telem.Alignment
	clock         telem.MonoClock
}

func (s *source) Init(node.Context) {}

// Reset advances the high water mark to the current channel alignment,
// ensuring that when a stage is (re-)activated it only responds to
// data that arrives after activation rather than stale pre-existing data.
func (s *source) Reset() {
	s.State.Reset()
	data, _, ok := s.state.readSeries(s.key)
	if !ok || len(data.Series) == 0 {
		return
	}
	ab := data.Series[len(data.Series)-1].AlignmentBounds()
	if ab.Upper > s.highWaterMark {
		s.highWaterMark = ab.Upper
	}
}

func (s *source) Next(ctx node.Context) {
	data, indexData, ok := s.state.readSeries(s.key)
	if !ok {
		return
	}
	for i, ser := range data.Series {
		ab := ser.AlignmentBounds()
		if ab.Lower >= s.highWaterMark {
			var timeSeries telem.Series
			if indexData.DataType() == telem.UnknownT {
				timeSeries = telem.Arrange(
					s.clock.Now(),
					int(ser.Len()),
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
			ctx.MarkChanged(0)
			return
		}
	}
}

func (s *source) Outputs() []string { return []string{ir.DefaultOutputParam} }

type sink struct {
	*node.State
	state *ProgramState
	key   uint32
}

func (s *sink) Outputs() []string { return []string{ir.DefaultOutputParam} }

func (s *sink) Next(ctx node.Context) {
	if !s.RefreshInputs() {
		return
	}
	data := s.Input(0)
	time := s.InputTime(0)
	if data.Len() == 0 {
		return
	}
	s.state.writeChannel(s.key, data, time)
	lastTS := telem.ValueAt[telem.TimeStamp](time, -1)
	out := s.Output(0)
	out.Resize(1)
	telem.SetValueAt[uint8](*out, 0, 1)
	out.Alignment = data.Alignment
	out.TimeRange = data.TimeRange
	outTime := s.OutputTime(0)
	outTime.Resize(1)
	telem.SetValueAt[telem.TimeStamp](*outTime, 0, lastTS)
	outTime.Alignment = data.Alignment
	outTime.TimeRange = data.TimeRange
	ctx.MarkChanged(0)
}

type i32Compatible interface {
	uint8 | uint16 | uint32 | int8 | int16 | int32
}

func bindI32[T i32Compatible](
	builder wazero.HostModuleBuilder,
	cs *ProgramState,
	suffix string,
) wazero.HostModuleBuilder {
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, chID uint32) uint32 {
			series, ok := cs.ReadValue(chID)
			if !ok || series.Len() == 0 {
				return 0
			}
			return uint32(telem.ValueAt[T](series, -1))
		}).Export("read_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, chID uint32, val uint32) {
			appendFixedWriteSample(cs, chID, T(val))
			cs.writeIndexedTimestamp(chID)
		}).Export("write_" + suffix)
	return builder
}

type i64Compatible interface {
	uint64 | int64
}

func bindI64[T i64Compatible](
	builder wazero.HostModuleBuilder,
	cs *ProgramState,
	suffix string,
) wazero.HostModuleBuilder {
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, chID uint32) uint64 {
			series, ok := cs.ReadValue(chID)
			if !ok || series.Len() == 0 {
				return 0
			}
			return uint64(telem.ValueAt[T](series, -1))
		}).Export("read_" + suffix)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, chID uint32, val uint64) {
			appendFixedWriteSample(cs, chID, T(val))
			cs.writeIndexedTimestamp(chID)
		}).Export("write_" + suffix)
	return builder
}

func bindF32(builder wazero.HostModuleBuilder, cs *ProgramState) wazero.HostModuleBuilder {
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, chID uint32) float32 {
			series, ok := cs.ReadValue(chID)
			if !ok || series.Len() == 0 {
				return 0
			}
			return telem.ValueAt[float32](series, -1)
		}).Export("read_f32")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, chID uint32, val float32) {
			cs.WriteChannelF32(chID, val)
		}).Export("write_f32")
	return builder
}

func bindF64(builder wazero.HostModuleBuilder, cs *ProgramState) wazero.HostModuleBuilder {
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, chID uint32) float64 {
			series, ok := cs.ReadValue(chID)
			if !ok || series.Len() == 0 {
				return 0
			}
			return telem.ValueAt[float64](series, -1)
		}).Export("read_f64")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, chID uint32, val float64) {
			cs.WriteChannelF64(chID, val)
		}).Export("write_f64")
	return builder
}

func bindStr(builder wazero.HostModuleBuilder, cs *ProgramState, ss *strings.ProgramState) wazero.HostModuleBuilder {
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, chID uint32) uint32 {
			series, ok := cs.ReadValue(chID)
			if !ok || series.Len() == 0 {
				return 0
			}
			unmarshaled := telem.UnmarshalSeries[string](series)
			if len(unmarshaled) == 0 {
				return 0
			}
			return ss.Create(unmarshaled[len(unmarshaled)-1])
		}).Export("read_str")
	builder = builder.NewFunctionBuilder().
		WithFunc(func(_ context.Context, chID uint32, handle uint32) {
			str, ok := ss.Get(handle)
			if !ok {
				return
			}
			cs.writeValue(chID, telem.NewSeriesV[string](str))
		}).Export("write_str")
	return builder
}
