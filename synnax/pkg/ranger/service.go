// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/cdc"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"io"
)

type Config struct {
	DB       *gorp.DB
	Ontology *ontology.Ontology
	Group    *group.Service
	CDC      *cdc.Service
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("ranger")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	validate.NotNil(v, "Group", c.Group)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

type Service struct {
	Config
	group group.Group
	cdc   io.Closer
}

func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	g, err := maybeCreateGroup(ctx, cfg.Group)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg, group: g}
	cfg.Ontology.RegisterService(s)
	return s, err
}

func (s *Service) Close() error { return s.cdc.Close() }

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:    tx,
		otg:   s.Ontology.NewWriter(tx),
		group: s.group,
	}
}

func (s *Service) NewRetrieve() Retrieve {
	return newRetrieve(s.DB, s.Ontology)
}

const groupName = "Ranges"

func maybeCreateGroup(ctx context.Context, svc *group.Service) (g group.Group, err error) {
	err = svc.NewRetrieve().Entry(&g).WhereNames(groupName).Exec(ctx, nil)
	if g.Key != uuid.Nil || err != nil {
		return g, err
	}
	return svc.NewWriter(nil).Create(ctx, groupName, ontology.RootID)
}
