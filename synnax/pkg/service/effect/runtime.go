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
	"github.com/synnaxlabs/synnax/pkg/service/slate"
	"github.com/synnaxlabs/synnax/pkg/service/slate/event"
	"github.com/synnaxlabs/synnax/pkg/service/slate/spec"
	changex "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/signal"
)

type entry struct {
	shutdown io.Closer
}

type Runtime struct {
	cfg Config
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
			existing.shutdown.Close()
		}
		if e.Variant == changex.Delete {
			return
		}
		var slt slate.Slate
		if err := s.cfg.Slate.NewRetrieve().WhereKeys(e.Value.Slate).Entry(&slt).Exec(ctx, nil); err != nil {
			return
		}
		specCfg := spec.Config{
			Channel: s.cfg.Channel,
			Framer:  s.cfg.Framer,
		}
		if _, err := spec.Validate(ctx, specCfg, slt.Graph); err != nil {
			fmt.Println(err)
			return
		}
		cfs, err := event.Create(ctx, specCfg, slt.Graph)
		if err != nil {
			fmt.Println(err)
			return
		}
		sCtx, cancel := signal.Isolated()
		cfs.Flow(sCtx)
		s.mu.entries[e.Key] = &entry{shutdown: signal.NewHardShutdown(sCtx, cancel)}
	}
}
