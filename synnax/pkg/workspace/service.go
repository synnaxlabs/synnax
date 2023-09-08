// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package workspace

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/workspace/pid"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for creating a Service.
type Config struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Group    *group.Service
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("workspace")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	return v.Error()
}

type Service struct {
	Config
	PID *pid.Service
}

func NewService(configs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	pidSvc, err := pid.NewService(pid.Config{DB: cfg.DB})
	return &Service{Config: cfg, PID: pidSvc}, nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: gorp.OverrideTx(s.DB, tx), otg: s.Ontology.NewWriter(tx)}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: gorp.NewRetrieve[uuid.UUID, Workspace]()}
}
