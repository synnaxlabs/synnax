// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cdc

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"io"
)

type CoreConfig struct {
	Set    channel.Channel
	Delete channel.Channel
	Obs    observe.Observable[[]change.Change[[]byte, struct{}]]
}

func (s *Service) OpenCore(ctx context.Context, cfg CoreConfig) (io.Closer, error) {
	channels := []channel.Channel{cfg.Set, cfg.Delete}
	if err := s.channel.RetrieveByNameOrCreate(ctx, &channels); err != nil {
		return nil, err
	}
	keys := channel.KeysFromChannels(channels)
	w, err := s.framer.NewStreamWriter(ctx, framer.WriterConfig{
		Keys:  keys,
		Start: telem.Now(),
	})
	if err != nil {
		return nil, err
	}
	t := &confluence.TransformSubscriber[[]change.Change[[]byte, struct{}], framer.WriterRequest]{
		Observable: cfg.Obs,
		Transform: func(ctx context.Context, r []change.Change[[]byte, struct{}]) (framer.WriterRequest, bool, error) {
			var (
				frame   framer.Frame
				sets    telem.Series
				deletes telem.Series
			)
			for _, c := range r {
				if c.Variant == change.Delete {
					sets.Data = append(sets.Data, c.Key...)
				} else {
					deletes.Data = append(deletes.Data, c.Key...)
				}
			}
			if len(sets.Data) > 0 {
				frame.Keys = []channel.Key{cfg.Set.Key()}
				frame.Series = []telem.Series{sets}
			}
			if len(deletes.Data) > 0 {
				frame.Keys = append(frame.Keys, cfg.Delete.Key())
				frame.Series = append(frame.Series, deletes)
			}
			return framer.WriterRequest{Command: writer.Data, Frame: frame}, true, nil
		},
	}
	p := plumber.New()
	plumber.SetSource[framer.WriterRequest](p, "source", t)
	plumber.SetSink[framer.WriterRequest](p, "sink", w)
	sCtx, cancel := signal.Isolated()
	p.Flow(sCtx)
	return signal.NewShutdown(sCtx, cancel), nil
}
