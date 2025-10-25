// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/arc/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Arc service.
type ServiceConfig struct {
	// Instrumentation is used for logging, tracing, and metrics.
	alamos.Instrumentation
	// DB is the database that the Arc service will store arcs in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between arcs and other entities in
	// the Synnax resource graph.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Channel is used for retrieving channel information from the cluster.
	//
	// [REQUIRED]
	Channel channel.Readable
	// Framer is used for reading and writing telemetry frames to/from the cluster.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Status is used for creating arc-related statuses
	//
	// [REQUIRED]
	Status *status.Service
	// Signals is used for propagating changes to arcs through the cluster.
	//
	// [OPTIONAL] - Defaults to nil. Signals will not be propagated if this service
	// is nil.
	Signals *signals.Provider
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultConfig is the default configuration for opening a Arc service.
	DefaultConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("Arc")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "status", c.Status)
	return v.Error()
}

func (c ServiceConfig) baseRuntimeConfig() runtime.Config {
	return runtime.Config{Channel: c.Channel, Framer: c.Framer, Status: c.Status}
}

// Service is the primary service for retrieving and modifying arcs from Synnax.
type Service struct {
	cfg            ServiceConfig
	symbolResolver arc.SymbolResolver
	mu             struct {
		sync.Mutex
		entries map[uuid.UUID]*entry
		closer  io.Closer
	}
}

func (s *Service) SymbolResolver() arc.SymbolResolver {
	return s.symbolResolver
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := errors.NewCatcher(errors.WithAggregation())
	c.Exec(s.mu.closer.Close)
	for _, e := range s.mu.entries {
		c.Exec(e.runtime.Close)
	}
	return c.Error()
}

func (s *Service) Deploy(ctx context.Context, key uuid.UUID) error {
	var prog Arc
	if err := s.NewRetrieve().WhereKeys(key).Entry(&prog).Exec(ctx, nil); err != nil {
		return nil
	}
	prog.Deploy = true
	return s.NewWriter(nil).Create(ctx, &prog)
}

func (s *Service) Stop(ctx context.Context, key uuid.UUID) error {
	var prog Arc
	if err := s.NewRetrieve().WhereKeys(key).Entry(&prog).Exec(ctx, nil); err != nil {
		return nil
	}
	prog.Deploy = false
	return s.NewWriter(nil).Create(ctx, &prog)
}

func (s *Service) AnalyzeCalculation(ctx context.Context, expr string) (telem.DataType, error) {
	t, err := parser.ParseBlock(fmt.Sprintf("{%s}", expr))
	if err != nil {
		return telem.UnknownT, err
	}
	aCtx := acontext.CreateRoot(
		ctx,
		t,
		s.SymbolResolver(),
	)
	dataType, ok := statement.AnalyzeFunctionBody(aCtx)
	if !ok {
		return telem.UnknownT, aCtx.Diagnostics.Error()
	}
	return types.ToTelem(dataType), nil
}

// OpenService instantiates a new Arc service using the provided configurations. Each
// configuration will be used as an override for the previous configuration in the list.
// See the ConfigValues struct for information on which fields should be set.
func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	var (
		closer xio.MultiCloser
		s      = &Service{cfg: cfg}
	)
	s.symbolResolver, err = symbol.CreateResolver(cfg.baseRuntimeConfig())
	if err != nil {
		return nil, err
	}
	s.mu.entries = make(map[uuid.UUID]*entry)
	cfg.Ontology.RegisterService(s)
	if cfg.Signals != nil {
		stopSignals, err := signals.PublishFromGorp(ctx, s.cfg.Signals, signals.GorpPublisherConfigUUID[Arc](cfg.DB))
		if err != nil {
			return nil, err
		}
		closer = append(closer, stopSignals)
	}
	obs := gorp.Observe[uuid.UUID, Arc](cfg.DB)
	closer = append(closer, xio.NopCloserFunc(obs.OnChange(s.handleChange)))
	s.mu.closer = closer
	return s, nil
}

// NewWriter opens a new writer for creating, updating, and deleting arcs in Synnax. If
// tx is provided, the writer will use that transaction. If tx is nil, the Writer
// will execute the operations directly on the underlyinu gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:     gorp.OverrideTx(s.cfg.DB, tx),
		otg:    s.cfg.Ontology.NewWriter(tx),
		status: s.cfg.Status.NewWriter(tx),
	}
}

// NewRetrieve opens a new query builder for retrieving arcs from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[uuid.UUID, Arc](),
		baseTX: s.cfg.DB,
		otg:    s.cfg.Ontology,
	}
}
