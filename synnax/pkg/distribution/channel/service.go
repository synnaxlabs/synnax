// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// service is central entity for managing channels within delta's distribution layer. It provides facilities for creating
// and retrieving channels.
type service struct {
	*gorp.DB
	proxy *leaseProxy
	otg   *ontology.Ontology
	group group.Group
}

type Service interface {
	Readable
	Writeable
	ontology.Service
}

type Writeable interface {
	NewWriter(tx gorp.Tx) Writer
}

type Readable interface {
	NewRetrieve() Retrieve
}

type ServiceConfig struct {
	HostResolver core.HostResolver
	ClusterDB    *gorp.DB
	TSChannel    *ts.DB
	Transport    Transport
	Ontology     *ontology.Ontology
	Group        *group.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.channel")
	validate.NotNil(v, "HostResolver", c.HostResolver)
	validate.NotNil(v, "ClusterDB", c.ClusterDB)
	validate.NotNil(v, "TSChannel", c.TSChannel)
	validate.NotNil(v, "Transport", c.Transport)
	return v.Error()
}

func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.ClusterDB = override.Nil(c.ClusterDB, other.ClusterDB)
	c.TSChannel = override.Nil(c.TSChannel, other.TSChannel)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

var DefaultConfig = ServiceConfig{}

func New(ctx context.Context, configs ...ServiceConfig) (Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	g, err := maybeCreateGroup(ctx, cfg.Group)
	if err != nil {
		return nil, err
	}
	proxy, err := newLeaseProxy(cfg, g)
	if err != nil {
		return nil, err
	}
	return &service{
		DB:    cfg.ClusterDB,
		proxy: proxy,
		otg:   cfg.Ontology,
	}, nil
}

func (s *service) NewWriter(tx gorp.Tx) Writer {
	return Writer{proxy: s.proxy, tx: s.DB.OverrideTx(tx)}
}

func (s *service) NewRetrieve() Retrieve { return newRetrieve(s.DB, s.otg) }

const groupName = "Channels"

func maybeCreateGroup(ctx context.Context, svc *group.Service) (g group.Group, err error) {
	if svc == nil {
		return g, nil
	}
	err = svc.NewRetrieve().Entry(&g).WhereNames(groupName).Exec(ctx, nil)
	if g.Key != uuid.Nil || err != nil {
		return g, err
	}
	return svc.NewWriter(nil).Create(ctx, groupName, ontology.RootID)
}
