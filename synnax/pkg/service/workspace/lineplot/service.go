// Copyright 2023 Synnax Labs, Inc.
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

type Config struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Properties.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	return c
}

// Validate implements config.Properties.
func (c Config) Validate() error {
	v := validate.New("lineplot")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	return v.Error()
}

type Service struct{ Config }

func NewService(configs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg}
	cfg.Ontology.RegisterService(s)
	return s, nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	tx = gorp.OverrideTx(s.DB, tx)
	return Writer{
		tx:  tx,
		otg: s.Ontology.NewWriter(tx),
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[uuid.UUID, LinePlot](),
		baseTX: s.DB,
	}
}
