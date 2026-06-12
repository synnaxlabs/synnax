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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"
)

const (
	bareSymbolName      = "stable_for"
	qualifiedMemberName = "for"
	name                = "stable"
)

var (
	memberDoc = doc.New(
		doc.Paragraph("Emits a value only after it has remained stable for a specified duration. Prevents spurious signals from transient fluctuations."),
		doc.Divider(),
		doc.Code("arc", "sensor -> stable.for{duration=5s} -> output"),
	)
	moduleDoc = doc.New(
		doc.Paragraph("Debouncing helpers that emit only after values remain stable for a duration."),
	)
)

func newSymbolType() types.Type {
	return types.Function(types.FunctionProperties{
		Inputs: types.Params{
			{Name: ir.DefaultInputParam, Type: types.Variable("T", nil)},
			{Name: "duration", Type: types.TimeSpan()},
		},
		Outputs: types.Params{
			{Name: ir.DefaultOutputParam, Type: types.Variable("T", nil)},
		},
	})
}

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the stable module plus the deprecated `stable_for` bare
// alias whose Deprecated field points at the canonical stable.for member.
func NewSymbols() []*symbol.Symbol {
	member := &symbol.Symbol{
		Name:    qualifiedMemberName,
		Kind:    symbol.KindFunction,
		Exec:    symbol.ExecFlow,
		Type:    newSymbolType(),
		Trigger: symbol.TriggerInput(ir.DefaultInputParam),
		Doc:     memberDoc,
	}
	mod := &symbol.Symbol{Name: name, Kind: symbol.KindModule, Doc: moduleDoc}
	mod.AddChild(member)
	bare := *member
	bare.Parent = nil
	bare.Name = bareSymbolName
	bare.Deprecated = mod.FindChild(qualifiedMemberName)
	return []*symbol.Symbol{mod, &bare}
}

// Host is the runtime host-side support for the stable module: a node
// factory for stable.for. Stable has no WASM host bindings.
type Host struct {
	now func() telem.TimeStamp
}

// NewHost constructs a stable Host. By default uses telem.Now; pass
// WithNow(fn) to override (used by tests for deterministic timestamps).
func NewHost(opts ...func(*Host)) *Host {
	h := &Host{now: telem.Now}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// WithNow overrides the clock used by stable nodes. Tests pass a
// deterministic clock here.
func WithNow(fn func() telem.TimeStamp) func(*Host) {
	return func(h *Host) { h.now = fn }
}

func (h *Host) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != bareSymbolName && cfg.Node.Type != qualifiedMemberName {
		return nil, query.ErrNotFound
	}
	var cfgVals config
	if err := configSchema.Parse(cfg.Node.Inputs.ValueMap(), &cfgVals); err != nil {
		return nil, err
	}
	return &forNode{State: cfg.State, duration: cfgVals.Duration, now: h.now}, nil
}

type config struct {
	Duration telem.TimeSpan
}

var configSchema = zyn.Object(map[string]zyn.Schema{
	"duration": zyn.Int64().Coerce(),
})

type forNode struct {
	*node.State
	value       *uint8
	lastSent    *uint8
	now         func() telem.TimeStamp
	duration    telem.TimeSpan
	lastChanged telem.TimeStamp
}

func (s *forNode) Reset() {
	s.State.Reset()
	s.value = nil
	s.lastSent = nil
	s.lastChanged = 0
}

var _ node.Node = (*forNode)(nil)

func (s *forNode) Next(ctx node.Context) {
	if s.RefreshInputs() {
		inputData := s.InputNamed(ir.DefaultInputParam)
		inputTime := s.InputTimeNamed(ir.DefaultInputParam)
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
			ctx.MarkChanged(0)
		}
	}
}
