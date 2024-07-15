// Copyright 2024 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
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
	svc channel.Readable,
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

func decodeChanges(variant change.Variant, series []telem.Series) (changes []change.Change[[]byte, struct{}]) {
	for _, s := range series {
		for _, k := range s.Split() {
			changes = append(changes, change.Change[[]byte, struct{}]{
				Variant: variant,
				Key:     k,
			})
		}
	}
	return
}

func (s *Provider) Subscribe(
	sCtx signal.Context,
	configs ...ObservableSubscriberConfig,
) (observe.Observable[[]change.Change[[]byte, struct{}]], error) {
	var err error
	cfg, err := config.New(DefaultObservableSubscriberConfig, configs...)
	if err != nil {
		return nil, err
	}
	cfg.SetChannelKey, err = resolveChannelKey(sCtx, cfg.SetChannelKey, cfg.SetChannelName, s.Channel)
	if err != nil {
		return nil, err
	}
	cfg.DeleteChannelKey, err = resolveChannelKey(sCtx, cfg.DeleteChannelKey, cfg.DeleteChannelName, s.Channel)
	if err != nil {
		return nil, err
	}
	keys := make([]channel.Key, 0, 2)
	if cfg.SetChannelKey != 0 {
		keys = append(keys, cfg.SetChannelKey)
	}
	if cfg.DeleteChannelKey != 0 {
		keys = append(keys, cfg.DeleteChannelKey)
	}
	streamer, err := s.Framer.NewStreamer(sCtx, framer.StreamerConfig{
		Keys:  keys,
		Start: telem.TimeStampMax,
	})
	if err != nil {
		return nil, err
	}
	obs := confluence.NewObservableTransformSubscriber(func(ctx context.Context, r framer.StreamerResponse) ([]change.Change[[]byte, struct{}], bool, error) {
		var changes []change.Change[[]byte, struct{}]
		changes = append(changes, decodeChanges(change.Delete, r.Frame.Get(cfg.DeleteChannelKey))...)
		changes = append(changes, decodeChanges(change.Set, r.Frame.Get(cfg.SetChannelKey))...)
		return changes, len(changes) > 0, nil
	})
	p := plumber.New()
	plumber.SetSegment[framer.StreamerRequest, framer.StreamerResponse](p, "streamer", streamer)
	plumber.SetSink[framer.StreamerResponse](p, "observable", obs)
	plumber.MustConnect[framer.StreamerResponse](p, "streamer", "observable", 10)
	streamer.InFrom(confluence.NewStream[framer.StreamerRequest]())
	p.Flow(sCtx, confluence.CloseInletsOnExit(), confluence.RecoverWithErrOnPanic())
	return obs, nil
}
