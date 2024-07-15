// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

type (
	FrameIteratorRequest  = framer.IteratorRequest
	FrameIteratorResponse = framer.IteratorResponse
	FrameIteratorStream   = freighter.ServerStream[FrameIteratorRequest, FrameIteratorResponse]
)

func (s *FrameService) Iterate(ctx context.Context, stream FrameIteratorStream) error {
	iter, err := s.openIterator(ctx, stream)
	if err != nil {
		return err
	}

	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Instrumentation.Child("frame_iterator")))
	// Cancellation here would occur for one of two reasons. Either we encounter
	// a fatal error (transport or iterator internal) and we need to free all
	// resources, OR the client executed the close command on the iterator (in
	// which case resources have already been freed and cancel does nothing).
	defer cancel()

	receiver := &freightfluence.Receiver[iterator.Request]{Receiver: stream}
	sender := &freightfluence.TransformSender[iterator.Response, iterator.Response]{
		Sender: freighter.SenderNopCloser[iterator.Response]{StreamSender: stream},
		Transform: func(ctx context.Context, res iterator.Response) (iterator.Response, bool, error) {
			res.Error = errors.Encode(ctx, res.Error, false)
			return res, true, nil
		},
	}
	pipe := plumber.New()
	plumber.SetSegment(pipe, "iterator", iter)
	plumber.SetSink[iterator.Response](pipe, "sender", sender)
	plumber.SetSource[iterator.Request](pipe, "receiver", receiver)
	plumber.MustConnect[iterator.Response](pipe, "iterator", "sender", 1)
	plumber.MustConnect[iterator.Request](pipe, "receiver", "iterator", 1)

	pipe.Flow(sCtx, confluence.CloseInletsOnExit())
	return sCtx.Wait()
}

func (s *FrameService) openIterator(ctx context.Context, srv FrameIteratorStream) (framer.StreamIterator, error) {
	req, err := srv.Receive()
	if err != nil {
		return nil, err
	}
	iter, err := s.Internal.NewStreamIterator(ctx, framer.IteratorConfig{
		Bounds:    req.Bounds,
		Keys:      req.Keys,
		ChunkSize: req.ChunkSize,
	})
	if err != nil {
		return nil, err
	}
	return iter, srv.Send(framer.IteratorResponse{
		Variant: iterator.AckResponse,
		Ack:     true,
	})
}
