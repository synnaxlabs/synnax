// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"context"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type streamerSeg struct {
	confluence.UnarySink[framer.StreamerResponse]
	// requests are used to manage the life cycle of the telemetry frame streamerSeg.
	requests confluence.Inlet[framer.StreamerRequest]
}

func (s *streamerSeg) Close() error {
	s.requests.Close()
	confluence.Drain(s.In)
	return nil
}

type writerSeg struct {
	confluence.UnarySink[framer.WriterResponse]
	confluence.AbstractUnarySource[framer.WriterRequest]
}

func (w *writerSeg) sink(_ context.Context, res framer.WriterResponse) error {
	return nil
}

func (w *writerSeg) Write(ctx context.Context, fr framer.Frame) error {
	return signal.SendUnderContext(
		ctx,
		w.Out.Inlet(),
		framer.WriterRequest{Frame: fr, Command: writer.Write},
	)
}

func (w *writerSeg) Close() error {
	if w.Out != nil {
		w.Out.Close()
		confluence.Drain(w.In)
	}
	return nil
}

func retrieveChannels(
	ctx context.Context,
	channelSvc *channel.Service,
	keys []channel.Key,
) ([]channel.Channel, error) {
	channels := make([]channel.Channel, 0, len(keys))
	if err := channelSvc.NewRetrieve().
		WhereKeys(keys...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	indexes := lo.FilterMap(channels, func(item channel.Channel, index int) (channel.Key, bool) {
		return item.Index(), !item.Virtual
	})
	indexChannels := make([]channel.Channel, 0, len(indexes))
	if err := channelSvc.NewRetrieve().
		WhereKeys(indexes...).
		Entries(&indexChannels).Exec(ctx, nil); err != nil {
		return nil, err
	}
	return slices.Concat(channels, indexChannels), nil
}

var (
	streamerAddr address.Address = "streamerSeg"
	writerAddr   address.Address = "writer"
	runtimeAddr  address.Address = "runtime"
)

func createStreamPipeline(
	ctx context.Context,
	t *Task,
	frameSvc *framer.Service,
	readChannelKeys []channel.Key,
) (confluence.Flow, confluence.Inlet[framer.StreamerRequest], error) {
	p := plumber.New()
	streamer, err := frameSvc.NewStreamer(
		ctx,
		framer.StreamerConfig{Keys: readChannelKeys},
	)
	if err != nil {
		return nil, nil, err
	}
	plumber.SetSegment(p, streamerAddr, streamer)
	t.streamer.Sink = t.processFrame
	plumber.SetSink[framer.StreamerResponse](p, runtimeAddr, t.streamer)
	streamer.InFrom(confluence.NewStream[framer.StreamerRequest]())
	plumber.MustConnect[framer.StreamerResponse](p, streamerAddr, runtimeAddr, 10)
	requests := confluence.NewStream[framer.StreamerRequest]()
	streamer.InFrom(requests)
	return p, requests, nil
}

func createWritePipeline(
	ctx context.Context,
	name string,
	t *Task,
	frameSvc *framer.Service,
	writeChannelKeys []channel.Key,
) (confluence.Flow, error) {
	p := plumber.New()
	w, err := frameSvc.NewStreamWriter(
		ctx,
		framer.WriterConfig{
			ControlSubject: control.Subject{Name: name},
			Start:          telem.Now(),
			Keys:           writeChannelKeys,
		},
	)
	if err != nil {
		return nil, err
	}
	t.writer.Sink = t.writer.sink
	plumber.SetSegment(p, writerAddr, w)
	plumber.SetSegment[framer.WriterResponse, framer.WriterRequest](p, runtimeAddr, t.writer)
	plumber.MustConnect[framer.WriterResponse](p, writerAddr, runtimeAddr, 10)
	plumber.MustConnect[framer.WriterRequest](p, runtimeAddr, writerAddr, 10)
	return p, nil
}
