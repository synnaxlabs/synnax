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
	"slices"

	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/stl/constant"
	"github.com/synnaxlabs/arc/stl/control"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	stlop "github.com/synnaxlabs/arc/stl/op"
	"github.com/synnaxlabs/arc/stl/selector"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stable"
	"github.com/synnaxlabs/arc/stl/stateful"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/stl/variable"
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
	scheduler      *scheduler.Scheduler
	channelState   *channels.ProgramState
	authorityState *control.ProgramState
	nodeState      *node.ProgramState
	wasmRT         wazero.Runtime
	closers        []func(context.Context) error
	alignment      telem.Alignment
}

func newRuntimeHarness(
	ctx context.Context,
	source string,
	channelSyms []symbol.Symbol,
	channelDigests ...channels.Digest,
) *runtimeHarness {
	stlSyms := stl.NewSymbols()
	ambient := make([]*symbol.Symbol, 0, len(stlSyms)+len(channelSyms))
	ambient = append(ambient, stlSyms...)
	for i := range channelSyms {
		s := channelSyms[i]
		ambient = append(ambient, &s)
	}
	root := symbol.NewRoot(nil, ambient)
	prog := MustSucceed(arc.CompileText(ctx, arc.Text{Raw: source}, root))

	nodeState := node.New(prog.IR)
	channelState := channels.NewProgramState(channelDigests)
	seriesState := series.NewProgramState()
	stringsState := stlstrings.NewProgramState()
	authorityState := &control.ProgramState{}

	wasmRT := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())

	timeMod := MustSucceed(time.NewHost(ctx, wasmRT))
	channelMod := MustSucceed(channels.NewHost(ctx, wasmRT, channelState, stringsState))
	statefulMod := MustSucceed(stateful.NewHost(ctx, wasmRT, seriesState, stringsState))
	MustSucceed(series.NewHost(ctx, wasmRT, seriesState))
	stringsMod := MustSucceed(stlstrings.NewHost(ctx, wasmRT, stringsState, nil))
	mathMod := MustSucceed(stlmath.NewHost(ctx, wasmRT))
	errorsMod := MustSucceed(stlerrors.NewHost(ctx, wasmRT, nil))

	factory := node.CompoundFactory{
		channelMod,
		statefulMod,
		timeMod,
		selector.NewHost(),
		constant.NewHost(),
		variable.NewHost(),
		stlop.NewHost(),
		stable.NewHost(),
		control.NewHost(authorityState),
		mathMod,
	}

	h := &runtimeHarness{
		channelState:   channelState,
		authorityState: authorityState,
		nodeState:      nodeState,
		wasmRT:         wasmRT,
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
	for _, v := range slices.Backward(h.closers) {
		Expect(v(ctx)).To(Succeed())
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

func (h *runtimeHarness) IngestIndexed(
	indexKey uint32,
	timestamps telem.Series,
	dataKey uint32,
	data telem.Series,
) {
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

// FlushAuthority drains and returns all authority changes buffered by
// set_authority nodes this cycle. Tests assert on the returned slice to
// verify authority semantics that aren't observable via channel writes.
func (h *runtimeHarness) FlushAuthority() []control.AuthorityChange {
	return h.authorityState.Flush()
}

type channelDef struct {
	dt types.Type
	id int
}

// channelSymbols builds a flat list of channel symbols from a map of
// (name → typed-id) entries. Used by runtime tests to attach a fixed set
// of channels to a test root's ambient prelude.
func channelSymbols(channels map[string]channelDef) []symbol.Symbol {
	r := make([]symbol.Symbol, 0, len(channels))
	for name, ch := range channels {
		r = append(r, symbol.Symbol{
			Name: name,
			Kind: symbol.KindChannel,
			Type: types.Chan(ch.dt),
			ID:   ch.id,
		})
	}
	return r
}
