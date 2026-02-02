// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/arc"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/arc/symbol"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a Arc service.
type ServiceConfig struct {
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
	Channel *channel.Service
	// Task is used for deleting tasks associated with arcs when arcs are deleted.
	//
	// [REQUIRED]
	Task *task.Service
	// Signals is used for propagating changes to arcs through the cluster.
	//
	// [OPTIONAL] - Defaults to nil. Signals will not be propagated if this service
	// is nil.
	Signals *signals.Provider
	// Instrumentation is used for logging, tracing, and metrics.
	alamos.Instrumentation
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening a Arc service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Task = override.Nil(c.Task, other.Task)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("arc")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "task", c.Task)
	return v.Error()
}

// Service is the primary service for retrieving and modifying arcs from Synnax.
type Service struct {
	symbolResolver arc.SymbolResolver
	closer         io.Closer
	cfg            ServiceConfig
}

func (s *Service) SymbolResolver() arc.SymbolResolver {
	return s.symbolResolver
}

func (s *Service) NewLSP() (*lsp.Server, error) {
	return lsp.New(lsp.Config{
		Instrumentation: s.cfg.Child("lsp"),
		GlobalResolver:  s.SymbolResolver(),
		OnExternalChange: observe.Translator[gorp.TxReader[channel.Key, channel.Channel], struct{}]{
			Observable: s.cfg.Channel.NewObservable(),
			Translate: func(
				ctx context.Context,
				r gorp.TxReader[channel.Key, channel.Channel],
			) (struct{}, bool) {
				return struct{}{}, true
			},
		},
	})
}

func (s *Service) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
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
	dataType := statement.AnalyzeFunctionBody(aCtx)
	if !aCtx.Diagnostics.Ok() {
		return telem.UnknownT, aCtx.Diagnostics
	}
	return types.ToTelem(dataType), nil
}

// CompileModule retrieves an Arc program by key and compiles its Module.
// The returned Arc has its Module field populated with the compiled module.
func (s *Service) CompileModule(ctx context.Context, key uuid.UUID) (Arc, error) {
	var prog Arc
	err := s.NewRetrieve().WhereKeys(key).Entry(&prog).Exec(ctx, nil)
	if err != nil {
		return Arc{}, err
	}
	resolverOpt := arc.WithResolver(s.symbolResolver)
	var mod arc.Module
	if prog.Mode == "text" {
		mod, err = arc.CompileText(ctx, prog.Text, resolverOpt)
	} else {
		mod, err = arc.CompileGraph(ctx, prog.Graph, resolverOpt)
	}
	if err != nil {
		return Arc{}, err
	}
	prog.Module = &mod
	return prog, nil
}

// OpenService instantiates a new Arc service using the provided configurations. Each
// configuration will be used as an override for the previous configuration in the list.
// See the ConfigValues struct for information on which fields should be set.
func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, configs...)
	if err != nil {
		return nil, err
	}
	var s = &Service{cfg: cfg}
	s.symbolResolver = symbol.CreateResolver(cfg.Channel)
	cfg.Ontology.RegisterService(s)
	if cfg.Signals != nil {
		s.closer, err = signals.PublishFromGorp(ctx, s.cfg.Signals, signals.GorpPublisherConfigUUID[Arc](cfg.DB))
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// NewWriter opens a new writer for creating, updating, and deleting arcs in Synnax. If
// tx is provided, the writer will use that transaction. If tx is nil, the Writer will
// execute the operations directly on the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:   gorp.OverrideTx(s.cfg.DB, tx),
		otg:  s.cfg.Ontology.NewWriter(tx),
		task: s.cfg.Task.NewWriter(tx),
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
