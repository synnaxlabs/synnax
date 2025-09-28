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
}

func (s *Service) handleChange(
	ctx context.Context,
	reader gorp.TxReader[uuid.UUID, Arc],
) {
	for {
		e, ok := reader.Next(ctx)
		a := e.Value
		if !ok {
			return
		}
		existing, found := s.mu.entries[e.Key]
		if found {
			if err := existing.runtime.Close(); err != nil {
				s.cfg.L.Error("arc shut down with error", zap.Error(err))
			}
			if err := s.cfg.Status.NewWriter(nil).SetWithParent(
				ctx,
				&status.Status{
					Name:    a.Name,
					Key:     a.Key.String(),
					Variant: xstatus.DisabledVariant,
					Message: "Stopped",
					Time:    telem.Now(),
					Details: map[string]any{"running": false},
				},
				OntologyID(a.Key),
			); err != nil {
				s.cfg.L.Error("failed to set arc status", zap.Error(err))
			}
		}
		if e.Variant == changex.Delete || !a.Deploy {
			return
		}
		mod, err := arc.CompileGraph(ctx, e.Value.Graph, arc.WithResolver(s.symbolResolver))
		if err != nil {
			if err := s.cfg.Status.NewWriter(nil).SetWithParent(
				ctx,
				&status.Status{
					Name:        fmt.Sprintf("%s Status", a.Name),
					Key:         a.Key.String(),
					Variant:     xstatus.ErrorVariant,
					Message:     "Deployment Failed",
					Description: err.Error(),
					Time:        telem.Now(),
					Details:     map[string]any{"running": false},
				},
				OntologyID(a.Key),
			); err != nil {
				s.cfg.L.Error("failed to set arc status", zap.Error(err))
			}
			continue
		}
		baseCfg := s.cfg.baseRuntimeConfig()
		baseCfg.Module = mod
		r, err := runtime.Open(ctx, baseCfg)
		if err != nil {
			if err := s.cfg.Status.NewWriter(nil).SetWithParent(
				ctx,
				&status.Status{
					Name:        fmt.Sprintf("%s Status", a.Name),
					Key:         a.Key.String(),
					Message:     "Deployment Failed",
					Variant:     xstatus.ErrorVariant,
					Description: err.Error(),
					Time:        telem.Now(),
					Details:     map[string]any{"running": false},
				},
				OntologyID(a.Key),
			); err != nil {
				s.cfg.L.Error("failed to set arc status", zap.Error(err))
			}
			continue
		}
		s.mu.entries[e.Key] = &entry{runtime: r}
		if err := s.cfg.Status.NewWriter(nil).SetWithParent(
			ctx,
			&status.Status{
				Name:    fmt.Sprintf("%s Status", a.Name),
				Key:     a.Key.String(),
				Message: "Deployment Successful",
				Variant: xstatus.SuccessVariant,
				Time:    telem.Now(),
				Details: map[string]any{"running": true},
			},
			OntologyID(a.Key),
		); err != nil {
			s.cfg.L.Error("failed to set arc status", zap.Error(err))
		}
	}
}
