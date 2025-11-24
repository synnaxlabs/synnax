// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening a line plot service.
type Config struct {
	// DB is the database that the line plot service will store line plots in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between line plots and other entities in
	// the Synnax resource graph.
	// [REQUIRED]
	Ontology *ontology.Ontology
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for opening a line plot service.
	DefaultConfig = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("lineplot")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	return v.Error()
}

// Service is the primary service for retrieving and modifying line plots from Synnax.
type Service struct{ Config }

// OpenService instantiates a new line plot service using the provided configurations.
// Each configuration will be used as an override for the previous configuration in the
// list. See the Config struct for information on which fields should be set.
func OpenService(cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg}
	cfg.Ontology.RegisterService(s)
	return s, nil
}

// NewWriter opens a new writer for creating, updating, and deleting line plots in Synnax. If
// tx is nil, the writer will perform operations directly against the underlying gorp.DB provided
// to the line plot service. If tx is provided, the writer will use that transaction.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.DB, tx)
	return Writer{
		tx:  tx,
		otg: s.Ontology.NewWriter(tx),
	}
}

// NewRetrieve opens a new query builder for retrieving line plots from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[uuid.UUID, LinePlot](),
		baseTX: s.DB,
	}
}
