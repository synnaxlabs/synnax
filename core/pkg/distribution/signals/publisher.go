// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals

import (
	"context"
	"io"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ObservablePublisherConfig is the configuration for opening a Signals pipeline that subscribes
// to the provided observable and writes changes to the provided channels. Higher
// level Signals pipeline should be preferred, such as the PublishFromGorp.
type ObservablePublisherConfig struct {
	// Name is an optional name for the Signals pipeline, used for debugging purposes.
	Name string
	// SetChannel is the channel used to propagate set operations. Only Name and SetDataType
	// need to be provided. The config will automatically set Leaseholder to Free
	// and Virtual to true.
	SetChannel channel.Channel
	// DeleteChannel is the channel used to propagate delete operations. Only Name and
	// SetDataType need to be provided. The config will automatically set Leaseholder
	// to Free and Virtual to true.
	DeleteChannel channel.Channel
	// Observable is the observable used to subscribe to changes. This observable should
	// return byte slice keys that are properly encoded for the channel's data type.
	Observable observe.Observable[[]change.Change[[]byte, struct{}]]
}

var (
	_ config.Config[ObservablePublisherConfig] = ObservablePublisherConfig{}
	// DefaultObservableConfig is the default configuration for the PublishFromObservable
	// Signals pipeline.
	DefaultObservableConfig = ObservablePublisherConfig{}
)

const (
	nonVirtual = "Signals can only work with virtual free channels. Received false for %s"
	nonFree    = "Signals can only work with free channels. Received leaseholder %s that is not equal to Free"
)

// Validate implements config.Config.
func (c ObservablePublisherConfig) Validate() error {
	v := validate.New("signals.observable_publisher_config")
	validate.NotEmptyString(v, "set_channel.name", c.SetChannel.Name)
	validate.NotEmptyString(v, "delete_channel.name", c.DeleteChannel.Name)
	v.Ternaryf("set_channel.leaseholder", !c.SetChannel.Free(), nonFree, c.SetChannel.Leaseholder)
	v.Ternaryf("delete_channel.leaseholder", !c.DeleteChannel.Free(), nonFree, c.DeleteChannel.Leaseholder)
	v.Ternaryf("set_channel.virtual", !c.SetChannel.Virtual, nonVirtual, c.SetChannel.Name)
	v.Ternaryf("delete_channel.virtual", !c.DeleteChannel.Virtual, nonVirtual, c.DeleteChannel.Name)
	validate.NotNil(v, "observable", c.Observable)
	return v.Error()
}

// Override implements config.Config.
func (c ObservablePublisherConfig) Override(other ObservablePublisherConfig) ObservablePublisherConfig {
	c.Name = override.If(c.Name, other.Name, c.Name == "")
	c.SetChannel = override.If(c.SetChannel, other.SetChannel, c.SetChannel.Name == "")
	c.DeleteChannel = override.If(c.DeleteChannel, other.DeleteChannel, c.DeleteChannel.Name == "")
	c.Observable = override.Nil(c.Observable, other.Observable)
	c.SetChannel.Virtual = true
	c.DeleteChannel.Virtual = true
	c.SetChannel.Leaseholder = cluster.Free
	c.DeleteChannel.Leaseholder = cluster.Free
	return c
}

// PublishFromObservable opens a new Signals pipeline that subscribes to the configured
// ObservableSubscriber and writes changes to the configured channels. The returned io.Closer
// can be used to close the pipeline when done.
func (s *Provider) PublishFromObservable(ctx context.Context, cfgs ...ObservablePublisherConfig) (io.Closer, error) {
	cfg, err := config.New(DefaultObservableConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	channels := []channel.Channel{cfg.SetChannel, cfg.DeleteChannel}
	if err = s.Channel.CreateMany(ctx, &channels, channel.RetrieveIfNameExists(), channel.OverwriteIfNameExistsAndDifferentProperties()); err != nil {
		return nil, err
	}
	keys := channel.KeysFromChannels(channels)
	w, err := s.Framer.NewStreamWriter(ctx, framer.WriterConfig{
		Keys:        keys,
		Start:       telem.Now(),
		Authorities: []control.Authority{255},
	})
	if err != nil {
		return nil, err
	}
	for _, ch := range channels {
		if ch.Name == cfg.SetChannel.Name {
			cfg.SetChannel = ch
		} else {
			cfg.DeleteChannel = ch
		}
	}
	t := &confluence.ObservableTransformPublisher[[]change.Change[[]byte, struct{}], framer.WriterRequest]{
		Observable: cfg.Observable,
		Transform: func(ctx context.Context, r []change.Change[[]byte, struct{}]) (framer.WriterRequest, bool, error) {
			if len(r) == 0 {
				return framer.WriterRequest{}, false, nil
			}
			var (
				frame   framer.Frame
				sets    = telem.Series{DataType: cfg.SetChannel.DataType}
				deletes = telem.Series{DataType: cfg.DeleteChannel.DataType}
			)
			for _, c := range r {
				if c.Variant == change.Delete {
					deletes.Data = append(deletes.Data, c.Key...)
				} else {
					sets.Data = append(sets.Data, c.Key...)
				}
			}
			if len(sets.Data) > 0 {
				frame = frame.Append(cfg.SetChannel.Key(), sets)
			}
			if len(deletes.Data) > 0 {
				frame = frame.Append(cfg.DeleteChannel.Key(), deletes)
			}
			return framer.WriterRequest{Command: writer.Write, Frame: frame}, true, nil
		},
	}
	p := plumber.New()
	plumber.SetSource(p, "source", t)
	plumber.SetSegment(p, "writer", w)
	responses := &confluence.UnarySink[framer.WriterResponse]{
		Sink: func(ctx context.Context, value framer.WriterResponse) error {
			s.L.Error("unexpected writer response", zap.Int("seqNum", value.SeqNum))
			return nil
		},
	}
	plumber.SetSink(p, "responses", responses)
	plumber.MustConnect[framer.WriterRequest](p, "source", "writer", 10)
	plumber.MustConnect[framer.WriterResponse](p, "writer", "responses", 10)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.Child(lo.Ternary(cfg.Name != "", cfg.Name, cfg.SetChannel.Name))))
	p.Flow(sCtx, confluence.CloseOutputInletsOnExit(), confluence.RecoverWithErrOnPanic())
	return signal.NewHardShutdown(sCtx, cancel), nil
}
