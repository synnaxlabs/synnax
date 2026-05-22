// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xstatus "github.com/synnaxlabs/x/status"
	"go.uber.org/zap"
)

const (
	bareSymbolName      = "set_status"
	qualifiedMemberName = "set"
	moduleName          = "status"
)

var (
	memberDoc = doc.New(
		doc.Paragraph("Sets a status notification on the cluster. Used to report alarms, warnings, or operational state."),
		doc.Divider(),
		doc.Code("arc", "sensor -> status.set{status_key=\"ox_alarm\", variant=\"error\", message=\"Overpressure\"}"),
		doc.Divider(),
		doc.Paragraph("Accepted variants: success, error, warning, info."),
	)
	moduleDoc = doc.New(
		doc.Paragraph("Publishes status notifications (alarms, warnings, operational state) to the cluster."),
	)
)

func newSymbolProps() types.Type {
	return types.Function(types.FunctionProperties{
		Config: types.Params{
			{Name: "status_key", Type: types.String()},
			{Name: "variant", Type: types.String()},
			{Name: "message", Type: types.String()},
			{Name: "name", Type: types.String(), Value: ""},
		},
		Inputs: types.Params{
			{Name: ir.DefaultOutputParam, Type: types.U8()},
		},
	})
}

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the status module plus the deprecated `set_status` bare
// global whose Deprecated field points at the canonical status.set member.
func NewSymbols() []*symbol.Symbol {
	member := &symbol.Symbol{
		Name: qualifiedMemberName,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: newSymbolProps(),
		Doc:  memberDoc,
	}
	mod := &symbol.Symbol{Name: moduleName, Kind: symbol.KindModule, Doc: moduleDoc}
	mod.AddChild(member)
	bare := &symbol.Symbol{
		Name:       bareSymbolName,
		Kind:       symbol.KindFunction,
		Exec:       symbol.ExecFlow,
		Type:       newSymbolProps(),
		Deprecated: mod.FindChild(qualifiedMemberName),
	}
	return []*symbol.Symbol{mod, bare}
}

type Module struct {
	stat *status.Service
}

func NewModule(stat *status.Service) *Module {
	return &Module{stat: stat}
}

func (m *Module) ModuleName() string { return moduleName }

func (m *Module) Create(ctx context.Context, cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != bareSymbolName && cfg.Node.Type != qualifiedMemberName {
		return nil, query.ErrNotFound
	}
	var nodeCfg setNodeConfig
	if err := setNodeConfigSchema.Parse(cfg.Node.Config.ValueMap(), &nodeCfg); err != nil {
		return nil, err
	}
	var stat status.Status[any]
	if err := m.stat.NewRetrieve().
		Where(status.MatchKeys[any](nodeCfg.StatusKey)).
		Entry(&stat).
		Exec(ctx, nil); errors.Skip(err, query.ErrNotFound) != nil {
		return nil, err
	}
	stat.Key = nodeCfg.StatusKey
	stat.Name = nodeCfg.Name
	stat.Message = nodeCfg.Message
	stat.Variant = xstatus.Variant(nodeCfg.Variant)
	return &setNode{ins: cfg.Instrumentation, stat: stat, statusSvc: m.stat}, nil
}

type setNodeConfig struct {
	StatusKey string `json:"status_key"`
	Message   string `json:"message"`
	Variant   string `json:"variant"`
	Name      string `json:"name"`
}

var setNodeConfigSchema = zyn.Object(map[string]zyn.Schema{
	"status_key": zyn.String(),
	"message":    zyn.String(),
	"variant":    zyn.String(),
	"name":       zyn.String().Optional(),
})

type setNode struct {
	statusSvc *status.Service
	ins       alamos.Instrumentation
	stat      status.Status[any]
}

func (s *setNode) Init(node.Context) {}

func (s *setNode) Reset() {}

func (s *setNode) IsOutputTruthy(int) bool { return false }

func (s *setNode) Next(ctx node.Context) {
	s.stat.Time = telem.Now()
	if err := s.statusSvc.NewWriter(nil).Set(ctx, &s.stat); err != nil {
		s.ins.L.Error("error setting status", zap.Error(err))
	}
}
