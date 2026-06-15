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

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
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

// ObservablePublisherConfig is the configuration for opening a Signals pipeline that
// subscribes to the provided observable and writes changes to the provided channels.
// Higher level Signals pipeline should be preferred, such as the PublishFromGorp.
type ObservablePublisherConfig struct {
	// Observable is the observable used to subscribe to changes. This observable should
	// return byte slice keys that are properly encoded for the channel's data type.
	Observable observe.Observable[[]change.Change[[]byte, struct{}]]
	// Name is an optional name for the Signals pipeline, used for debugging purposes.
	Name string
	// SetChannel is the channel used to propagate set operations. Only Name and
	// DataType need to be provided. The config will automatically set Leaseholder to
	// Free and Virtual to true. Leave Name empty to disable the set channel; in that
	// case VariantSet events from the observable are dropped.
	SetChannel channel.Channel
	// DeleteChannel is the channel used to propagate delete operations. Only Name and
	// DataType need to be provided. The config will automatically set Leaseholder to
	// Free and Virtual to true. Leave Name empty to disable the delete channel; in that
	// case VariantDelete events from the observable are dropped.
	DeleteChannel channel.Channel
}

var _ config.Config[ObservablePublisherConfig] = ObservablePublisherConfig{}

const (
	nonVirtual = "Signals can only work with virtual free channels. Received false for %s"
	nonFree    = "Signals can only work with free channels. Received leaseholder %s that is not equal to Free"
)

// Validate implements config.Config.
func (c ObservablePublisherConfig) Validate() error {
	v := validate.New("signals.observable_publisher_config")
	v.Ternary(
		"channels",
		c.SetChannel.Name == "" && c.DeleteChannel.Name == "",
		"at least one of set_channel or delete_channel must be provided",
	)
	if c.SetChannel.Name != "" {
		v.Ternaryf("set_channel.leaseholder", !c.SetChannel.Free(), nonFree, c.SetChannel.Leaseholder)
		v.Ternaryf("set_channel.virtual", !c.SetChannel.Virtual, nonVirtual, c.SetChannel.Name)
	}
	if c.DeleteChannel.Name != "" {
		v.Ternaryf(
			"delete_channel.leaseholder",
			!c.DeleteChannel.Free(),
			nonFree,
			c.DeleteChannel.Leaseholder,
		)
		v.Ternaryf(
			"delete_channel.virtual",
			!c.DeleteChannel.Virtual,
			nonVirtual,
			c.DeleteChannel.Name,
		)
	}
	validate.NotNil(v, "observable", c.Observable)
	return v.Error()
}

// Override implements config.Config.
func (c ObservablePublisherConfig) Override(other ObservablePublisherConfig) ObservablePublisherConfig {
	c.Name = override.If(c.Name, other.Name, c.Name == "")
	c.SetChannel = override.If(c.SetChannel, other.SetChannel, c.SetChannel.Name == "")
	c.DeleteChannel = override.If(
		c.DeleteChannel, other.DeleteChannel, c.DeleteChannel.Name == "",
	)
	c.Observable = override.Nil(c.Observable, other.Observable)
	if c.SetChannel.Name != "" {
		c.SetChannel.Virtual = true
		c.SetChannel.Leaseholder = node.KeyFree
	}
	if c.DeleteChannel.Name != "" {
		c.DeleteChannel.Virtual = true
		c.DeleteChannel.Leaseholder = node.KeyFree
	}
	return c
}

// PublishFromObservable opens a new Signals pipeline that subscribes to the configured
// ObservableSubscriber and writes changes to the configured channels. The returned
// io.Closer can be used to close the pipeline when done.
func (s *Provider) PublishFromObservable(
	ctx context.Context,
	cfgs ...ObservablePublisherConfig,
) (io.Closer, error) {
	cfg, err := config.New(ObservablePublisherConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	setEnabled := cfg.SetChannel.Name != ""
	deleteEnabled := cfg.DeleteChannel.Name != ""
	var channels []channel.Channel
	if setEnabled {
		channels = append(channels, cfg.SetChannel)
	}
	if deleteEnabled {
		channels = append(channels, cfg.DeleteChannel)
	}
	if err = s.Channel.CreateMany(
		ctx,
		&channels,
		channel.RetrieveIfNameExists(),
		channel.OverwriteIfNameExistsAndDifferentProperties(),
	); err != nil {
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
		switch ch.Name {
		case cfg.SetChannel.Name:
			cfg.SetChannel = ch
		case cfg.DeleteChannel.Name:
			cfg.DeleteChannel = ch
		}
	}
	t := &confluence.ObservableTransformPublisher[
		[]change.Change[[]byte, struct{}], framer.WriterRequest,
	]{
		Instrumentation: s.Instrumentation,
		Observable:      cfg.Observable,
		Transform: func(
			_ context.Context, changes []change.Change[[]byte, struct{}],
		) (framer.WriterRequest, bool, error) {
			if len(changes) == 0 {
				return framer.WriterRequest{}, false, nil
			}
			var (
				frame   framer.Frame
				sets    = telem.Series{DataType: cfg.SetChannel.DataType}
				deletes = telem.Series{DataType: cfg.DeleteChannel.DataType}
			)
			for _, ch := range changes {
				if ch.Variant == change.VariantDelete {
					if !deleteEnabled {
						continue
					}
					deletes.Data = append(deletes.Data, ch.Key...)
				} else {
					if !setEnabled {
						continue
					}
					sets.Data = append(sets.Data, ch.Key...)
				}
			}
			if len(sets.Data) > 0 {
				frame = frame.Append(cfg.SetChannel.Key(), sets)
			}
			if len(deletes.Data) > 0 {
				frame = frame.Append(cfg.DeleteChannel.Key(), deletes)
			}
			if len(sets.Data) == 0 && len(deletes.Data) == 0 {
				return framer.WriterRequest{}, false, nil
			}
			return framer.WriterRequest{Command: writer.CommandWrite, Frame: frame},
				true, nil
		},
	}
	p := plumber.New()
	plumber.SetSource(p, "source", t)
	plumber.SetSegment(p, "writer", w)
	responses := &confluence.UnarySink[framer.WriterResponse]{
		Sink: func(_ context.Context, value framer.WriterResponse) error {
			s.L.Error("unexpected writer response", zap.Int("seqNum", value.SeqNum))
			return nil
		},
	}
	plumber.SetSink(p, "responses", responses)
	plumber.MustConnect[framer.WriterRequest](p, "source", "writer", 10)
	plumber.MustConnect[framer.WriterResponse](p, "writer", "responses", 10)
	name := cfg.Name
	if name == "" {
		if setEnabled {
			name = cfg.SetChannel.Name
		} else {
			name = cfg.DeleteChannel.Name
		}
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.Child(name)))
	p.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.RecoverWithErrOnPanic(),
	)
	return signal.NewHardShutdown(sCtx, cancel), nil
}
