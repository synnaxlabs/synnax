// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for creating a Service.
type ServiceConfig struct {
	alamos.Instrumentation
	Signals  *signals.Provider
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Search   *search.Index
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Search = override.Nil(c.Search, other.Search)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("panel")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

type Service struct {
	cfg    ServiceConfig
	closer xio.MultiCloser
	table  *gorp.Table[Key, Panel]
	state  *actions.State[Key, Action]
}

func OpenService(
	ctx context.Context,
	configs ...ServiceConfig,
) (s *Service, err error) {
	cfg, err := config.New(ServiceConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg, state: actions.NewState[Key, Action]()}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Panel]{
		DB:              cfg.DB,
		Migrations:      versions.Migrations,
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	if cfg.Signals == nil {
		return s, nil
	}
	// Broadcast action vectors on sy_panel_set; the gorp delete publisher's Set
	// is disabled because action frames carry the mutation payload, but it still
	// emits deletes so clients prune cached panels.
	var sig io.Closer
	if sig, err = actions.PublishSignals(ctx, actions.SignalsConfig[Key, Action]{
		Provider: cfg.Signals,
		State:    s.state,
		Name:     "panel",
	}); !ok(err, sig) {
		return nil, err
	}
	deleteCfg := signals.GorpPublisherConfigUUID(s.table.Observe())
	deleteCfg.DisableSet = true
	if sig, err = signals.PublishFromGorp(ctx, cfg.Signals, deleteCfg); !ok(err, sig) {
		return nil, err
	}
	return s, nil
}

func (s *Service) Close() error { return s.closer.Close() }

// OnAction subscribes the given handler to the action stream emitted by
// Writer.Dispatch. The handler runs synchronously inside Dispatch after the underlying
// transaction commits. The returned Disconnect removes the handler.
func (s *Service) OnAction(
	handler func(context.Context, actions.Scoped[Key, Action]),
) observe.Disconnect {
	return s.state.OnAction(handler)
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:         gorp.OverrideTx(s.cfg.DB, tx),
		otg:        s.cfg.Ontology.NewWriter(tx),
		table:      s.table,
		dispatcher: s.state.Dispatcher(),
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		search: s.cfg.Search,
		baseTX: s.cfg.DB,
		gorp:   s.table.NewRetrieve(),
	}
}
