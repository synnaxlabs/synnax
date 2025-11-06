// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
)

type server struct{ ServiceConfig }

func newServer(cfg ServiceConfig) *server {
	s := &server{ServiceConfig: cfg}
	cfg.Transport.Server().BindHandler(s.handle)
	return s
}

// handle implements freighter.StreamServer.
func (sf *server) handle(ctx context.Context, server ServerStream) error {
	sCtx, cancel := signal.WithCancel(ctx)
	defer cancel()

	req, err := server.Receive()
	if err != nil {
		return err
	}

	receiver := &freightfluence.TransformReceiver[ts.IteratorRequest, Request]{Receiver: server}
	receiver.Transform = newStorageRequestTranslator(false)
	sender := &freightfluence.TransformSender[ts.IteratorResponse, Response]{
		Sender: freighter.SenderNopCloser[Response]{StreamSender: server},
	}
	sender.Transform = newStorageResponseTranslator(sf.HostResolver.HostKey())

	iter, err := sf.TS.NewStreamIterator(ts.IteratorConfig{
		Channels:      req.Keys.Storage(),
		Bounds:        req.Bounds,
		AutoChunkSize: req.ChunkSize,
	})
	if err != nil {
		return err
	}

	pipe := plumber.New()
	plumber.SetSegment(pipe, "storage", iter)
	plumber.SetSource(pipe, "receiver", receiver)
	plumber.SetSink(pipe, "sender", sender)
	plumber.MustConnect[ts.IteratorRequest](pipe, "receiver", "storage", 1)
	plumber.MustConnect[ts.IteratorResponse](pipe, "storage", "sender", 1)
	pipe.Flow(sCtx, confluence.CloseOutputInletsOnExit(), confluence.RecoverWithErrOnPanic())
	return sCtx.Wait()
}
