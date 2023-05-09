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
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
)

type server struct {
	Config
	newReader func(ctx context.Context, cfg ReaderConfig) (confluence.Segment[Request, Response], error)
}

func startServer(
	cfg Config,
	newReader func(ctx context.Context, cfg ReaderConfig) (confluence.Segment[Request, Response], error),
) *server {
	s := &server{Config: cfg, newReader: newReader}
	cfg.Transport.Server().BindHandler(s.handle)
	return s
}

func (s *server) handle(ctx context.Context, server ServerStream) error {
	var (
		sCtx, cancel = signal.WithCancel(ctx)
		rcv          = &freightfluence.Receiver[Request]{Receiver: server}
		sender       = &freightfluence.Sender[Response]{
			Sender: freighter.SenderNopCloser[Response]{StreamSender: server},
		}
		reader, err = s.newReader(ctx, ReaderConfig{})
		pipe        = plumber.New()
	)
	defer cancel()
	if err != nil {
		return err
	}
	plumber.SetSegment(pipe, "reader", reader)
	plumber.SetSource[Request](pipe, "rcv", rcv)
	plumber.SetSink[Response](pipe, "sender", sender)
	plumber.MustConnect[Request](pipe, "rcv", "reader", 1)
	plumber.MustConnect[Response](pipe, "reader", "sender", 1)
	pipe.Flow(sCtx, confluence.CloseInletsOnExit())
	return sCtx.Wait()
}
