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
	"github.com/samber/lo"
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

// ObservableConfig is the configuration for opening a CDC pipeline that subscribes
// to the provided observable and writes changes to the provided channels. Higher
// level CDC pipeline should be preferred, such as the SubscribeToGorp.
type ObservableConfig struct {
	// Name is an optional name for the CDC pipeline, used for debugging purposes.
	Name string
	// Set is the channel used to propagate set operations. Only Name and SetDataType
	// need to be provided. The config will automatically set Leaseholder to Free
	// and Virtual to true.
	Set channel.Channel
	// Delete is the channel used to propagate delete operations. Only Name and
	// SetDataType need to be provided. The config will automatically set Leaseholder
	// to Free and Virtual to true.
	Delete channel.Channel
	// Observable is the observable used to subscribe to changes. This observable should
	// return byte slice keys that are properly encoded for the channel's data type.
	Observable observe.Observable[[]change.Change[[]byte, struct{}]]
}

var (
	_ config.Config[ObservableConfig] = ObservableConfig{}
	// DefaultObservableConfig is the default configuration for the SubscribeToObservable
	// CDC pipeline.
	DefaultObservableConfig = ObservableConfig{}
)

const (
	nonVirtual = "CDC can only work with virtual free channels. Received false for %s"
	nonFree    = "CDC can only work with free channels. Received leaseholder %s that is not equal to Free"
)

// Validate implements config.Config.
func (c ObservableConfig) Validate() error {
	v := validate.New("cdc.Core")
	validate.NotEmptyString(v, "Set.Name", c.Set.Name)
	validate.NotEmptyString(v, "Delete.Name", c.Delete.Name)
	v.Ternaryf(!c.Set.Free(), nonFree, c.Set.Leaseholder)
	v.Ternaryf(!c.Delete.Free(), nonFree, c.Delete.Leaseholder)
	v.Ternaryf(!c.Set.Virtual, nonVirtual, c.Set.Name)
	v.Ternaryf(!c.Delete.Virtual, nonVirtual, c.Delete.Name)
	validate.NotNil(v, "Observable", c.Observable)
	return v.Error()
}

// Override implements config.Config.
func (c ObservableConfig) Override(other ObservableConfig) ObservableConfig {
	c.Name = override.If(c.Name, other.Name, c.Name == "")
	c.Set = override.If(c.Set, other.Set, c.Set.Name == "")
	c.Delete = override.If(c.Delete, other.Delete, c.Delete.Name == "")
	c.Observable = override.Nil(c.Observable, other.Observable)
	c.Set.Virtual = true
	c.Delete.Virtual = true
	c.Set.Leaseholder = core.Free
	c.Delete.Leaseholder = core.Free
	return c
}

// SubscribeToObservable opens a new CDC pipeline that subscribes to the configured
// Observable and writes changes to the configured channels. The returned io.Closer
// can be used to close the pipeline when done.
func (s *Provider) SubscribeToObservable(ctx context.Context, cfgs ...ObservableConfig) (io.Closer, error) {
	cfg, err := config.New(DefaultObservableConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	channels := []channel.Channel{cfg.Set, cfg.Delete}
	if err := s.Channel.CreateManyIfNamesDontExist(ctx, &channels); err != nil {
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
		Observable: cfg.Observable,
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
	plumber.SetSegment[framer.WriterRequest, framer.WriterResponse](p, "writer", w)
	responses := &confluence.UnarySink[framer.WriterResponse]{
		Sink: func(ctx context.Context, value framer.WriterResponse) error {
			s.Instrumentation.L.Error("unexpected writer response", zap.Bool("ack", value.Ack))
			return nil
		},
	}
	plumber.SetSink[framer.WriterResponse](p, "responses", responses)
	plumber.MustConnect[framer.WriterRequest](p, "source", "writer", 10)
	plumber.MustConnect[framer.WriterResponse](p, "writer", "responses", 10)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.Instrumentation.Child(lo.Ternary(cfg.Name != "", cfg.Name, cfg.Set.Name))))
	p.Flow(sCtx, confluence.CloseInletsOnExit())
	return signal.NewShutdown(sCtx, cancel), nil
}
