// Copyright 2025 Synnax Labs, Inc.
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
	"go/types"
	"io"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ServiceConfig is the configuration for opening the calculation service.
type ServiceConfig struct {
	alamos.Instrumentation
	// Framer is the underlying frame service to stream cache channel values and write
	// calculated samples.
	// [REQUIRED]
	Framer *framer.Service
	// Channel is used to retrieve information about the channels being calculated.
	// [REQUIRED]
	Channel channel.Service
	Arc     *arc.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultConfig is the default configuration for opening the calculation service.
	DefaultConfig = ServiceConfig{}
)

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("calculate")
	validate.NotNil(v, "Framer", c.Framer)
	validate.NotNil(v, "Channel", c.Channel)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	return c
}

// entry is used to manage the lifecycle of a calculation.
type entry struct {
	// channel is the calculated channel.
	ch channel.Channel
	// count is the number of active requests for the calculation.
	count int
}

type Status = status.Status[types.Nil]

// Service creates and operates calculations on channels.
type Service struct {
	cfg ServiceConfig
	mu  struct {
		sync.Mutex
		entries map[channel.Key]*entry
	}
	disconnectFromChannelChanges observe.Disconnect
	stateKey                     channel.Key
	w                            *framer.Writer
}

const StatusChannelName = "sy_calculation_status"

// OpenService opens the service with the provided configuration. The service must be closed
// when it is no longer needed.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}

	calculationStateCh := channel.Channel{
		Name:        StatusChannelName,
		DataType:    telem.JSONT,
		Virtual:     true,
		Leaseholder: cluster.Free,
		Internal:    true,
	}

	if err = cfg.Channel.MapRename(ctx, map[string]string{
		"sy_calculation_state": StatusChannelName,
	}, true); err != nil {
		return nil, err
	}

	if err = cfg.Channel.Create(
		ctx,
		&calculationStateCh,
		channel.RetrieveIfNameExists(true),
	); err != nil {
		return nil, err
	}

	w, err := cfg.Framer.OpenWriter(ctx, framer.WriterConfig{
		Keys:        []channel.Key{calculationStateCh.Key()},
		Start:       telem.Now(),
		Authorities: []control.Authority{255},
	})
	if err != nil {
		return nil, err
	}

	s := &Service{cfg: cfg, w: w, stateKey: calculationStateCh.Key()}
	s.mu.entries = make(map[channel.Key]*entry)

	return s, nil
}

func (s *Service) setStatus(
	_ context.Context,
	status Status,
) {
	if _, err := s.w.Write(core.UnaryFrame(
		s.stateKey,
		telem.NewSeriesStaticJSONV(status),
	)); err != nil {
		s.cfg.L.Error("failed to encode state", zap.Error(err))
	}
}

func (s *Service) releaseEntryCloser(ctx context.Context, key channel.Key) io.Closer {
	return xio.CloserFunc(func() (err error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		e, found := s.mu.entries[key]
		if !found {
			return
		}
		e.count--
		if e.count != 0 {
			return
		}
		s.cfg.L.Debug("closing calculated channel", zap.Stringer("key", key))
		if err := s.cfg.Arc.Stop(ctx, e.ch.Calculation); err != nil {
			s.cfg.L.Error("failed to close calculation channel", zap.Error(err))
		}
		delete(s.mu.entries, key)
		return
	})
}

// Close stops all calculations and closes the service. No other methods should be
// called after Close.
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disconnectFromChannelChanges()
	c := errors.NewCatcher(errors.WithAggregation())
	c.Exec(s.w.Close)
	return c.Error()
}

// Request requests that the Service starts calculation the channel with the provided
// key. The calculation will be started if the channel is calculated and not already
// being calculated. If the channel is already being calculated, the number of active
// requests will be increased. The caller must close the returned io.Closer when the
// calculation is no longer needed, which will decrement the number of active requests.
func (s *Service) Request(ctx context.Context, key channel.Key) (io.Closer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startCalculation(ctx, key, 1)
}

func (s *Service) startCalculation(
	ctx context.Context,
	key channel.Key,
	initialCount int,
) (io.Closer, error) {
	var ch channel.Channel
	ch.LocalKey = key.LocalKey()
	ch.Leaseholder = key.Leaseholder()
	if err := s.cfg.Channel.NewRetrieve().WhereKeys(key).Entry(&ch).Exec(ctx, nil); err != nil {
		return nil, err
	}
	if !ch.IsCalculated() {
		return nil, errors.Wrapf(validate.Error, "channel %v is not calculated", ch)
	}
	if _, exists := s.mu.entries[key]; exists {
		s.mu.entries[key].count++
		return s.releaseEntryCloser(ctx, key), nil
	}
	if err := s.cfg.Arc.Deploy(ctx, ch.Calculation); err != nil {
		return nil, nil
	}
	s.mu.entries[key] = &entry{ch: ch, count: initialCount}
	return s.releaseEntryCloser(ctx, key), nil
}
