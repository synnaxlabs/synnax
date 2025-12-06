// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rbac

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	DB *gorp.DB
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements [config.Config].
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
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
	entryManager *gorp.EntryManager[uuid.UUID, Policy]
}

func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	entryManager, err := gorp.OpenEntryManager[uuid.UUID, Policy](ctx, cfg.DB)
	if err != nil {
		return nil, err
	}
	return &Service{Config: cfg, entryManager: entryManager}, nil
}

func (s *Service) Close() error {
	return s.entryManager.Close()
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: gorp.OverrideTx(s.DB, tx)}
}

func (s *Service) NewRetrieve() Retriever {
	return Retriever{
		baseTx: s.DB,
		gorp:   gorp.NewRetrieve[uuid.UUID, Policy](),
	}
}
