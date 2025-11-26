// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic/symbol"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening a schematic service.
type Config struct {
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
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening a schematic service.
	DefaultConfig = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("Schematic")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	return v.Error()
}

// Service is the primary service for retrieving and modifying schematics from Synnax.
type Service struct {
	Config
	Symbol *symbol.Service
}

// OpenService instantiates a new schematic service using the provided configurations.
// Each configuration will be used as an override for the previous configuration in the
// list. See the Config struct for information on which fields should be set.
func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg}
	cfg.Ontology.RegisterService(s)

	if s.Symbol, err = symbol.OpenService(ctx, symbol.Config{
		DB:       cfg.DB,
		Ontology: cfg.Ontology,
		Group:    cfg.Group,
		Signals:  cfg.Signals,
	}); err != nil {
		return nil, err
	}

	return s, nil
}

// Close closes the schematic service and releases any resources that it may have
// acquired.
func (s *Service) Close() error { return s.Symbol.Close() }

// NewWriter opens a new writer for creating, updating, and deleting logs in Synnax. If
// tx is provided, the writer will use that transaction. If tx is nil, the Writer
// will execute the operations directly on the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.DB, tx)
	return Writer{
		tx:        tx,
		otgWriter: s.Ontology.NewWriter(tx),
		otg:       s.Ontology,
	}
}

// NewRetrieve opens a new query build for retrieving logs from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[uuid.UUID, Schematic](),
		baseTX: s.DB,
	}
}
