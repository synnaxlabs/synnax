// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
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

func (sf *server) handle(ctx context.Context, server ServerStream) error {
	sCtx, cancel := signal.WithCancel(ctx)
	defer cancel()

	// The first request provides the parameters for opening the toStorage writer
	req, err := server.Receive()
	if err != nil {
		return err
	}

	// Senders and receivers must be set up to distribution requests and responses
	// to their storage counterparts.
	receiver := &freightfluence.TransformReceiver[ts.WriterRequest, Request]{Receiver: server}
	receiver.Transform = newRequestTranslator()
	sender := &freightfluence.TransformSender[ts.WriterResponse, Response]{Sender: freighter.SenderNopCloser[Response]{StreamSender: server}}
	sender.Transform = newResponseTranslator(sf.HostResolver.HostKey())

	w, err := sf.TS.NewStreamWriter(ctx, req.Config.toStorage())
	if err != nil {
		return err
	}

	pipe := plumber.New()
	plumber.SetSegment(pipe, "toStorage", w)
	plumber.SetSource(pipe, "receiver", receiver)
	plumber.SetSink(pipe, "sender", sender)
	plumber.MustConnect[ts.WriterRequest](pipe, "receiver", "toStorage", 1)
	plumber.MustConnect[ts.WriterResponse](pipe, "toStorage", "sender", 1)
	pipe.Flow(sCtx, confluence.CloseOutputInletsOnExit(), confluence.RecoverWithErrOnPanic())

	err = sCtx.Wait()
	return err
}
