// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package interval

import (
	"context"
	"time"

	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

const (
	intervalSymbolName = "interval"
	periodParam        = "period"
	initialDelayParam  = "initial_delay"
	tickOutput         = "tick"
	timestampOutput    = "timestamp"
	elapsedOutput      = "elapsed"
)

var (
	intervalSymbol = symbol.Symbol{
		Name: intervalSymbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: &types.Params{
				Keys:   []string{periodParam, initialDelayParam},
				Values: []types.Type{types.TimeSpan(), types.TimeSpan()},
			},
			Inputs: &types.Params{}, // No inputs
			Outputs: &types.Params{
				Keys:   []string{tickOutput, timestampOutput, elapsedOutput},
				Values: []types.Type{types.U64(), types.TimeStamp(), types.TimeSpan()},
			},
		}),
	}
	SymbolResolver = symbol.MapResolver{
		intervalSymbolName: intervalSymbol,
	}
)

// Node is a runtime implementation of the interval builtin stage.
type Node struct {
	key       string
	timeWheel *Wheel
	state     *state.Node
}

func (n *Node) Init(node.Context) {
	// Nothing to do - time wheel manages the interval
}

func (n *Node) Next(ctx node.Context) {
	tick, timestamp, elapsed, ok := n.timeWheel.GetState(n.key)
	if !ok {
		return
	}

	// Interval nodes have no inputs, so RefreshInputs always returns true
	n.state.RefreshInputs()

	// Write outputs to state
	// Output 0: tick (u64)
	tickData := n.state.Output(0)
	tickTime := n.state.OutputTime(0)
	tickData.Resize(1)
	tickTime.Resize(1)
	telem.MarshalUint64(tickData.Data, tick)
	telem.MarshalTimeStamp(tickTime.Data, timestamp)

	// Output 1: timestamp (timestamp)
	timestampData := n.state.Output(1)
	timestampTime := n.state.OutputTime(1)
	timestampData.Resize(1)
	timestampTime.Resize(1)
	telem.MarshalTimeStamp(timestampData.Data, timestamp)
	telem.MarshalTimeStamp(timestampTime.Data, timestamp)

	// Output 2: elapsed (timespan)
	elapsedData := n.state.Output(2)
	elapsedTime := n.state.OutputTime(2)
	elapsedData.Resize(1)
	elapsedTime.Resize(1)
	telem.MarshalInt64(elapsedData.Data, int64(elapsed))
	telem.MarshalTimeStamp(elapsedTime.Data, timestamp)

	// Mark all outputs as changed
	ctx.MarkChanged(tickOutput)
	ctx.MarkChanged(timestampOutput)
	ctx.MarkChanged(elapsedOutput)
}

type factory struct {
	timeWheel *Wheel
}

type NodeConfig = node.Config

func (f *factory) Create(_ context.Context, cfg NodeConfig) (node.Node, error) {
	if cfg.Node.Type != intervalSymbolName {
		return nil, query.NotFound
	}

	// Extract period from config
	periodVal, ok := cfg.Node.ConfigValues[periodParam]
	if !ok {
		return nil, query.NotFound
	}
	period := time.Duration(periodVal.(telem.TimeSpan))

	// Extract optional initial_delay
	var initialDelay time.Duration
	if delayVal, ok := cfg.Node.ConfigValues[initialDelayParam]; ok {
		initialDelay = time.Duration(delayVal.(telem.TimeSpan))
	}

	// Register with time wheel using node key
	f.timeWheel.Register(cfg.Node.Key, period, initialDelay)

	return &Node{
		key:       cfg.Node.Key,
		timeWheel: f.timeWheel,
		state:     cfg.State,
	}, nil
}

func NewFactory(wheel *Wheel) *factory {
	return &factory{
		timeWheel: wheel,
	}
}
