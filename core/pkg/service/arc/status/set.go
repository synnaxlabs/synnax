// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xstatus "github.com/synnaxlabs/x/status"
	"go.uber.org/zap"
)

var (
	symbolName = "set_status"
	symbolSet  = symbol.Symbol{
		Name: "set_status",
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Config: &types.Params{
				Keys: []string{"status_key", "variant", "message", "name"},
				Values: []types.Type{
					types.String(),
					types.String(),
					types.String(),
					types.String(),
				},
			},
			Inputs: &types.Params{
				Keys:   []string{ir.DefaultInputParam},
				Values: []types.Type{types.U8()},
			},
		}),
	}
	SymbolResolver = symbol.MapResolver{symbolName: symbolSet}
)

type setStatus struct {
	stat      status.Status
	statusSvc *status.Service
	ins       alamos.Instrumentation
}

func (s *setStatus) Init(node.Context) {}

func (s *setStatus) Next(ctx node.Context) {
	s.stat.Time = telem.Now()
	if err := s.statusSvc.NewWriter(nil).Set(ctx, &s.stat); err != nil {
		s.ins.L.Error("error setting status", zap.Error(err))
	}
}

type statusFactory struct {
	stat *status.Service
}

var schema = zyn.Object(map[string]zyn.Schema{
	"status_key": zyn.String(),
	"message":    zyn.String(),
	"variant":    zyn.String(),
})

type nodeConfig struct {
	StatusKey string
	Message   string
	Variant   string
}

func (s *statusFactory) Create(ctx context.Context, cfg node.Config) (node.Node, error) {
	var nodeCfg nodeConfig
	if err := schema.Parse(cfg.Node.ConfigValues, &nodeCfg); err != nil {
		return nil, err
	}
	var stat status.Status
	if err := s.stat.NewRetrieve().
		WhereKeys(nodeCfg.StatusKey).
		Entry(&stat).
		Exec(ctx, nil); errors.Skip(err, query.NotFound) != nil {
		return nil, err
	}
	stat.Key = nodeCfg.StatusKey
	stat.Message = nodeCfg.Message
	stat.Variant = xstatus.Variant(nodeCfg.Variant)
	return &setStatus{ins: cfg.Instrumentation, stat: stat, statusSvc: s.stat}, nil
}

func NewFactory(stat *status.Service) node.Factory {
	return &statusFactory{stat: stat}
}
