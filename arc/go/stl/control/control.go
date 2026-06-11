// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	"context"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/zyn"
)

const (
	symbolName = "set_authority"
	name       = "control"
)

var (
	memberDoc = doc.New(
		doc.Paragraph("Dynamically changes the control authority of write channels at runtime."),
		doc.Divider(),
		doc.Code("arc", "control.set_authority{value=255}"),
		doc.Divider(),
		doc.Paragraph("Set authority for a specific channel:"),
		doc.Divider(),
		doc.Code("arc", "control.set_authority{value=255, channel=valve_cmd}"),
		doc.Divider(),
		doc.Paragraph("Authority is a u8 (0-255). Higher values take priority. Setting authority to 0 releases control of the channel."),
	)
	moduleDoc = doc.New(
		doc.Paragraph("Runtime control over write authority — promote or release writes targeting shared channels."),
	)
)

func newSymbolProps() types.Type {
	return types.Function(types.FunctionProperties{
		Inputs: types.Params{
			{Name: "value", Type: types.U8()},
			{Name: "channel", Type: types.WriteChan(types.Variable("T", nil)), Value: uint32(0)},
			{Name: ir.DefaultOutputParam, Type: types.U8(), Value: uint8(0)},
		},
	})
}

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the control module plus the deprecated bare alias
// (set_authority) whose Deprecated field points at the canonical member.
func NewSymbols() []*symbol.Symbol {
	member := &symbol.Symbol{
		Name:    symbolName,
		Kind:    symbol.KindFunction,
		Exec:    symbol.ExecFlow,
		Type:    newSymbolProps(),
		Trigger: symbol.TriggerInput(ir.DefaultOutputParam),
		Doc:     memberDoc,
	}
	mod := &symbol.Symbol{Name: name, Kind: symbol.KindModule, Doc: moduleDoc}
	mod.AddChild(member)
	bare := *member
	bare.Parent = nil
	bare.Deprecated = mod.FindChild(symbolName)
	return []*symbol.Symbol{mod, &bare}
}

// Host is the runtime host-side support for the control module: it acts as
// the node factory for set_authority. There are no WASM bindings for
// control; authority changes flow through ProgramState directly.
type Host struct {
	auth *ProgramState
}

// NewHost constructs a control Host backed by the given authority
// ProgramState. Control has no WASM host bindings; the Host is purely a
// node factory.
func NewHost(ab *ProgramState) *Host { return &Host{auth: ab} }

func (h *Host) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != symbolName {
		return nil, query.ErrNotFound
	}
	var nodeCfg nodeConfig
	if err := schema.Parse(cfg.Node.Inputs.ValueMap(), &nodeCfg); err != nil {
		return nil, errors.Wrap(err, "control.set_authority config")
	}
	var channelKey *uint32
	if nodeCfg.Channel != 0 {
		channelKey = &nodeCfg.Channel
	}
	return &setAuthority{
		auth:       h.auth,
		authority:  nodeCfg.Value,
		channelKey: channelKey,
	}, nil
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
	auth        *ProgramState
	authority   uint8
	channelKey  *uint32
	initialized bool
}

func (s *setAuthority) Reset()                  { s.initialized = false }
func (s *setAuthority) IsOutputTruthy(int) bool { return false }

func (s *setAuthority) Next(node.Context) {
	if s.initialized {
		return
	}
	s.initialized = true
	s.auth.Set(s.channelKey, s.authority)
}
