// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package reactive

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/telem"
)

type rangeCreate struct {
	confluence.MultiSink[spec.Value]
	cfg        spec.RangeCreateConfig
	currRng    ranger.Range
	rngService *ranger.Service
}

func newRangeCreate(_ context.Context, cfg factoryConfig) (bool, error) {
	if cfg.node.Type != spec.RangeCreateType {
		return false, nil
	}
	rCfg := spec.RangeCreateConfig{}
	if err := rCfg.Parse(cfg.node.Config); err != nil {
		return true, err
	}
	rCfg.Range.Key = uuid.Nil
	rCfg.Range.Stage = ""
	r := &rangeCreate{currRng: rCfg.Range, rngService: cfg.Ranger, cfg: rCfg}
	r.Sink = r.sink
	plumber.SetSink[spec.Value](cfg.pipeline, address.Address(cfg.node.Key), r)
	return true, nil
}

func (r *rangeCreate) sink(ctx context.Context, origin address.Address, _ spec.Value) error {
	originStage := ranger.Stage(origin)
	shouldCreate := r.currRng.Stage != originStage
	if r.currRng.Key != uuid.Nil {
		if r.currRng.Stage.EarlierThan(ranger.Stage(origin)) {
			r.currRng = r.cfg.Range
			r.currRng.Key = uuid.Nil
		} else if r.currRng.Stage != originStage {
			r.currRng.Stage = originStage
		}
	}
	if r.currRng.Key == uuid.Nil {
		r.currRng.Key = uuid.New()
		r.currRng.Stage = originStage
		r.currRng.TimeRange.Start = telem.Now()
	}
	if shouldCreate {
		r.currRng.TimeRange.End = telem.Now()
		return r.rngService.NewWriter(nil).Create(ctx, &r.currRng)
	}
	return nil
}
