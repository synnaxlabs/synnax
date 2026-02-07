// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stable

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

var (
	symbolName = "stable_for"
	symbolDef  = symbol.Symbol{
		Name: symbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: types.Params{
				{Name: "duration", Type: types.TimeSpan()},
			},
			Inputs: types.Params{
				{Name: ir.DefaultInputParam, Type: types.Variable("T", nil)},
			},
			Outputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.Variable("T", nil)},
			},
		}),
	}
	SymbolResolver = symbol.MapResolver{symbolName: symbolDef}
)

type Module struct {
	Now func() telem.TimeStamp
}

var _ stl.Module = (*Module)(nil)

func NewModule() *Module {
	return &Module{Now: telem.Now}
}

func (m *Module) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	return SymbolResolver.Resolve(ctx, name)
}

func (m *Module) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	return SymbolResolver.Search(ctx, term)
}

func (m *Module) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.ErrNotFound
	}
	var cfgVals config
	if err := configSchema.Parse(cfg.Node.Config.ValueMap(), &cfgVals); err != nil {
		return nil, err
	}
	return &stableFor{Node: cfg.State, duration: cfgVals.Duration, now: m.Now}, nil
}

func (m *Module) BindTo(_ context.Context, _ stl.HostRuntime) error {
	return nil
}

type config struct {
	Duration telem.TimeSpan
}

var configSchema = zyn.Object(map[string]zyn.Schema{
	"duration": zyn.Int64().Coerce(),
})

type stableFor struct {
	*state.Node
	value       *uint8
	lastSent    *uint8
	now         func() telem.TimeStamp
	duration    telem.TimeSpan
	lastChanged telem.TimeStamp
}

func (s *stableFor) Reset() {
	s.Node.Reset()
	s.value = nil
	s.lastSent = nil
	s.lastChanged = 0
}

var _ node.Node = (*stableFor)(nil)

func (s *stableFor) Next(ctx node.Context) {
	if s.RefreshInputs() {
		inputData := s.Input(0)
		inputTime := s.InputTime(0)
		if inputData.Len() > 0 {
			for i := int64(0); i < inputData.Len(); i++ {
				currentValue := telem.ValueAt[uint8](inputData, int(i))
				currentTime := telem.ValueAt[telem.TimeStamp](inputTime, int(i))
				if s.value == nil || *s.value != currentValue {
					s.value = &currentValue
					s.lastChanged = currentTime
				}
			}
		}
	}

	if s.value == nil {
		return
	}
	currentValue := *s.value
	if telem.TimeSpan(s.now()-s.lastChanged) >= s.duration {
		if s.lastSent == nil || *s.lastSent != currentValue {
			*s.Output(0) = telem.NewSeriesV[uint8](currentValue)
			*s.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](s.now())
			s.lastSent = &currentValue
			ctx.MarkChanged(ir.DefaultOutputParam)
		}
	}
}
