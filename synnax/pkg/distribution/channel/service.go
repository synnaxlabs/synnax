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
	Reader
	Writer
	ontology.Service
}

type Writer interface {
	Create(channel *Channel) error
	CreateMany(channels *[]Channel) error
	CreateWithTxn(txn gorp.Txn, channel *Channel) error
	CreateManyWithTxn(txn gorp.Txn, channels *[]Channel) error
}

type Reader interface {
	NewRetrieve() Retrieve
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
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	proxy, err := newLeaseProxy(cfg)
	if err != nil {
		return nil, err
	}
	return &service{clusterDB: cfg.ClusterDB, proxy: proxy}, nil
}

func (s *service) Create(channel *Channel) error { return s.CreateWithTxn(s.clusterDB, channel) }

func (s *service) CreateMany(channels *[]Channel) error {
	return s.CreateManyWithTxn(s.clusterDB, channels)
}

func (s *service) CreateWithTxn(txn gorp.Txn, ch *Channel) error {
	channels := []Channel{*ch}
	err := s.proxy.create(context.TODO(), txn, &channels)
	if err != nil {
		return err
	}
	*ch = channels[0]
	return nil
}

func (s *service) CreateManyWithTxn(txn gorp.Txn, channels *[]Channel) error {
	return s.proxy.create(context.TODO(), txn, channels)
}

func (s *service) NewRetrieve() Retrieve { return NewRetrieve(s.clusterDB) }
