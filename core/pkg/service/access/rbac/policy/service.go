// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	DB      *gorp.DB
	Signals *signals.Provider
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements [config.Config].
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
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
	cfg     Config
	signals io.Closer
}

func OpenService(ctx context.Context, configs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	if cfg.Signals != nil {
		if s.signals, err = signals.PublishFromGorp[uuid.UUID, Policy](
			ctx,
			cfg.Signals,
			signals.GorpPublisherConfigUUID[Policy](cfg.DB),
		); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Service) Close() error {
	if s.signals == nil {
		return nil
	}
	return s.signals.Close()
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: gorp.OverrideTx(s.cfg.DB, tx)}
}

func (s *Service) NewRetrieve() Retriever {
	return Retriever{
		baseTx: s.cfg.DB,
		gorp:   gorp.NewRetrieve[uuid.UUID, Policy](),
	}
}

func (s *Service) NewRoleRetriever() RoleRetriever {
	return RoleRetriever{
		baseTx: s.cfg.DB,
		gorp:   gorp.NewRetrieve[uuid.UUID, role.Role](),
	}
}
