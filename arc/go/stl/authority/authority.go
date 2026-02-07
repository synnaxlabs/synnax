// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package authority

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/zyn"
)

var (
	symbolName = "set_authority"
	symbolDef  = symbol.Symbol{
		Name: symbolName,
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: types.Params{
				{Name: "value", Type: types.U8()},
				{Name: "channel", Type: types.WriteChan(types.U8()), Value: uint32(0)},
			},
			Inputs: types.Params{
				{Name: ir.DefaultOutputParam, Type: types.U8(), Value: uint8(0)},
			},
		}),
	}
	SymbolResolver = symbol.MapResolver{symbolName: symbolDef}
)

type Module struct {
	auth *state.AuthorityBuffer
}

var _ stl.Module = (*Module)(nil)

func NewModule(ab *state.AuthorityBuffer) *Module {
	return &Module{auth: ab}
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
	var nodeCfg nodeConfig
	if err := schema.Parse(cfg.Node.Config.ValueMap(), &nodeCfg); err != nil {
		return nil, errors.Wrap(err, "set_authority config")
	}
	var channelKey *uint32
	if nodeCfg.Channel != 0 {
		channelKey = &nodeCfg.Channel
	}
	return &setAuthority{
		auth:       m.auth,
		authority:  nodeCfg.Value,
		channelKey: channelKey,
	}, nil
}

func (m *Module) BindTo(_ context.Context, _ stl.HostRuntime) error {
	return nil
}

var schema = zyn.Object(map[string]zyn.Schema{
	"value":   zyn.Number().Uint8(),
	"channel": zyn.Number().Uint32(),
})

type nodeConfig struct {
	Value   uint8  `json:"value"`
	Channel uint32 `json:"channel"`
}

type setAuthority struct {
	auth        *state.AuthorityBuffer
	authority   uint8
	channelKey  *uint32
	initialized bool
}

func (s *setAuthority) Reset()                     { s.initialized = false }
func (s *setAuthority) IsOutputTruthy(string) bool { return false }

func (s *setAuthority) Next(node.Context) {
	if s.initialized {
		return
	}
	s.initialized = true
	s.auth.Set(s.channelKey, s.authority)
}
