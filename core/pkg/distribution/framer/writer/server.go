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

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

type server struct{ ServiceConfig }

func startServer(cfg ServiceConfig) *server {
	s := &server{ServiceConfig: cfg}
	cfg.Transport.Server().BindHandler(s.handle)
	return s
}

// handle services a single peer writer stream. When AutoIndexing is enabled in the
// open-time config, an autoIndexStamper is inserted between the receiver and the
// storage writer so timestamps for indexes local to this leaseholder are produced from
// this node's clock.
func (sf *server) handle(ctx context.Context, server ServerStream) error {
	sCtx, cancel := signal.WithCancel(ctx)
	defer cancel()

	// The first request provides the parameters for opening the toStorage writer
	req, err := server.Receive()
	if err != nil {
		return err
	}

	w, err := sf.TS.NewStreamWriter(ctx, req.Config.toStorage())
	if err != nil {
		return err
	}

	pipe := plumber.New()
	plumber.SetSegment(pipe, "toStorage", w)

	sender := &freightfluence.TransformSender[ts.WriterResponse, Response]{
		Sender: freighter.SenderNopCloser[Response]{StreamSender: server},
	}
	sender.Transform = newResponseTranslator(sf.HostResolver.HostKey())
	plumber.SetSink(pipe, "sender", sender)
	plumber.MustConnect[ts.WriterResponse](pipe, "toStorage", "sender", 1)

	rcv := &freightfluence.Receiver[Request]{Receiver: server}
	plumber.SetSource(pipe, "receiver", rcv)
	reqXform := &confluence.LinearTransform[Request, ts.WriterRequest]{}
	reqXform.Transform = newRequestTranslator()
	plumber.SetSegment(pipe, "request_translator", reqXform)
	plumber.MustConnect[ts.WriterRequest](pipe, "request_translator", "toStorage", 1)

	if req.Config.AutoIndexing != nil && *req.Config.AutoIndexing {
		channels, err := sf.resolveChannels(ctx, req.Config.Keys)
		if err != nil {
			return err
		}
		ai := newAutoIndexer(channels, req.Config)
		plumber.SetSegment(pipe, "auto_indexer", ai)
		plumber.MustConnect[Request](pipe, "receiver", "auto_indexer", 1)
		plumber.MustConnect[Request](pipe, "auto_indexer", "request_translator", 1)
	} else {
		plumber.MustConnect[Request](pipe, "receiver", "request_translator", 1)
	}

	pipe.Flow(sCtx, confluence.CloseOutputInletsOnExit(), confluence.RecoverWithErrOnPanic())

	err = sCtx.Wait()
	return err
}

func (sf *server) resolveChannels(
	ctx context.Context,
	keys channel.Keys,
) ([]channel.Channel, error) {
	var channels []channel.Channel
	if err := sf.Channel.
		NewRetrieve().
		Entries(&channels).
		Where(channel.MatchKeys(keys...)).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	if len(channels) != len(keys) {
		missing, _ := lo.Difference(keys, channel.KeysFromChannels(channels))
		return nil, errors.Wrapf(validate.ErrValidation, "missing channels: %v", missing)
	}
	return channels, nil
}
