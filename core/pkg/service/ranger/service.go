// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the ranger.Service.
type ServiceConfig struct {
	// Instrumentation for logging, tracing, and metrics.
	alamos.Instrumentation
	// DB is the underlying database that the service will use to store Ranges.
	DB *gorp.DB
	// Ontology will be used to create relationships between ranges (parent-child) and
	// with other resources within the Synnax cluster.
	Ontology *ontology.Ontology
	// Group is used to create the top level "Ranges" group that will be the default
	// parent of all ranges.
	Group *group.Service
	// Signals is used to publish signals on channels when ranges are created, updated,
	// deleted, along with changes to aliases and key-value pairs.
	Signals *signals.Provider
	// Label is the label service used to attach, remove, and query labels related to
	// changes.
	Label *label.Service
	// ForceMigration will force all migrations to run, regardless of whether they have
	// already been run.
	ForceMigration *bool
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultConfig is the default configuration for opening a range service.
	DefaultConfig = ServiceConfig{ForceMigration: config.False()}
)

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("service.ranger")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "label", c.Label)
	validate.NotNil(v, "force_migration", c.ForceMigration)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Label = override.Nil(c.Label, other.Label)
	c.ForceMigration = override.Nil(c.ForceMigration, other.ForceMigration)
	return c
}

// Service is the main entrypoint for managing ranges within Synnax. It provides
// mechanisms for creating, deleting, and listening to changes in ranges. It also
// provides mechanisms for setting channel aliases for a specific range, and for setting
// metadata on a range.
type Service struct {
	cfg             ServiceConfig
	shutdownSignals io.Closer
}

// OpenService opens a new ranger.Service with the provided configuration. If error is
// nil, the services is ready for use and must be closed by calling Close to prevent
// resource leaks.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	cfg.Ontology.RegisterService(s)
	cfg.Ontology.RegisterService(&aliasOntologyService{db: cfg.DB})
	if err := s.migrate(ctx); err != nil {
		return nil, err
	}
	if cfg.Signals == nil {
		return s, nil
	}
	rangeSignals, err := signals.PublishFromGorp(
		ctx,
		cfg.Signals,
		signals.GorpPublisherConfigUUID[Range](cfg.DB),
	)
	if err != nil {
		return nil, err
	}
	aliasSignalsCfg := signals.GorpPublisherConfigString[Alias](cfg.DB)
	aliasSignalsCfg.SetName = "sy_range_alias_set"
	aliasSignalsCfg.DeleteName = "sy_range_alias_delete"
	aliasSignals, err := signals.PublishFromGorp(ctx, cfg.Signals, aliasSignalsCfg)
	if err != nil {
		return nil, err
	}
	kvSignalsCfg := signals.GorpPublisherConfigString[KVPair](cfg.DB)
	kvSignalsCfg.SetName = "sy_range_kv_set"
	kvSignalsCfg.DeleteName = "sy_range_kv_delete"
	kvSignals, err := signals.PublishFromGorp(ctx, cfg.Signals, kvSignalsCfg)
	if err != nil {
		return nil, err
	}
	s.shutdownSignals = xio.MultiCloser{rangeSignals, aliasSignals, kvSignals}
	return s, nil
}

// Close closes the service and releases any resources that it may have acquired. Close
// is not safe to call concurrently with any other Service methods (including Writer(s)
// and Retrieve(s)).
func (s *Service) Close() error {
	if s.shutdownSignals != nil {
		return s.shutdownSignals.Close()
	}
	return nil
}

// NewWriter opens a new Writer to create, update, and delete ranges. If tx is not nil,
// the writer will use it to execute all operations. If tx is nil, the writer will
// execute all operations directly against the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:        gorp.OverrideTx(s.cfg.DB, tx),
		otg:       s.cfg.Ontology,
		otgWriter: s.cfg.Ontology.NewWriter(tx),
	}
}

// NewRetrieve opens a new Retrieve query to fetch ranges from the database.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[uuid.UUID, Range](),
		baseTX: s.cfg.DB,
		otg:    s.cfg.Ontology,
		label:  s.cfg.Label,
	}
}
