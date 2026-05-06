// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/migrations/v0"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/validate"
)

type ServiceConfig struct {
	alamos.Instrumentation
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Search   *search.Index
	Signals  *signals.Provider
}

var (
	_                    config.Config[ServiceConfig] = ServiceConfig{}
	DefaultServiceConfig                              = ServiceConfig{}
)

// Override implements [config.Config].
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Search = override.Nil(c.Search, other.Search)
	return c
}

// Validate implements [config.Config].
func (c ServiceConfig) Validate() error {
	v := validate.New("policy")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

type Service struct {
	cfg    ServiceConfig
	closer xio.MultiCloser
	table  *gorp.Table[uuid.UUID, Policy]
}

func OpenService(ctx context.Context, configs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(DefaultServiceConfig, configs...)
	if err != nil {
		return nil, err
	}
	s = &Service{cfg: cfg}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	v0Mig := v0.Migration()
	if s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Policy]{
		DB:              cfg.DB,
		Instrumentation: cfg.Instrumentation,
		Migrations: []migrate.Migration{
			v0Mig,
			gorp.CodecMigration[uuid.UUID, Policy]("msgpack_to_orc", v0Mig.Key()),
		},
	}); err != nil {
		return nil, err
	}
	if cfg.Signals != nil {
		var sig io.Closer
		if sig, err = signals.PublishFromGorp(
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigUUID[Policy](s.table.Observe()),
		); !ok(err, sig) {
			return nil, err
		}
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	return s, nil
}

func (s *Service) Close() error { return s.closer.Close() }

func (s *Service) NewWriter(tx gorp.Tx, allowInternal bool) Writer {
	tx = gorp.OverrideTx(s.cfg.DB, tx)
	return Writer{
		tx:            tx,
		otg:           s.cfg.Ontology.NewWriter(tx),
		allowInternal: allowInternal,
		table:         s.table,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		baseTX:   s.cfg.DB,
		gorp:     s.table.NewRetrieve(),
		ontology: s.cfg.Ontology,
	}
}
