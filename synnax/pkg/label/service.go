/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package label

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
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
	Signals  *signals.Provider
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Properties.
func (c Config) Validate() error {
	v := validate.New("label")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Ontology", c.Ontology)
	validate.NotNil(v, "Group", c.Group)
	return v.Error()
}

// Override implements config.Properties.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

type Service struct {
	Config
	signals io.Closer
	group   group.Group
}

//const groupName = "Labels"

func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	//g, err := cfg.Group.CreateOrRetrieve(ctx, groupName, ontology.RootID)
	//if err != nil {
	//	return nil, err
	//}
	s := &Service{Config: cfg}
	cfg.Ontology.RegisterService(s)
	if cfg.Signals != nil {
		s.signals, err = signals.PublishFromGorp(ctx, cfg.Signals, signals.GorpPublisherConfigUUID[Label](cfg.DB))
		if err != nil {
			return nil, err
		}
	}
	return s, err
}

func (s *Service) NewRetrieve() Retrieve { return NewRetrieve(s.DB, s.Ontology) }

func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: tx, otg: s.Ontology.NewWriter(tx)}
}
