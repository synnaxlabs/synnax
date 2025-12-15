// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package stage provides the StageEntry node for Arc runtime stage transitions.
// StageEntry nodes listen for activation signals (u8 value of 1) and trigger
// stage transitions via a callback to the scheduler.
package stage

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

const (
	symName             = "stage_entry"
	sequenceConfigParam = "sequence"
	stageConfigParam    = "stage"
)

var (
	sym = symbol.Symbol{
		Name: symName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs: types.Params{{Name: ir.DefaultInputParam, Type: types.U8()}},
		}),
	}
	// SymbolResolver provides the stage_entry symbol for the Arc analyzer.
	SymbolResolver = symbol.MapResolver{symName: sym}
)

// ActivateCallback is called when a stage_entry node receives an activation signal.
// Parameters are (sequenceName, stageName).
type ActivateCallback func(sequenceName, stageName string)

// SharedCallback allows callbacks to be set after node creation.
// All StageEntry nodes share the same callback instance, which can be set
// after the scheduler is created.
type SharedCallback struct {
	Callback ActivateCallback
}

// Invoke calls the callback if it is set.
func (s *SharedCallback) Invoke(sequenceName, stageName string) {
	if s.Callback != nil {
		s.Callback(sequenceName, stageName)
	}
}

// StageEntry is a node that triggers stage transitions when it receives
// an activation signal (input value of u8(1)).
type StageEntry struct {
	state          *state.Node
	sequenceName   string
	stageName      string
	sharedCallback *SharedCallback
}

// Init performs one-time initialization (no-op for StageEntry).
func (s *StageEntry) Init(_ node.Context) {}

// Next checks for activation signals and triggers stage transitions.
func (s *StageEntry) Next(_ node.Context) {
	// Check if we have new input
	if !s.state.RefreshInputs() {
		return
	}

	input := s.state.Input(0)
	if input.Len() == 0 {
		return
	}

	// Activation signal is a u8 with value 1
	signal := telem.ValueAt[uint8](input, 0)
	if signal == 1 && s.sharedCallback != nil {
		s.sharedCallback.Invoke(s.sequenceName, s.stageName)
	}
}

// Factory creates StageEntry nodes for "stage_entry" type nodes in the IR.
type Factory struct {
	// sharedCallback is shared by all StageEntry nodes created by this factory.
	sharedCallback *SharedCallback
}

// NewFactory creates a new StageEntry factory with a shared callback.
func NewFactory() *Factory {
	return &Factory{sharedCallback: &SharedCallback{}}
}

// Create constructs a StageEntry node from the given configuration.
// Returns query.NotFound if the node type is not "stage_entry".
func (f *Factory) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symName {
		return nil, query.NotFound
	}

	seqParam, seqOk := cfg.Node.Config.Get(sequenceConfigParam)
	stageParam, stageOk := cfg.Node.Config.Get(stageConfigParam)

	if !seqOk || !stageOk {
		return nil, errors.New("stage_entry node missing sequence or stage config")
	}

	sequenceName, ok := seqParam.Value.(string)
	if !ok {
		return nil, errors.Newf("stage_entry sequence config must be a string, got %T", seqParam.Value)
	}

	stageName, ok := stageParam.Value.(string)
	if !ok {
		return nil, errors.Newf("stage_entry stage config must be a string, got %T", stageParam.Value)
	}

	return &StageEntry{
		state:          cfg.State,
		sequenceName:   sequenceName,
		stageName:      stageName,
		sharedCallback: f.sharedCallback,
	}, nil
}

// SetActivateCallback sets the callback to be invoked when any StageEntry node
// created by this factory receives an activation signal. This should be called
// after the scheduler is created to wire up stage transitions.
func (f *Factory) SetActivateCallback(callback ActivateCallback) {
	f.sharedCallback.Callback = callback
}
