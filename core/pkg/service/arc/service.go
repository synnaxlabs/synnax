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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening a arc service.
type ServiceConfig struct {
	// DB is the database that the arc service will store slates in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between slates and other entities in
	// the Synnax resource graph.
	Ontology *ontology.Ontology
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultConfig is the default configuration for opening a arc service.
	DefaultConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("arc")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	return v.Error()
}

// Service is the primary service for retrieving and modifying slates from Synnax.
type Service struct {
	cfg ServiceConfig
}

func (s Service) Close() error { return nil }

// OpenService instantiates a new arc service using the provided configurations. Each
// configuration will be used as an override for the previous configuration in the list.
// See the Config struct for information on which fields should be set.
func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	cfg.Ontology.RegisterService(s)
	return s, nil
}

// NewWriter opens a new writer for creating, updating, and deleting slates in Synnax. If
// tx is provided, the writer will use that transaction. If tx is nil, the Writer
// will execute the operations directly on the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:        gorp.OverrideTx(s.cfg.DB, tx),
		otgWriter: s.cfg.Ontology.NewWriter(tx),
		otg:       s.cfg.Ontology,
	}
}

// NewRetrieve opens a new query builder for retrieving slates from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: gorp.NewRetrieve[uuid.UUID, arc](), baseTX: s.cfg.DB}
}
