// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Config the configuration for opening an Iterator or StreamIterator.
type Config struct {
	// Keys are the keys of the channels to iterator over. At least one key must be
	// specified. An iterator cannot iterate over non-calculated virtual channels or
	// free channels, and calls to Open or NewStream will return an error when
	// attempting to iterate over channels of these types.
	//
	// [REQUIRED] - must have at least one key.
	Keys channel.Keys
	// Bounds sets the time range to iterate over. This time range must be valid i.e.,
	// the start value must be before or equal to the end value.
	//
	// [REQUIRED]
	Bounds telem.TimeRange
	// ChunkSize sets the default number of samples to iterate over per-channel when
	// calling Next or Prev with AutoSpan.
	//
	// [OPTIONAL]
	ChunkSize int64
	// Reduction reduces the samples of each channel read from storage.
	//
	// [OPTIONAL]
	Reduction telem.Reduction
}

// Validate ensures that Keys is non-empty and contains no free channels, which cannot
// be iterated over as they have no underlying storage.
func (cfg Config) Validate() error {
	v := validate.New("distribution.framer.iterator")
	if v.NotEmptySlice("keys", cfg.Keys) {
		return v.Error()
	}
	for _, k := range cfg.Keys {
		if k.Free() {
			return errors.Wrapf(
				validate.ErrValidation,
				"cannot read from free channel %v",
				k,
			)
		}
	}
	return v.Error()
}

// ServiceConfig is the configuration for opening the iterator Service, the main
// entrypoint for using iterators.
type ServiceConfig struct {
	// TS is the underlying storage layer time-series database for reading frames.
	//
	// [REQUIRED]
	TS *ts.DB
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL] - defaults to noop instrumentation.
	alamos.Instrumentation
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	return cfg
}

// Validate implements Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.iterator")
	v.NotNil("ts", cfg.TS)
	return v.Error()
}

// Service is the distribution layer entry point for using iterators within Synnax.
// Iterators allow for reading chunks of historical data from channels.
type Service struct{ cfg ServiceConfig }

// NewService opens a new iterator service using the provided configuration. If the
// configuration is invalid, NewService returns a nil service and an error.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

// Open opens a new iterator for reading historical data from a Synnax cluster.
// If the returned error is nil, the iterator must be closed after use. For
// information on configuration parameters, see the IteratorConfig struct.
//
// The returned iterator uses a synchronous, method-based model. For a channel-based
// iterator model, use NewStream.
func (s *Service) Open(ctx context.Context, cfg Config) (*Iterator, error) {
	stream, err := s.NewStream(ctx, cfg)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(s.cfg.Instrumentation))
	req := confluence.NewStream[Request]()
	res := confluence.NewStream[Response]()
	stream.InFrom(req)
	stream.OutTo(res)
	stream.Flow(
		sCtx,
		confluence.CloseOutputInletsOnExit(),
		confluence.CancelOnFail(),
		confluence.RecoverWithErrOnPanic(),
	)
	return &Iterator{pipeline: Pipeline{
		Requests:  req,
		Responses: res,
		Shutdown:  cancel,
		WaitGroup: sCtx,
	}}, nil
}

// NewStream returns an iterator for reading historical data from a Synnax cluster. The
// returned StreamIterator is a confluence.Segment that uses a channel-based interface,
// where requests are sent through an input stream, and responses are received through
// an output stream.
func (s *Service) NewStream(_ context.Context, cfg Config) (StreamIterator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.Keys = cfg.Keys.Unique()
	return s.newStorage(cfg)
}
