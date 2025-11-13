// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package streamer

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

type calculationUpdaterTransform struct {
	alamos.Instrumentation
	readable    channel.Readable
	calcManager *calculation.RequestManager
	confluence.LinearTransform[Request, framer.StreamerRequest]
}

var _ confluence.Segment[Request, framer.StreamerRequest] = &calculationUpdaterTransform{}

func (t *calculationUpdaterTransform) transform(ctx context.Context, req Request) (framer.StreamerRequest, bool, error) {
	if err := t.calcManager.Set(ctx, req.Keys); err != nil {
		t.L.Error("failed to update calculated channels", zap.Error(err))
	}
	return framer.StreamerRequest{Keys: req.Keys}, true, nil
}

func (t *calculationUpdaterTransform) Flow(ctx signal.Context, opts ...confluence.Option) {
	t.LinearTransform.Flow(ctx, append(opts, confluence.Defer(func() {
		// Explicitly use a context.TODO() here as the parent context may have
		// been cancelled.
		if err := t.calcManager.Close(context.TODO()); err != nil {
			t.L.Error("failed to close calculation", zap.Error(err))
		}
	}))...)
}
