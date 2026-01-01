// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/core"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type entry struct {
	runtime *runtime.Runtime
	arc     Arc
}

func (s *Service) handleChange(
	ctx context.Context,
	reader gorp.TxReader[uuid.UUID, Arc],
) {
	for e := range reader {
		a := e.Value
		existing, found := s.mu.entries[e.Key]
		isDelete := e.Variant == changex.Delete
		if found {
			if err := existing.runtime.Close(); err != nil {
				s.cfg.L.Error("arc shut down with error", zap.Error(err))
			}
			if !isDelete {
				if err := status.NewWriter[core.StatusDetails](s.cfg.Status, nil).SetWithParent(
					ctx,
					&status.Status[core.StatusDetails]{
						Name:    existing.arc.Name,
						Key:     a.Key.String(),
						Variant: xstatus.DisabledVariant,
						Message: "Stopped",
						Time:    telem.Now(),
						Details: core.StatusDetails{Running: false},
					},
					OntologyID(a.Key),
				); err != nil {
					s.cfg.L.Error("failed to set arc status", zap.Error(err))
				}
			}
		}
		if isDelete || !a.Deploy {
			return
		}
		mod, err := arc.CompileGraph(ctx, e.Value.Graph, arc.WithResolver(s.symbolResolver))
		if err != nil {
			if err := status.NewWriter[core.StatusDetails](s.cfg.Status, nil).SetWithParent(
				ctx,
				&status.Status[core.StatusDetails]{
					Name:        fmt.Sprintf("%s Status", a.Name),
					Key:         a.Key.String(),
					Variant:     xstatus.VariantError,
					Message:     "Deployment Failed",
					Description: err.Error(),
					Time:        telem.Now(),
					Details:     core.StatusDetails{Running: false},
				},
				OntologyID(a.Key),
			); err != nil {
				s.cfg.L.Error("failed to set arc status", zap.Error(err))
			}
			continue
		}
		baseCfg := s.cfg.baseRuntimeConfig()
		baseCfg.Module = mod
		baseCfg.Name = a.Name
		r, err := runtime.Open(ctx, baseCfg)
		if err != nil {
			if err := status.NewWriter[core.StatusDetails](s.cfg.Status, nil).SetWithParent(
				ctx,
				&status.Status[core.StatusDetails]{
					Name:        fmt.Sprintf("%s Status", a.Name),
					Key:         a.Key.String(),
					Message:     "Deployment Failed",
					Variant:     xstatus.VariantError,
					Description: err.Error(),
					Time:        telem.Now(),
					Details:     core.StatusDetails{Running: false},
				},
				OntologyID(a.Key),
			); err != nil {
				s.cfg.L.Error("failed to set arc status", zap.Error(err))
			}
			continue
		}
		s.mu.entries[e.Key] = &entry{runtime: r, arc: a}
		if err := status.NewWriter[core.StatusDetails](s.cfg.Status, nil).SetWithParent(
			ctx,
			&status.Status[core.StatusDetails]{
				Name:    fmt.Sprintf("%s Status", a.Name),
				Key:     a.Key.String(),
				Message: "Deployment Successful",
				Variant: xstatus.VariantSuccess,
				Time:    telem.Now(),
				Details: core.StatusDetails{Running: true},
			},
			OntologyID(a.Key),
		); err != nil {
			s.cfg.L.Error("failed to set arc status", zap.Error(err))
		}
	}
}
