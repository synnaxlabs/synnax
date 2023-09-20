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
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"io"
)

// CoreConfig is the configuration for
type CoreConfig struct {
	// Set is the channel used to propagate set operations. Only Name and DataType
	// need to be provided. The config will automatically set Leaseholder to Free
	// and Virtual to true.
	Set channel.Channel
	// Delete is the channel used to propagate delete operations. Only Name and
	// DataType need to be provided. The config will automatically set Leaseholder
	// to Free and Virtual to true.
	Delete channel.Channel
	// Obs is the observable used to subscribe to changes. This observable should
	// return byte slice keys that are properly encoded for the channel's data type.
	Obs observe.Observable[[]change.Change[[]byte, struct{}]]
}

var (
	_                 config.Config[CoreConfig] = CoreConfig{}
	DefaultCoreConfig                           = CoreConfig{}
)

const (
	nonVirtual = "CDC can only work with virtual free channels. Received false for Delete.Virtual"
	nonFree    = "CDC can only work with free channels. Received leasholder %s that is not equal to Free"
)

// Validate implements config.Config.
func (c CoreConfig) Validate() error {
	v := validate.New("cdc.Core")
	validate.NotEmptyString(v, "Set.Name", c.Set.Name)
	validate.NotEmptyString(v, "Delete.Name", c.Delete.Name)
	v.Ternaryf(!c.Set.Free(), nonFree, c.Set.Leaseholder)
	v.Ternaryf(!c.Delete.Free(), nonFree, c.Delete.Leaseholder)
	v.Ternary(c.Set.Virtual, nonVirtual)
	v.Ternary(!c.Delete.Virtual, nonVirtual)
	validate.NotNil(v, "Obs", c.Obs)
	return v.Error()
}

// Override implements config.Config.
func (c CoreConfig) Override(other CoreConfig) CoreConfig {
	c.Set = override.If(c.Set, other.Set, c.Set.Name == "")
	c.Delete = override.If(c.Delete, other.Delete, c.Delete.Name == "")
	c.Obs = override.Nil(c.Obs, other.Obs)
	c.Set.Virtual = true
	c.Delete.Virtual = true
	c.Set.Leaseholder = core.Free
	c.Delete.Leaseholder = core.Free
	return c
}

func (s *Service) OpenCore(ctx context.Context, cfgs ...CoreConfig) (io.Closer, error) {
	cfg, err := config.New(DefaultCoreConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	channels := []channel.Channel{cfg.Set, cfg.Delete}
	if err := s.Channel.RetrieveByNameOrCreate(ctx, &channels); err != nil {
		return nil, err
	}
	keys := channel.KeysFromChannels(channels)
	w, err := s.Framer.NewStreamWriter(ctx, framer.WriterConfig{
		Keys:  keys,
		Start: telem.Now(),
	})
	if err != nil {
		return nil, err
	}
	for _, ch := range channels {
		if ch.Name == cfg.Set.Name {
			cfg.Set = ch
		} else {
			cfg.Delete = ch
		}
	}
	t := &confluence.TransformSubscriber[[]change.Change[[]byte, struct{}], framer.WriterRequest]{
		Observable: cfg.Obs,
		Transform: func(ctx context.Context, r []change.Change[[]byte, struct{}]) (framer.WriterRequest, bool, error) {
			if len(r) == 0 {
				return framer.WriterRequest{}, false, nil
			}
			var (
				frame   framer.Frame
				sets    = telem.Series{DataType: cfg.Set.DataType}
				deletes = telem.Series{DataType: cfg.Delete.DataType}
			)
			for _, c := range r {
				if c.Variant == change.Delete {
					deletes.Data = append(deletes.Data, c.Key...)
				} else {
					sets.Data = append(sets.Data, c.Key...)
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
	plumber.SetSegment[framer.WriterRequest, framer.WriterResponse](p, "sink", w)
	responses := &confluence.UnarySink[framer.WriterResponse]{
		Sink: func(ctx context.Context, value framer.WriterResponse) error {
			s.Instrumentation.L.Error("Unexpected writer response", zap.Bool("ack", value.Ack))
			return nil
		},
	}
	plumber.SetSink[framer.WriterResponse](p, "responses", responses)
	plumber.MustConnect[framer.WriterRequest](p, "source", "sink", 10)
	plumber.MustConnect[framer.WriterResponse](p, "sink", "responses", 10)

	sCtx, cancel := signal.Isolated()
	p.Flow(sCtx)
	return signal.NewShutdown(sCtx, cancel), nil
}
