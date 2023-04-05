// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import "C"
import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// service is central entity for managing channels within delta's distribution layer. It provides facilities for creating
// and retrieving channels.
type service struct {
	clusterDB *gorp.DB
	proxy     *leaseProxy
}

type Service interface {
	Readable
	Writeable
	ontology.Service
}

type Writeable interface {
	NewWriter(writer gorp.WriteContext) Writer
}

type Readable interface {
	NewRetrieve(ctx context.Context) Retrieve
}

type ServiceConfig struct {
	HostResolver core.HostResolver
	ClusterDB    *gorp.DB
	TSChannel    storage.TSChannelManager
	Transport    Transport
	Ontology     *ontology.Ontology
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
	return c
}

var DefaultConfig = ServiceConfig{}

func New(configs ...ServiceConfig) (Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	proxy, err := newLeaseProxy(cfg)
	if err != nil {
		return nil, err
	}
	return &service{clusterDB: cfg.ClusterDB, proxy: proxy}, nil
}

func (s *service) NewWriter(writer gorp.WriteContext) Writer {
	return Writer{proxy: s.proxy, writer: writer}
}

func (s *service) NewRetrieve(ctx context.Context) Retrieve {
	return NewRetrieve(s.clusterDB.BeginRead(ctx))
}
