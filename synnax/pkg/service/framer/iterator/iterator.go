// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	Framer  *framer.Service
	Channel channel.Readable
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.Framer = override.Nil(cfg.Framer, other.Framer)
	cfg.Channel = override.Nil(cfg.Channel, other.Channel)
	return cfg
}

func (cfg ServiceConfig) Validate() error {
	v := validate.New("iterator")
	validate.NotNil(v, "framer", cfg.Framer)
	validate.NotNil(v, "channel", cfg.Channel)
	return v.Error()
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	return &Service{cfg: cfg}, err
}

type Service struct{ cfg ServiceConfig }

type (
	Config   = framer.IteratorConfig
	Iterator = framer.StreamIterator
	Request  = framer.IteratorRequest
	Response = framer.IteratorResponse
)

type ResponseSegment = confluence.Segment[Response, Response]

func (s *Service) New(ctx context.Context, cfg Config) (Iterator, error) {
	p := plumber.New()
	t, err := s.newCalculationTransform(ctx, &cfg)
	if err != nil {
		return nil, err
	}
	dist, err := s.cfg.Framer.NewStreamIterator(ctx, cfg)
	if err != nil {
		return nil, err
	}
	plumber.SetSegment(p, "distribution", dist)
	var routeOutletFrom address.Address = "distribution"
	if t != nil {
		plumber.SetSegment(p, "calculation", t)
		plumber.MustConnect[Response](p, "distribution", "calculation", 25)
		routeOutletFrom = "calculation"
	}
	return &plumber.Segment[Request, Response]{
		Pipeline:         p,
		RouteInletsTo:    []address.Address{"distribution"},
		RouteOutletsFrom: []address.Address{routeOutletFrom},
	}, nil
}

func (s *Service) newCalculationTransform(ctx context.Context, cfg *Config) (ResponseSegment, error) {
	var (
		channels   []channel.Channel
		calculated = make(set.Set[channel.Key, channel.Channel], len(channels))
		required   = make(set.Set[channel.Key, channel.Channel], len(channels))
	)
	if err := s.cfg.Channel.NewRetrieve().
		WhereKeys(cfg.Keys...).
		Entries(&channels).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	for _, ch := range channels {
		if ch.IsCalculated() {
			calculated[ch.Key()] = ch
			required.Add(ch.Requires...)
		}
	}
	hasCalculated := len(calculated) > 0
	if !hasCalculated {
		return nil, nil
	}
	cfg.Keys = lo.Filter(cfg.Keys, func(item channel.Key, index int) bool {
		return !calculated.Contains(item)
	})
	cfg.Keys = append(cfg.Keys, required.Keys()...)
	var requiredCh []channel.Channel
	err := s.cfg.Channel.NewRetrieve().
		WhereKeys(required.Keys()...).
		Entries(&requiredCh).
		Exec(ctx, nil)
	if err != nil {
		return nil, err
	}
	calculators := make([]*calculation.Calculator, len(calculated))
	for i, v := range calculated.Values() {
		calculators[i], err = calculation.OpenCalculator(v, requiredCh)
	}
	return newCalculationTransform(calculators), nil
}
