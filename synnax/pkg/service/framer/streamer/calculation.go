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
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

type calculationUpdaterTransform struct {
	alamos.Instrumentation
	c        *calculation.Service
	readable channel.Readable
	closer   xio.MultiCloser
	confluence.LinearTransform[Request, framer.StreamerRequest]
}

var _ confluence.Segment[Request, framer.StreamerRequest] = &calculationUpdaterTransform{}

func (t *calculationUpdaterTransform) update(ctx context.Context, keys []channel.Key) error {
	if err := t.closer.Close(); err != nil {
		return err
	}
	var channels []channel.Channel
	if err := t.readable.NewRetrieve().WhereKeys(keys...).Entries(&channels).Exec(ctx, nil); err != nil {
		return err
	}
	for _, ch := range channels {
		if ch.IsCalculated() {
			closer, err := t.c.Request(ctx, ch.Key())
			if err != nil {
				return err
			}
			t.closer = append(t.closer, closer)
		}
	}
	return nil
}

func (t *calculationUpdaterTransform) transform(ctx context.Context, req Request) (framer.StreamerRequest, bool, error) {
	if err := t.update(ctx, req.Keys); err != nil {
		t.L.Error("failed to update calculated channels", zap.Error(err))
	}
	return framer.StreamerRequest{Keys: req.Keys}, true, nil
}

func (t *calculationUpdaterTransform) Flow(ctx signal.Context, opts ...confluence.Option) {
	t.LinearTransform.Flow(ctx, append(opts, confluence.Defer(func() {
		if err := t.closer.Close(); err != nil {
			t.L.Error("failed to close calculated channels", zap.Error(err))
		}
	}))...)
}
