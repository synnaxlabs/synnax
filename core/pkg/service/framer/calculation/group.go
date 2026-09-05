// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/synnax/pkg/service/framer/writer"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type group struct {
	shutdown         io.Closer
	streamerRequests confluence.Inlet[framer.StreamerRequest]
	Calculators      []*calculator.Calculator
}

func (g *group) Close() error {
	g.streamerRequests.Close()
	return g.shutdown.Close()
}

type OnStatusChange = func(context.Context, ...calculator.Status)

type groupConfig struct {
	alamos.Instrumentation
	framer         *framer.Service
	writer         *writer.Service
	onStatusChange OnStatusChange
	calculators    calculator.Group
}

var _ config.Config[groupConfig] = (*groupConfig)(nil)

func (c groupConfig) Override(other groupConfig) groupConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.framer = override.Nil(c.framer, other.framer)
	c.writer = override.Nil(c.writer, other.writer)
	c.calculators = override.Slice(c.calculators, other.calculators)
	c.onStatusChange = override.Nil(c.onStatusChange, other.onStatusChange)
	return c
}

func (c groupConfig) Validate() error {
	v := validate.New("calculation.group.config")
	v.NotNil("framer", c.framer)
	v.NotNil("writer", c.writer)
	v.NotEmptySlice("calculators", c.calculators)
	v.NotNil("on_status_change", c.onStatusChange)
	return v.Error()
}

const (
	defaultPipelineBufferSize                 = 500
	streamerAddr              address.Address = "streamer"
	calculatorAddr            address.Address = "calculator"
	writerAddr                address.Address = "writer"
	writerObserverAddr        address.Address = "writer_observer"
)

func openGroup(ctx context.Context, cfgs ...groupConfig) (*group, error) {
	cfg, err := config.New(groupConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}

	readKeys := cfg.calculators.ReadFrom()
	writeKeys := cfg.calculators.WriteTo()

	cfg.L.Debug("opening group pipeline",
		zap.Int("calculator_count", len(cfg.calculators)),
		zap.Int("read_channel_count", len(readKeys)),
		zap.Int("write_channel_count", len(writeKeys)),
	)

	strm, err := cfg.framer.NewStreamer(framer.StreamerConfig{Keys: readKeys})
	if err != nil {
		return nil, err
	}

	wrt, err := cfg.writer.NewStream(ctx, writer.Config{
		Keys:  writeKeys,
		Start: telem.Now(),
	})
	if err != nil {
		return nil, err
	}

	p := plumber.New()
	p.SetSource[framer.StreamerResponse](streamerAddr, strm)
	p.SetSegment[framer.WriterRequest, framer.WriterResponse](writerAddr, wrt)

	streamerRequests := confluence.NewStream[framer.StreamerRequest](10)
	strm.InFrom(streamerRequests)
	c := &transform{
		streamerRequests: streamerRequests,
		calculators:      cfg.calculators,
		onStatusChange:   cfg.onStatusChange,
	}

	p.SetSegment[framer.StreamerResponse, framer.WriterRequest](calculatorAddr, c)
	o := confluence.NewObservableSubscriber[framer.WriterResponse]()
	o.OnChange(func(ctx context.Context, res framer.WriterResponse) {
		cfg.L.DPanic("write of calculated channel value failed", zap.Error(res.Err))
	})
	p.SetSink[framer.WriterResponse](writerObserverAddr, o)
	p.MustConnect[framer.StreamerResponse](
		streamerAddr,
		calculatorAddr,
		defaultPipelineBufferSize,
	)
	p.MustConnect[framer.WriterRequest](
		calculatorAddr,
		writerAddr,
		defaultPipelineBufferSize,
	)
	p.MustConnect[framer.WriterResponse](
		writerAddr,
		writerObserverAddr,
		defaultPipelineBufferSize,
	)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	p.Flow(sCtx, confluence.CloseOutputInletsOnExit(), confluence.WithRetryOnPanic())

	cfg.L.Debug("group pipeline opened successfully",
		zap.Int("calculator_count", len(cfg.calculators)),
	)

	return &group{
		shutdown:         signal.NewGracefulShutdown(sCtx, cancel),
		streamerRequests: streamerRequests,
		Calculators:      cfg.calculators,
	}, nil
}
