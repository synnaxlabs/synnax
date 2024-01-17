/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package module

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/device/rack"
	"github.com/synnaxlabs/synnax/pkg/distribution/cdc"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"io"
)

// Config is the configuration for creating a Service.
type Config struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Group    *group.Service
	Rack     *rack.Service
	CDC      *cdc.Provider
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
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.CDC = override.Nil(c.CDC, other.CDC)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("workspace")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "rack", c.Rack)
	return v.Error()
}

type Service struct {
	Config
	cdc io.Closer
}

func OpenService(ctx context.Context, configs ...Config) (s *Service, err error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return
	}
	s = &Service{Config: cfg}
	cfg.Ontology.RegisterService(s)

	if cfg.CDC == nil {
		return s, nil
	}

	cdcS, err := cdc.SubscribeToGorp(ctx, cfg.CDC, cdc.GorpConfigPureNumeric[Key, Module](cfg.DB, telem.Uint64T))
	if err != nil {
		return
	}
	s.cdc = cdcS
	return
}

func (s *Service) Close() error {
	if s.cdc != nil {
		return s.cdc.Close()
	}
	return nil
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:   gorp.OverrideTx(s.DB, tx),
		otg:  s.Ontology.NewWriter(tx),
		rack: s.Rack.NewWriter(tx),
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		otg:    s.Ontology,
		baseTX: s.DB,
		gorp:   gorp.NewRetrieve[Key, Module](),
	}
}
