// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"

	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/stl/constant"
	"github.com/synnaxlabs/arc/stl/control"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	stlop "github.com/synnaxlabs/arc/stl/op"
	"github.com/synnaxlabs/arc/stl/selector"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stable"
	"github.com/synnaxlabs/arc/stl/stat"
	"github.com/synnaxlabs/arc/stl/stateful"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/stl/wasm"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
)

// runtimeHarness provides a full end-to-end test harness that compiles Arc source
// code and executes it through the scheduler with real wasm nodes.
type runtimeHarness struct {
	scheduler    *scheduler.Scheduler
	channelState *channel.ProgramState
	nodeState    *node.ProgramState
	wasmRT       wazero.Runtime
	closers      []func(context.Context) error
	alignment    telem.Alignment
}

func newRuntimeHarness(
	ctx context.Context,
	source string,
	resolver symbol.Resolver,
	channelDigests ...channel.Digest,
) *runtimeHarness {
	compileResolver := symbol.CompoundResolver{stl.SymbolResolver}
	if resolver != nil {
		compileResolver = append(compileResolver, resolver)
	}

	prog := MustSucceed(arc.CompileText(ctx, arc.Text{Raw: source}, arc.WithResolver(compileResolver)))

	nodeState := node.New(prog.IR)
	channelState := channel.NewProgramState(channelDigests)
	seriesState := series.NewProgramState()
	stringsState := stlstrings.NewProgramState()
	controlState := &control.ProgramState{}

	wasmRT := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())

	timeMod := MustSucceed(time.NewModule(ctx, wasmRT))
	channelMod := MustSucceed(channel.NewModule(ctx, channelState, stringsState, wasmRT))
	statefulMod := MustSucceed(stateful.NewModule(ctx, seriesState, stringsState, wasmRT))
	MustSucceed(series.NewModule(ctx, seriesState, wasmRT))
	stringsMod := MustSucceed(stlstrings.NewModule(ctx, stringsState, wasmRT, nil))
	MustSucceed(stlmath.NewModule(ctx, wasmRT))
	errorsMod := MustSucceed(stlerrors.NewModule(ctx, nil, wasmRT))

	factory := node.CompoundFactory{
		channelMod,
		statefulMod,
		timeMod,
		selector.NewModule(),
		constant.NewModule(),
		stlop.NewModule(),
		stable.NewModule(),
		control.NewModule(controlState),
		&stat.Module{},
	}

	h := &runtimeHarness{
		channelState: channelState,
		nodeState:    nodeState,
		wasmRT:       wasmRT,
	}

	if len(prog.WASM) > 0 {
		guest := MustSucceed(wasmRT.Instantiate(ctx, prog.WASM))
		stringsMod.SetMemory(guest.Memory())
		errorsMod.SetMemory(guest.Memory())
		h.closers = append(h.closers, func(ctx context.Context) error {
			return guest.Close(ctx)
		})
		factory = append(factory, &wasm.Module{
			Module:        guest,
			Memory:        guest.Memory(),
			Strings:       stringsState,
			NodeKeySetter: statefulMod,
		})
	}

	nodes := make(map[string]node.Node)
	for _, irNode := range prog.Nodes {
		n := MustSucceed(factory.Create(ctx, node.Config{
			Node:    irNode,
			Program: prog,
			State:   nodeState.Node(irNode.Key),
		}))
		nodes[irNode.Key] = n
	}

	tolerance := time.CalculateTolerance(timeMod.BaseInterval)
	h.scheduler = scheduler.New(prog.IR, nodes, tolerance)

	h.closers = append(h.closers, func(ctx context.Context) error {
		return wasmRT.Close(ctx)
	})

	return h
}

func (h *runtimeHarness) Close(ctx context.Context) {
	for i := len(h.closers) - 1; i >= 0; i-- {
		Expect(h.closers[i](ctx)).To(Succeed())
	}
}

func (h *runtimeHarness) Tick(ctx context.Context, elapsed telem.TimeSpan) {
	h.scheduler.Next(ctx, elapsed, node.ReasonTimerTick)
}

func (h *runtimeHarness) Ingest(channelKey uint32, data telem.Series) {
	data.Alignment = h.alignment
	h.alignment += telem.Alignment(data.Len())
	fr := telem.Frame[uint32]{}
	fr = fr.Append(channelKey, data)
	h.channelState.Ingest(fr)
}

func (h *runtimeHarness) IngestIndexed(indexKey uint32, timestamps telem.Series, dataKey uint32, data telem.Series) {
	timestamps.Alignment = h.alignment
	data.Alignment = h.alignment
	h.alignment += telem.Alignment(data.Len())
	fr := telem.Frame[uint32]{}
	fr = fr.Append(indexKey, timestamps)
	fr = fr.Append(dataKey, data)
	h.channelState.Ingest(fr)
}

func (h *runtimeHarness) Flush() (telem.Frame[uint32], bool) {
	return h.channelState.Flush(telem.Frame[uint32]{})
}

func (h *runtimeHarness) Output(nodeKey string, paramIdx int) telem.Series {
	return *h.nodeState.Node(nodeKey).Output(paramIdx)
}

func (h *runtimeHarness) OutputTime(nodeKey string, paramIdx int) telem.Series {
	return *h.nodeState.Node(nodeKey).OutputTime(paramIdx)
}

type channelDef struct {
	dt types.Type
	id int
}

func channelSymbols(channels map[string]channelDef) symbol.MapResolver {
	r := symbol.MapResolver{}
	for name, ch := range channels {
		r[name] = symbol.Symbol{
			Name: name,
			Kind: symbol.KindChannel,
			Type: types.Chan(ch.dt),
			ID:   ch.id,
		}
	}
	return r
}
