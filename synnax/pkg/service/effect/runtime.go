// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package effect

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/service/slate"
	"github.com/synnaxlabs/synnax/pkg/service/slate/reactive"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type entry struct {
	shutdown io.Closer
}

func (s *Service) handleChange(
	ctx context.Context,
	reader gorp.TxReader[uuid.UUID, Effect],
) {
	for {
		e, ok := reader.Next(ctx)
		if !ok {
			return
		}
		existing, found := s.mu.entries[e.Key]
		if found {
			if err := existing.shutdown.Close(); err != nil {
				s.cfg.L.Error("effect shut down with error", zap.Error(err))
			}
		}
		if e.Variant == changex.Delete {
			return
		}
		var slt slate.Slate
		if err := s.cfg.Slate.NewRetrieve().
			WhereKeys(e.Value.Slate).
			Entry(&slt).
			Exec(ctx, nil); err != nil {
			return
		}
		specCfg := spec.Config{
			Channel:    s.cfg.Channel,
			Framer:     s.cfg.Framer,
			Annotation: s.cfg.Annotation,
			Ranger:     s.cfg.Ranger,
			OnStatusChange: func(
				ctx context.Context,
				stat status.Status[any],
			) {
				os := Status{
					Key:         stat.Key,
					Variant:     stat.Variant,
					Message:     stat.Message,
					Description: stat.Description,
					Details:     StatusDetails{Effect: e.Key},
				}
				if _, err := s.effectStateWriter.Write(core.UnaryFrame(
					s.effectStateChannelKey,
					telem.NewSeriesStaticJSONV(os),
				)); err != nil {
					s.cfg.L.Error("effect status writer error", zap.Error(err))
				}
			},
		}
		if _, err := spec.Validate(ctx, specCfg, slt.Graph); err != nil {
			return
		}
		cfs, err := reactive.Create(ctx, specCfg, slt.Graph)
		if err != nil {
			return
		}
		sCtx, cancel := signal.Isolated(signal.WithInstrumentation(
			s.cfg.Instrumentation.Child(fmt.Sprintf("%s<%s>", e.Value.Name, e.Value.Key))))
		cfs.Flow(sCtx)
		s.mu.entries[e.Key] = &entry{shutdown: signal.NewHardShutdown(sCtx, cancel)}
	}
}
