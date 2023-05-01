// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
)

type server struct {
	Config
	newReader func(keys ...channel.Key) confluence.Segment[ReadRequest, Data]
}

func startServer(
	cfg Config,
	newReader func(keys ...channel.Key) confluence.Segment[ReadRequest, Data],
) *server {
	s := &server{Config: cfg, newReader: newReader}
	cfg.Transport.Server().BindHandler(s.handle)
	return s
}

func (s *server) handle(ctx context.Context, server ServerStream) error {
	var (
		sCtx, cancel = signal.WithCancel(ctx)
		receiver     = &freightfluence.Receiver[ReadRequest]{Receiver: server}
		sender       = &freightfluence.Sender[Data]{
			Sender: freighter.SenderNopCloser[Data]{StreamSender: server},
		}
		reader = s.newReader()
		pipe   = plumber.New()
	)
	defer cancel()
	plumber.SetSegment(pipe, "reader", reader)
	plumber.SetSource[ReadRequest](pipe, "receiver", receiver)
	plumber.SetSink[framer.Frame](pipe, "sender", sender)
	plumber.MustConnect[ReadRequest](pipe, "receiver", "reader", 1)
	plumber.MustConnect[framer.Frame](pipe, "reader", "sender", 1)
	pipe.Flow(sCtx, confluence.CloseInletsOnExit())
	return sCtx.Wait()
}
