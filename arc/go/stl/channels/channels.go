// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channels

import (
	"context"
	"slices"

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

// Name is the module name.
const Name = "channels"

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the channels module plus `on` and `write` as bare globals so
// flow-mode programs can reference them without an import.
func NewSymbols() []*symbol.Symbol {
	numConstraint := new(types.NumericConstraint())
	mod := &symbol.Symbol{Name: Name, Kind: symbol.KindModule, Internal: true}
	mod.AddChild(
		symbol.InternalHostFunc(
			"read",
			types.Params{{Name: "ch", Type: types.I32()}},
			types.Params{{Name: "value", Type: types.Variable("T", numConstraint)}},
		),
		symbol.InternalHostFunc(
			"write",
			types.Params{
				{Name: "ch", Type: types.I32()},
				{Name: "value", Type: types.Variable("T", numConstraint)},
			},
			nil,
		),
	)
	return []*symbol.Symbol{
		mod,
		{
			Name: "on",
			Kind: symbol.KindFunction,
			Exec: symbol.ExecFlow,
			Type: types.Function(types.FunctionProperties{
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.Variable("T", nil)}},
				Inputs:  types.Params{{Name: "channel", Type: types.ReadChan(types.Variable("T", nil))}},
			}),
			Trigger: symbol.TriggerOnly,
		},
		{
			Name: "write",
			Kind: symbol.KindFunction,
			Exec: symbol.ExecFlow,
			Type: types.Function(types.FunctionProperties{
				Inputs: types.Params{
					{Name: ir.DefaultInputParam, Type: types.Variable("T", nil)},
					{Name: "channel", Type: types.WriteChan(types.Variable("T", nil))},
				},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.U8()}},
			}),
			Trigger: symbol.TriggerInput(ir.DefaultInputParam),
		},
	}
}

// Host is the runtime host-side support for the channels module: it
// registers WASM host bindings (read/write per type) and acts as the node
// factory for `on` (source) and `write` (sink) flow nodes.
type Host struct {
	state   *ProgramState
	strings *strings.ProgramState
}

// NewHost registers the channels module's WASM host bindings with rt. cs
// is the channels ProgramState; stringState is the strings ProgramState
// (used by the read_str / write_str bindings).
func NewHost(
	ctx context.Context,
	rt wazero.Runtime,
	cs *ProgramState,
	stringState *strings.ProgramState,
) (*Host, error) {
	h := &Host{state: cs, strings: stringState}
	if rt == nil {
		return h, nil
	}
	builder := rt.NewHostModuleBuilder(Name)
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
	return h, nil
}

func (h *Host) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	isSource := cfg.Node.Type == "on"
	isSink := cfg.Node.Type == "write"
	if !isSource && !isSink {
		return nil, query.ErrNotFound
	}
	var inputs nodeInputs
	if err := schema.Parse(cfg.Node.Inputs.ValueMap(), &inputs); err != nil {
		return nil, err
	}
	if isSource {
		return &source{
			State:   cfg.State,
			key:     inputs.Channel,
			state:   h.state,
			varKind: cfg.Node.VarKind,
		}, nil
	}
	inputIdx, err := cfg.State.ResolveInput(ir.DefaultInputParam)
	if err != nil {
		return nil, err
	}
	return &sink{
		State:    cfg.State,
		state:    h.state,
		key:      inputs.Channel,
		varKind:  cfg.Node.VarKind,
		inputIdx: inputIdx,
	}, nil
}

var schema = zyn.Object(map[string]zyn.Schema{
	"channel": zyn.Uint32().Coerce(),
})

type nodeInputs struct {
	Channel uint32 `json:"channel"`
}

// backsInternalChannel reports whether kind k is backed by a program-local channel.
func backsInternalChannel(k ir.VarKind) bool {
	return k == ir.VarKindConstant || k == ir.VarKindReactive
}

type source struct {
	*node.State
	state         *ProgramState
	key           uint32
	varKind       ir.VarKind
	highWaterMark telem.Alignment
	clock         telem.MonoClock
}

func (s *source) Init(node.Context) {}

// Reset advances the high water mark to the current channel alignment,
// ensuring that when a stage is (re-)activated it only responds to
// data that arrives after activation rather than stale pre-existing data.
func (s *source) Reset() {
	s.State.Reset()
	if s.varKind == ir.VarKindConstant {
		return
	}
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
	// A constant variable reflects its current value: emit the newest unread series.
	if s.varKind == ir.VarKindConstant {
		for i := range slices.Backward(data.Series) {
			if data.Series[i].AlignmentBounds().Lower >= s.highWaterMark {
				s.emitSeries(ctx, data, indexData, i)
				return
			}
		}
		return
	}
	// A channel, alias, or reactive read streams: emit the oldest unread series.
	for i := range data.Series {
		if data.Series[i].AlignmentBounds().Lower >= s.highWaterMark {
			s.emitSeries(ctx, data, indexData, i)
			return
		}
	}
}

// emitSeries emits data.Series[i] and its aligned timestamp on output 0 and
// advances the high water mark; it is a no-op when no aligned timestamp exists.
func (s *source) emitSeries(ctx node.Context, data, indexData telem.MultiSeries, i int) {
	ser := data.Series[i]
	var timeSeries telem.Series
	if indexData.DataType() == telem.UnknownT {
		timeSeries = telem.Arrange(s.clock.Now(), int(ser.Len()), 1*telem.NanosecondTS)
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
	s.highWaterMark = ser.AlignmentBounds().Upper
	ctx.MarkChanged(0)
}

type sink struct {
	*node.State
	state    *ProgramState
	key      uint32
	varKind  ir.VarKind
	inputIdx int
}

func (s *sink) Next(ctx node.Context) {
	if !s.RefreshInputs() {
		return
	}
	data := s.Input(s.inputIdx)
	time := s.InputTime(s.inputIdx)
	if data.Len() == 0 {
		return
	}
	if backsInternalChannel(s.varKind) {
		s.state.appendVarRead(s.key, data.DeepCopy())
		ctx.SetDeadline(ctx.Elapsed)
	} else {
		s.state.writeChannel(s.key, data, time)
	}
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
