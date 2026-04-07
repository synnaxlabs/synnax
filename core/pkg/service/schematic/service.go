// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"
	"encoding/json"
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	xchange "github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a schematic service.
type ServiceConfig struct {
	// Instrumentation for logging, tracing, and metrics.
	alamos.Instrumentation
	// DB is the database that the schematic service will store schematics in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between schematics and other entities in
	// the Synnax resource graph.
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create and manage groups for symbols.
	// [OPTIONAL]
	Group *group.Service
	// Signals is used to propagate changes to schematics and symbols throughout the cluster.
	// [OPTIONAL]
	Signals *signals.Provider
	// Search is the search index for fuzzy searching schematics.
	// [REQUIRED]
	Search *search.Index
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for opening a schematic service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	c.Search = override.Nil(c.Search, other.Search)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("schematic")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

// Service is the primary service for retrieving and modifying schematics from Synnax.
type Service struct {
	ServiceConfig
	Symbol         *symbol.Service
	closer         io.MultiCloser
	table          *gorp.Table[uuid.UUID, Schematic]
	actionObserver observe.Observer[ScopedAction]
}

// OpenService instantiates a new schematic service using the provided configurations.
// Each configuration will be used as an override for the previous configuration in the
// list. See the Config struct for information on which fields should be set.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (s *Service, err error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s = &Service{ServiceConfig: cfg, actionObserver: observe.New[ScopedAction]}
	cleanup, ok := service.NewOpener(ctx, &s.closer)
	defer func() { err = cleanup(err) }()
	if s.table, err = gorp.OpenTable[uuid.UUID, Schematic](ctx, gorp.TableConfig[Schematic]{
		DB:              cfg.DB,
		Migrations:      []migrate.Migration{gorp.CodecMigration[uuid.UUID, Schematic]("msgpack_to_orc")},
		Instrumentation: cfg.Instrumentation,
	}); !ok(err, s.table) {
		return nil, err
	}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	if s.Symbol, err = symbol.OpenService(ctx, symbol.ServiceConfig{
		Instrumentation: cfg.Child("symbol"),
		DB:              cfg.DB,
		Ontology:        cfg.Ontology,
		Group:           cfg.Group,
		Signals:         cfg.Signals,
		Search:          cfg.Search,
	}); !ok(err, s.Symbol) {
		return nil, err
	}

	if cfg.Signals != nil {
		translated := observe.Translator[ScopedAction, []xchange.Change[[]byte, struct{}]]{
			Observable: s.actionObserver,
			Translate: func(_ context.Context, sa ScopedAction) ([]xchange.Change[[]byte, struct{}], bool) {
				b, err := json.Marshal(sa)
				if err != nil {
					return nil, false
				}
				return []xchange.Change[[]byte, struct{}]{
					{Variant: xchange.VariantSet, Key: append(b, '\n')},
				}, true
			},
		}
		var sig io.Closer
		if sig, err = cfg.Signals.PublishFromObservable(ctx, signals.ObservablePublisherConfig{
			Name:          "schematic_actions",
			Observable:    translated,
			SetChannel:    channel.Channel{Name: "sy_schematic_set", DataType: telem.JSONT, Internal: true},
			DeleteChannel: channel.Channel{Name: "sy_schematic_delete", DataType: telem.UUIDT, Internal: true},
		}); !ok(err, sig) {
			return nil, err
		}
	}

	return s, nil
}

// Close closes the schematic service and releases any resources that it may have
// acquired.
func (s *Service) Close() error { return s.closer.Close() }

// NewWriter opens a new writer for creating, updating, and deleting logs in Synnax. If
// tx is provided, the writer will use that transaction. If tx is nil, the Writer
// will execute the operations directly on the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.DB, tx)
	return Writer{
		tx:             tx,
		otgWriter:      s.Ontology.NewWriter(tx),
		otg:            s.Ontology,
		table:          s.table,
		actionObserver: s.actionObserver,
	}
}

// NewRetrieve opens a new query build for retrieving logs from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   s.table.NewRetrieve(),
		baseTX: s.DB,
	}
}
