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

	"github.com/google/uuid"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
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
		if !ok {
			return
		}
		existing, found := s.mu.entries[e.Key]
		if found {
			if err := existing.runtime.Close(); err != nil {
				s.cfg.L.Error("effect shut down with error", zap.Error(err))
			}
		}
		if e.Variant == changex.Delete {
			return
		}
		mod, err := arc.CompileGraph(ctx, e.Value.Graph, arc.WithResolver(s.symbolResolver))
		if err != nil {
			s.cfg.L.Error("failed to compile graph", zap.Error(err))
			continue
		}
		baseCfg := s.cfg.baseRuntimeConfig()
		baseCfg.Module = mod
		r, err := runtime.Open(ctx, baseCfg)
		if err != nil {
			s.cfg.L.Error("failed to open runtime", zap.Error(err))
			continue
		}
		s.mu.entries[e.Key] = &entry{runtime: r}
	}
}
