// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Group struct {
	shutdown                    io.Closer
	addRequests, removeRequests chan *calculator.Calculator
	streamerRequests            confluence.Inlet[framer.StreamerRequest]
}

func (g *Group) Add(ctx context.Context, c *calculator.Calculator) error {
	return signal.SendUnderContext(ctx, g.addRequests, c)
}

func (g *Group) Remove(c *calculator.Calculator) error {
	g.removeRequests <- c
}

func (g *Group) Close() error {
	g.streamerRequests.Close()
	return g.shutdown.Close()
}

type OnStatusChange = func(ctx context.Context, stats ...calculator.Status)

type Config struct {
	alamos.Instrumentation
	Framer         *framer.Service
	OnStatusChange OnStatusChange
}

var (
	_             config.Config[Config] = (*Config)(nil)
	DefaultConfig                       = Config{}
)

func (c Config) Override(other Config) Config {
	c.Framer = override.Nil(c.Framer, other.Framer)
	return c
}

func (c Config) Validate() error {
	v := validate.New("calculation.group.config")
	validate.NotNil(v, "framer", c.Framer)
	return v.Error()
}

const (
	defaultPipelineBufferSize                 = 500
	streamerAddr              address.Address = "streamer"
	calculatorAddr            address.Address = "calculator"
	writerAddr                address.Address = "writer"
	writerObserverAddr        address.Address = "writer_observer"
)

func Open(ctx context.Context, cfgs ...Config) (*Group, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	strm, err := cfg.Framer.NewStreamer(ctx, framer.StreamerConfig{})
	if err != nil {
		return nil, err
	}
	wrt, err := cfg.Framer.NewStreamWriter(ctx, framer.WriterConfig{
		Keys:  channel.Keys{},
		Start: telem.Now(),
	})
	if err != nil {
		return nil, err
	}

	p := plumber.New()
	plumber.SetSource[framer.StreamerResponse](p, streamerAddr, strm)
	plumber.SetSegment[framer.WriterRequest, framer.WriterResponse](p, writerAddr, wrt)

	streamerRequests := confluence.NewStream[framer.StreamerRequest](10)
	addRequests := make(chan *calculator.Calculator, 1)
	removeRequests := make(chan *calculator.Calculator, 1)
	strm.InFrom(streamerRequests)
	c := &transform{
		base:             &calculator.Group{},
		addRequests:      addRequests,
		removeRequests:   removeRequests,
		streamerRequests: streamerRequests,
		onStatusChange:   cfg.OnStatusChange,
	}

	plumber.SetSegment[framer.StreamerResponse, framer.WriterRequest](
		p,
		calculatorAddr,
		c,
	)
	o := confluence.NewObservableSubscriber[framer.WriterResponse]()
	o.OnChange(func(ctx context.Context, i framer.WriterResponse) {
		cfg.L.DPanic("write of calculated channel value failed")
	})
	plumber.SetSink[framer.WriterResponse](p, writerObserverAddr, o)
	plumber.MustConnect[framer.StreamerResponse](p, streamerAddr, calculatorAddr, defaultPipelineBufferSize)
	plumber.MustConnect[framer.WriterRequest](p, calculatorAddr, writerAddr, defaultPipelineBufferSize)
	plumber.MustConnect[framer.WriterResponse](p, writerAddr, writerObserverAddr, defaultPipelineBufferSize)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	p.Flow(sCtx, confluence.CloseOutputInletsOnExit(), confluence.WithRetryOnPanic())
	return &Group{
		shutdown:         signal.NewGracefulShutdown(sCtx, cancel),
		streamerRequests: streamerRequests,
		addRequests:      addRequests,
		removeRequests:   removeRequests,
	}, nil
}
