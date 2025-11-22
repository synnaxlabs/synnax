// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening a symbol service.
type Config struct {
	// DB is the database that the symbol service will store symbols in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between symbols and other entities in
	// the Synnax resource graph.
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create and manage the permanent group for symbols.
	// [OPTIONAL]
	Group *group.Service
	// Signals is used to propagate changes to symbols throughout the cluster.
	// [OPTIONAL]
	Signals *signals.Provider
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening a symbol service.
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
	v := validate.New("Symbol")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	return v.Error()
}

// Service is the primary service for retrieving and modifying symbols from Synnax.
type Service struct {
	Config
	signals io.Closer
	group   group.Group
}

// OpenService instantiates a new symbol service using the provided configurations. Each
// configuration will be used as an override for the previous configuration in the list.
// See the Config struct for information on which fields should be set.
func OpenService(ctx context.Context, configs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg}

	// Create or retrieve the permanent symbols group
	if cfg.Group != nil {
		g, err := cfg.Group.CreateOrRetrieve(ctx, "Schematic Symbols", ontology.RootID)
		if err != nil {
			return nil, err
		}
		s.group = g
	}

	cfg.Ontology.RegisterService(s)
	if cfg.Signals != nil {
		signalsCfg := signals.GorpPublisherConfigUUID[Symbol](cfg.DB)
		signalsCfg.SetName = "sy_schematic_symbol_set"
		signalsCfg.DeleteName = "sy_schematic_symbol_delete"
		s.signals, err = signals.PublishFromGorp(ctx, cfg.Signals, signalsCfg)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// NewWriter opens a new writer for creating, updating, and deleting symbols in Synnax. If
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

// NewRetrieve opens a new query build for retrieving symbols from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[uuid.UUID, Symbol](),
		baseTX: s.DB,
		otg:    s.Ontology,
	}
}

// Group returns the permanent group for schematic symbols.
func (s *Service) Group() group.Group { return s.group }

// Close closes the symbol service, shutting down signal publishers.
func (s *Service) Close() error {
	if s.signals != nil {
		return s.signals.Close()
	}
	return nil
}
