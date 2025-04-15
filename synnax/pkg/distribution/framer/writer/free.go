// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

func (s *Service) newFree(mode Mode, sync bool) StreamWriter {
	w := &freeWriter{freeWrites: s.FreeWrites, mode: mode, sync: sync}
	w.Transform = w.transform
	return w
}

type freeWriter struct {
	confluence.LinearTransform[Request, Response]
	freeWrites confluence.Inlet[relay.Response]
	mode       Mode
	sync       bool
}

func (w *freeWriter) transform(ctx context.Context, req Request) (res Response, ok bool, err error) {
	if req.Command == Data && w.mode.Stream() {
		if err = signal.SendUnderContext(
			ctx, w.freeWrites.Inlet(),
			relay.Response{Frame: req.Frame},
		); err != nil || !w.sync {
			return
		}
	}
	return Response{Command: req.Command, SeqNum: req.SeqNum}, true, nil
}
