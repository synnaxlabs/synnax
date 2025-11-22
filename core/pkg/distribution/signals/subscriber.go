// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type ObservableSubscriberConfig struct {
	SetChannelKey     channel.Key
	SetChannelName    string
	DeleteChannelKey  channel.Key
	DeleteChannelName string
}

var (
	_                                 config.Config[ObservableSubscriberConfig] = ObservableSubscriberConfig{}
	DefaultObservableSubscriberConfig                                           = ObservableSubscriberConfig{}
)

func (c ObservableSubscriberConfig) Override(other ObservableSubscriberConfig) ObservableSubscriberConfig {
	c.SetChannelKey = override.Numeric(c.SetChannelKey, other.SetChannelKey)
	c.SetChannelName = override.String(c.SetChannelName, other.SetChannelName)
	c.DeleteChannelKey = override.Numeric(c.DeleteChannelKey, other.DeleteChannelKey)
	c.DeleteChannelName = override.String(c.DeleteChannelName, other.DeleteChannelName)
	return c
}

func (c ObservableSubscriberConfig) Validate() error {
	v := validate.New("signals.ObservableSubscriberConfig")
	return v.Error()
}

func resolveChannelKey(
	ctx context.Context,
	key channel.Key,
	name string,
	svc *channel.Service,
) (channel.Key, error) {
	if key != 0 {
		return key, nil
	}
	if name == "" {
		return 0, nil
	}
	var ch channel.Channel
	err := svc.NewRetrieve().WhereNames(name).Entry(&ch).Exec(ctx, nil)
	return ch.Key(), err
}

func decodeChanges(
	variant change.Variant,
	series telem.MultiSeries,
) (changes []change.Change[[]byte, struct{}]) {
	for _, s := range series.Series {
		for sample := range s.Samples() {
			changes = append(changes, change.Change[[]byte, struct{}]{
				Variant: variant,
				Key:     sample,
			})
		}
	}
	return
}

func (s *Provider) Subscribe(
	ctx context.Context,
	configs ...ObservableSubscriberConfig,
) (observe.Observable[[]change.Change[[]byte, struct{}]], io.Closer, error) {
	var err error
	cfg, err := config.New(DefaultObservableSubscriberConfig, configs...)
	if err != nil {
		return nil, nil, err
	}
	cfg.SetChannelKey, err = resolveChannelKey(ctx, cfg.SetChannelKey, cfg.SetChannelName, s.Channel)
	if err != nil {
		return nil, nil, err
	}
	cfg.DeleteChannelKey, err = resolveChannelKey(ctx, cfg.DeleteChannelKey, cfg.DeleteChannelName, s.Channel)
	if err != nil {
		return nil, nil, err
	}
	keys := make([]channel.Key, 0, 2)
	if cfg.SetChannelKey != 0 {
		keys = append(keys, cfg.SetChannelKey)
	}
	if cfg.DeleteChannelKey != 0 {
		keys = append(keys, cfg.DeleteChannelKey)
	}
	streamer, err := s.Framer.NewStreamer(ctx, framer.StreamerConfig{
		Keys:        keys,
		SendOpenAck: config.True(),
	})
	if err != nil {
		return nil, nil, err
	}
	openAckChannel := make(chan struct{})
	opened := false
	obs := confluence.NewObservableTransformSubscriber(func(ctx context.Context, r framer.StreamerResponse) ([]change.Change[[]byte, struct{}], bool, error) {
		if !opened {
			opened = true
			close(openAckChannel)
		}
		var changes []change.Change[[]byte, struct{}]
		changes = append(changes, decodeChanges(change.Delete, r.Frame.Get(cfg.DeleteChannelKey))...)
		changes = append(changes, decodeChanges(change.Set, r.Frame.Get(cfg.SetChannelKey))...)
		return changes, len(changes) > 0, nil
	})
	p := plumber.New()
	plumber.SetSegment(p, "streamer", streamer)
	plumber.SetSink(p, "observable", obs)
	plumber.MustConnect[framer.StreamerResponse](p, "streamer", "observable", 10)
	inlet := confluence.NewStream[framer.StreamerRequest]()
	streamer.InFrom(inlet)
	sCtx, cancel := signal.Isolated()
	p.Flow(sCtx, confluence.CloseOutputInletsOnExit(), confluence.RecoverWithErrOnPanic())
	<-openAckChannel
	return obs, xio.CloserFunc(func() error {
		defer cancel()
		inlet.Close()
		return sCtx.Wait()
	}), nil
}
