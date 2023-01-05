// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
)

type server struct{ ServiceConfig }

func startServer(cfg ServiceConfig) *server {
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

	receiver := &freightfluence.TransformReceiver[storage.TSIteratorRequest, Request]{Receiver: server}
	receiver.Transform = newStorageRequestTranslator()
	sender := &freightfluence.TransformSender[storage.TSIteratorResponse, Response]{
		Sender: freighter.SenderNopCloser[Response]{StreamSender: server},
	}
	sender.Transform = newStorageResponseTranslator(sf.HostResolver.HostID())

	iter, err := sf.TS.NewStreamIterator(storage.IteratorConfig{Channels: req.Keys.Strings(), Bounds: req.Bounds})
	if err != nil {
		return err
	}

	pipe := plumber.New()
	plumber.SetSegment[storage.TSIteratorRequest, storage.TSIteratorResponse](pipe, "storage", iter)
	plumber.SetSource[storage.TSIteratorRequest](pipe, "receiver", receiver)
	plumber.SetSink[storage.TSIteratorResponse](pipe, "sender", sender)
	plumber.MustConnect[Request](pipe, "receiver", "storage", 1)
	plumber.MustConnect[Response](pipe, "storage", "sender", 1)
	pipe.Flow(sCtx, confluence.CloseInletsOnExit())
	return sCtx.Wait()
}
