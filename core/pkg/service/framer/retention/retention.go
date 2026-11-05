// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package retention deletes the data of index channels that is older than their
// retention.
package retention

import (
	"context"
	"io"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Config is the configuration for opening a Service.
type Config struct {
	alamos.Instrumentation
	// Channel finds the index channels that have a retention.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer deletes the old data.
	//
	// [REQUIRED]
	Framer *framer.Service
	// HostProvider gives the key of this Core. The Service deletes data only for the
	// index channels that this Core leases.
	//
	// [REQUIRED]
	HostProvider cluster.HostProvider
	// Interval is the time between two passes over the index channels.
	//
	// [OPTIONAL] - Defaults to 1 minute.
	Interval telem.TimeSpan
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for a Service.
	DefaultConfig = Config{Interval: telem.Minute}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.HostProvider = override.Nil(c.HostProvider, other.HostProvider)
	c.Interval = override.Numeric(c.Interval, other.Interval)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("retention")
	v.NotNil("channel", c.Channel)
	v.NotNil("framer", c.Framer)
	v.NotNil("host_provider", c.HostProvider)
	v.Positive("interval", c.Interval)
	return v.Error()
}

// Service deletes, once each interval, the data of each index channel that this Core
// leases and that is older than the channel's retention. The delete includes every
// channel on the index, and stops at the oldest open writer on those channels.
type Service struct {
	shutdown io.Closer
	cfg      Config
}

// Open starts a Service. Close it to stop.
func Open(cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	s.shutdown = signal.NewHardShutdown(sCtx, cancel)
	signal.GoTick(sCtx, cfg.Interval.Duration(), s.delete, signal.WithKey("retention"))
	return s, nil
}

// delete never returns an error, because a failed pass must not stop the next one.
func (s *Service) delete(ctx context.Context, now time.Time) error {
	var indexes []channel.Channel
	if err := s.cfg.Channel.NewRetrieve().
		Where(channel.And(
			channel.MatchLeaseholders(s.cfg.HostProvider.HostKey()),
			channel.Match(func(
				_ gorp.Context,
				_ channel.Retrieve,
				ch *channel.Channel,
			) (bool, error) {
				return ch.Retention > 0, nil
			}),
		)).
		Entries(&indexes).
		Exec(ctx, nil); err != nil {
		s.cfg.L.Error("failed to retrieve index channels", zap.Error(err))
		return nil
	}
	for _, index := range indexes {
		cutoff := telem.NewTimeStamp(now).Add(-index.Retention)
		if err := s.cfg.Framer.DeleteBefore(ctx, index.Key(), cutoff); err != nil {
			s.cfg.L.Error(
				"failed to delete old data",
				zap.Stringer("channel", index),
				zap.Error(err),
			)
		}
	}
	return nil
}

// Close stops the Service and waits for a running pass to end.
func (s *Service) Close() error { return s.shutdown.Close() }
