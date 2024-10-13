// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/computron"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	Framer  *framer.Service
	Channel channel.Readable
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("framer")
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	return c
}

type Service struct{ Config }

func (s *Service) OpenIterator(ctx context.Context, cfg framer.IteratorConfig) (*framer.Iterator, error) {
	return s.Framer.OpenIterator(ctx, cfg)
}

func (s *Service) NewStreamIterator(ctx context.Context, cfg framer.IteratorConfig) (framer.StreamIterator, error) {
	return s.Framer.NewStreamIterator(ctx, cfg)
}

func (s *Service) NewStreamWriter(ctx context.Context, cfg framer.WriterConfig) (framer.StreamWriter, error) {
	return s.Framer.NewStreamWriter(ctx, cfg)
}

func (s *Service) NewDeleter() framer.Deleter {
	return s.Framer.NewDeleter()
}

func (s *Service) NewStreamer(ctx context.Context, cfg framer.StreamerConfig) (framer.Streamer, error) {
	keys, err := channel.RetrieveRequiredKeys2(ctx, s.Channel, cfg.Keys)
	if err != nil {
		return nil, err
	}
	ch := make([]channel.Channel, 0)
	if err := s.Channel.NewRetrieve().WhereKeys(keys...).Entries(&ch).Exec(ctx, nil); err != nil {
		return nil, err
	}
	calc := calculator{channels: ch}
	sc := &streamCalculator{internal: calc}
	sc.Transform = sc.transform
	inter, err := s.Framer.NewStreamer(ctx, cfg)
	if err != nil {
		return inter, err
	}
	p := plumber.New()
	plumber.SetSegment[framer.StreamerRequest, framer.StreamerResponse](p, "internal", inter)
	plumber.SetSegment[framer.StreamerResponse, framer.StreamerResponse](p, "transform", sc)
	plumber.MustConnect[framer.StreamerResponse](p, "internal", "transform", 10)
	seg := &plumber.Segment[framer.StreamerRequest, framer.StreamerResponse]{
		Pipeline:         p,
		RouteInletsTo:    []address.Address{"internal"},
		RouteOutletsFrom: []address.Address{"transform"},
	}
	return seg, nil
}

func NewService(cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	return &Service{Config: cfg}, err
}

type streamCalculator struct {
	internal calculator
	confluence.LinearTransform[framer.StreamerResponse, framer.StreamerResponse]
}

func (s *streamCalculator) transform(ctx context.Context, i framer.StreamerResponse) (framer.StreamerResponse, bool, error) {
	i.Frame = s.internal.transform(i.Frame)
	return i, true, nil
}

type calculator struct {
	channels []channel.Channel
}

func (c calculator) transform(fr framer.Frame) framer.Frame {
	for _, ch := range c.channels {
		if ch.IsCalculated() {
			fr = c.calculate(ch, fr)
		}
	}
	return fr
}

func (c calculator) calculate(ch channel.Channel, fr framer.Frame) framer.Frame {
	globals := make(map[string]interface{})
	for _, k := range ch.Requires {
		s := fr.Get(k)
		if len(s) == 0 {
			continue
		}
		obj, err := main.New(s[0])
		if err != nil {
			continue
		}
		ch, found := lo.Find(c.channels, func(ch channel.Channel) bool {
			return ch.Key() == k
		})
		if !found {
			continue
		}
		globals[ch.Name] = obj
	}
	os, err := main.Exec(ch.Expression, globals)
	if err != nil {
		return fr
	}
	os.TimeRange = fr.Series[0].TimeRange
	os.Alignment = fr.Series[0].Alignment
	fr.Series = append(fr.Series, os)
	fr.Keys = append(fr.Keys, ch.Key())
	return fr
}
