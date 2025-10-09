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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xstatus "github.com/synnaxlabs/x/status"
	"go.uber.org/zap"
)

var (
	symbolName = "set_status"
	symbolSet  = ir.Symbol{
		Name: "set_status",
		Kind: ir.KindStage,
		Type: ir.Stage{
			Config: ir.NamedTypes{
				Keys:   []string{"status_key", "variant", "message", "name"},
				Values: []ir.Type{ir.String{}, ir.String{}, ir.String{}, ir.String{}},
			},
			Params: ir.NamedTypes{
				Keys:   []string{ir.DefaultInputParam},
				Values: []ir.Type{ir.U8{}},
			},
		},
	}
	SymbolResolver = ir.MapResolver{symbolName: symbolSet}
)

type setStatus struct {
	stat      status.Status
	statusSvc *status.Service
	ins       alamos.Instrumentation
}

func (s *setStatus) Init(ctx context.Context, _ func(string)) {}

func (s *setStatus) Next(ctx context.Context, _ func(string)) {
	s.stat.Time = telem.Now()
	if err := s.statusSvc.NewWriter(nil).Set(ctx, &s.stat); err != nil {
		s.ins.L.Error("error setting status", zap.Error(err))
	}
}

type statusFactory struct {
	stat *status.Service
}

func (s *statusFactory) Create(ctx context.Context, cfg node.Config) (node.Node, error) {
	key := cfg.Node.Config["status_key"].(string)
	var stat status.Status
	if err := s.stat.NewRetrieve().
		WhereKeys(key).
		Entry(&stat).
		Exec(ctx, nil); errors.Skip(err, query.NotFound) != nil {
		return nil, err
	}
	stat.Key = key
	stat.Message = cfg.Node.Config["message"].(string)
	stat.Variant = xstatus.Variant(cfg.Node.Config["variant"].(string))
	return &setStatus{ins: cfg.Instrumentation, stat: stat}, nil
}

func NewFactory(stat *status.Service) (node.Factory, error) {
	return &statusFactory{stat: stat}, nil
}
