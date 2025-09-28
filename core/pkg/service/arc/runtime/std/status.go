// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package std

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/value"
	"github.com/synnaxlabs/x/telem"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime/stage"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/maps"
	xstatus "github.com/synnaxlabs/x/status"
	"go.uber.org/zap"
)

var symbolSetStatus = ir.Symbol{
	Name: "set_status",
	Kind: ir.KindStage,
	Type: ir.Stage{
		Config: maps.Ordered[string, ir.Type]{
			Keys:   []string{"status_key", "variant", "message", "name"},
			Values: []ir.Type{ir.String{}, ir.String{}, ir.String{}, ir.String{}},
		},
		Params: maps.Ordered[string, ir.Type]{
			Keys:   []string{"input"},
			Values: []ir.Type{ir.U8{}},
		},
	},
}

type setStatus struct {
	base
	cfg  Config
	stat status.Status
}

func createSetStatus(_ context.Context, cfg Config) (stage.Stage, error) {
	s := status.Status{
		Name:    cfg.Node.Config["name"].(string),
		Key:     cfg.Node.Config["status_key"].(string),
		Message: cfg.Node.Config["message"].(string),
		Variant: xstatus.Variant(cfg.Node.Config["variant"].(string)),
	}
	stg := &setStatus{cfg: cfg, stat: s}
	stg.base.key = cfg.Node.Key
	return stg, nil
}

func (s *setStatus) Next(ctx context.Context, _ string, _ value.Value) {
	s.stat.Time = telem.Now()
	if err := s.cfg.Status.NewWriter(nil).Set(ctx, &s.stat); err != nil {
		s.cfg.L.Error("error setting status", zap.Error(err))
	}
}
