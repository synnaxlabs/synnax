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
			Keys:   []string{"key", "variant", "message", "name"},
			Values: []ir.Type{ir.String{}, ir.String{}, ir.String{}, ir.String{}},
		},
	},
}

type setStatus struct {
	base
	cfg     Config
	key     string
	variant xstatus.Variant
	message string
}

func createSetStatus(_ context.Context, cfg Config) (stage.Stage, error) {
	s := &setStatus{
		cfg:     cfg,
		key:     cfg.Node.Config["key"].(string),
		message: cfg.Node.Config["message"].(string),
		variant: xstatus.Variant(cfg.Node.Config["variant"].(string)),
	}
	s.base.key = cfg.Node.Key
	return s, nil
}

func (s *setStatus) Next(ctx context.Context, _ value.Value) {
	stat := status.Status{}
	stat.Key = s.key
	stat.Name = "OX Alarm"
	stat.Variant = s.variant
	stat.Message = s.message
	if err := s.cfg.Status.NewWriter(nil).Set(ctx, &stat); err != nil {
		s.cfg.L.Error("error setting status", zap.Error(err))
	}
}
