// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// UserRoleGetter is an interface for getting user roles. This allows the RBAC
// service to check role-based policies without depending on the full user service.
type UserRoleGetter interface {
	GetUserRoles(ctx context.Context, userKey uuid.UUID) ([]uuid.UUID, error)
}

type Config struct {
	DB             *gorp.DB
	Ontology       *ontology.Ontology
	Signals        *signals.Provider
	UserRoleGetter UserRoleGetter
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements [config.Config].
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Validate implements [config.Config].
func (c Config) Validate() error {
	v := validate.New("policy")
	validate.NotNil(v, "db", c.DB)
	return v.Error()
}

type Service struct {
	Config
	signals io.Closer
}

func (s *Service) Close() error {
	if s.signals != nil {
		return s.signals.Close()
	}
	return nil
}

func OpenService(ctx context.Context, configs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg}
	if cfg.Ontology != nil {
		cfg.Ontology.RegisterService(s)
	}
	if cfg.Signals != nil {
		if s.signals, err = signals.PublishFromGorp[uuid.UUID, Role](
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigUUID[Role](cfg.DB),
		); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:  gorp.OverrideTx(s.DB, tx),
		otg: s.Ontology.NewWriter(tx),
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{baseTx: s.DB, gorp: gorp.NewRetrieve[uuid.UUID, Role]()}
}
