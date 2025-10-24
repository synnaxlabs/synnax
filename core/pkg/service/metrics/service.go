// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package metrics

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
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

type Config struct {
	// Instrumentation is used for logging, tracing, and metrics.
	alamos.Instrumentation
	// Channel is used to create and retrieve metric collection channels.
	//
	// [REQUIRED]
	Channel channel.Service
	// Framer is used to write metrics to the metric channels.
	//
	// [REQUIRED]
	Framer *framer.Service
	// HostProvider is for identify the current host for channel naming.
	//
	// [REQUIRED]
	HostProvider cluster.HostProvider
	// CollectionInterval sets the interval at which metrics will be collected from the
	// host machine.
	//
	// [OPTIONAL] - Defaults to 2s
	CollectionInterval time.Duration
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for a metrics service.
	DefaultConfig = Config{CollectionInterval: 2 * time.Second}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	c.CollectionInterval = override.Numeric(
		c.CollectionInterval,
		other.CollectionInterval,
	)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("service.metrics")
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "host_provider", c.HostProvider)
	validate.Positive(v, "collection_interval", c.CollectionInterval)
	return v.Error()
}

// Service is used to collect metrics from the host machine (cpu, memory, disk) and
// write them to channels.
type Service struct {
	stopCollector chan struct{}
	shutdown      io.Closer
}

const (
	writerAddr        address.Address = "writer"
	collectorAddr     address.Address = "collector"
	loggerAddr        address.Address = "logger"
	channelBufferSize                 = 10
)

// OpenService opens a new metric.Service using the provided configuration. See the
// Config struct for details on the required configuration values. If OpenService
// returns an error, the service is not safe to use. If OpenService succeeds, it must be
// shut down by calling Close after use.
func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{stopCollector: make(chan struct{})}
	nameBase := fmt.Sprintf("sy_node_%s_metrics_", cfg.HostProvider.HostKey())
	c := &collector{
		ins:      cfg.Child("collector"),
		interval: cfg.CollectionInterval,
		stop:     s.stopCollector,
		metrics:  make([]metric, len(all)),
	}
	c.idx = channel.Channel{
		Name:     nameBase + "time",
		DataType: telem.TimeStampT,
		IsIndex:  true,
	}
	if err := cfg.Channel.Create(
		ctx,
		&c.idx,
		channel.RetrieveIfNameExists(true),
	); err != nil {
		return nil, err
	}
	metricChannels := make([]channel.Channel, len(all))
	for i, metric := range all {
		metric.ch.Name = nameBase + metric.ch.Name
		metric.ch.LocalIndex = c.idx.LocalKey
		metricChannels[i] = metric.ch
	}
	if err := cfg.Channel.CreateMany(
		ctx,
		&metricChannels,
		channel.RetrieveIfNameExists(true),
	); err != nil {
		return nil, err
	}
	for i, ch := range metricChannels {
		c.metrics[i] = metric{ch: ch, collect: all[i].collect}
	}
	w, err := cfg.Framer.NewStreamWriter(
		ctx,
		framer.WriterConfig{
			Keys: append(
				channel.KeysFromChannels(metricChannels),
				c.idx.Key(),
			),
			Start:                    telem.Now(),
			AutoIndexPersistInterval: telem.Second * 30,
		},
	)
	if err != nil {
		return nil, err
	}

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	p := plumber.New()
	plumber.SetSegment(p, writerAddr, w)
	plumber.SetSource(p, collectorAddr, c)
	o := confluence.NewObservableSubscriber[framer.WriterResponse]()
	o.OnChange(func(ctx context.Context, response framer.WriterResponse) {
		if response.Err != nil {
			cfg.L.Error("failed to write metrics to node", zap.Error(response.Err))
		}
	})
	plumber.SetSink(p, loggerAddr, o)
	plumber.MustConnect[framer.WriterRequest](
		p,
		collectorAddr,
		writerAddr,
		channelBufferSize,
	)
	plumber.MustConnect[framer.WriterResponse](
		p,
		writerAddr,
		loggerAddr,
		channelBufferSize,
	)
	s.shutdown = signal.NewGracefulShutdown(sCtx, cancel)
	p.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	return s, nil
}

// Close gracefully stops the service, waiting for all internal goroutines to exit.
func (s *Service) Close() error {
	close(s.stopCollector)
	return s.shutdown.Close()
}
