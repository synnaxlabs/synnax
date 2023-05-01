// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/relay"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type LiveWriter = confluence.Segment[framer.WriteRequest, framer.WriteResponse]

type liveWriter struct {
	confluence.AbstractUnarySink[framer.WriteRequest]
	confluence.AbstractUnarySource[framer.WriteResponse]
	relay confluence.Inlet[relay.Data]
}

func (l *liveWriter) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(l.Out)
	signal.GoRange(sCtx, l.In.Outlet(), func(ctx context.Context, request framer.WriteRequest) error {
		l.relay.Inlet() <- relay.Data{Frame: request.Frame}
		return nil
	})
}

func (s *Service) NewLiveWriter() LiveWriter {
	l := &liveWriter{}
	l.relay = s.relay.Writes()
	return l
}
