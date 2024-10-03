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
	"io"
	"sync"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for opening the ranger.Service.
type Config struct {
	// DB is the underlying database that the service will use to store Ranges.
	DB *gorp.DB
	// Ontology will be used to create relationships between ranges (parent-child) and
	// with other resources within the Synnax cluster.
	Ontology *ontology.Ontology
	// Group is used to create the top level "Ranges" group that will be the default
	// parent of all ranges.
	Group *group.Service
	// Signals is used to publish signals on channels when ranges are created, updated,
	// deleted, along with changes to aliases and key-value pairs.
	Signals *signals.Provider
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Properties.
func (c Config) Validate() error {
	v := validate.New("ranger")
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

// Service is the main entrypoint for managing ranges within Synnax. It provides
// mechanisms for creating, deleting, and listening to changes in ranges. It also
// provides mechanisms for setting channel aliases for a specific range, and for
// setting meta-data on a range.
type Service struct {
	Config
	group           group.Group
	shutdownSignals io.Closer
	mu              sync.Mutex
}

const groupName = "Ranges"

// OpenService opens a new ranger.Service with the provided configuration. If error
// is nil, the services is ready for use and must be closed by calling Close to
// prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...Config) (s *Service, err error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	g, err := cfg.Group.CreateOrRetrieve(ctx, groupName, ontology.RootID)
	if err != nil {
		return nil, err
	}
	s = &Service{Config: cfg, group: g}
	cfg.Ontology.RegisterService(s)
	cfg.Ontology.RegisterService(&aliasOntologyService{db: cfg.DB})
	if cfg.Signals == nil {
		return
	}
	rangeSignals, err := signals.PublishFromGorp(ctx, cfg.Signals, signals.GorpPublisherConfigUUID[Range](cfg.DB))
	if err != nil {
		return
	}
	aliasSignalsCfg := signals.GorpPublisherConfigString[alias](cfg.DB)
	aliasSignalsCfg.SetName = "sy_range_alias_set"
	aliasSignalsCfg.DeleteName = "sy_range_alias_delete"
	aliasSignals, err := signals.PublishFromGorp(ctx, cfg.Signals, aliasSignalsCfg)
	if err != nil {
		return
	}
	kvSignalsCfg := signals.GorpPublisherConfigString[KVPair](cfg.DB)
	kvSignalsCfg.SetName = "sy_range_kv_set"
	kvSignalsCfg.DeleteName = "sy_range_kv_delete"
	kvSignals, err := signals.PublishFromGorp(ctx, cfg.Signals, kvSignalsCfg)
	if err != nil {
		return
	}
	s.shutdownSignals = xio.MultiCloser{rangeSignals, aliasSignals, kvSignals}
	return
}

// Close closes the service and releases any resources that it may have acquired. Close
// is not safe to call concurrently with any other Service methods (including Writer(s)
// and Retrieve(s)).
func (s *Service) Close() error {
	if s.shutdownSignals != nil {
		return s.shutdownSignals.Close()
	}
	return nil
}

// NewWriter opens a new Writer to create, update, and delete ranges. If tx is not nil,
// the writer will use it to execute all operations. If tx is nil, the writer will execute
// all operations directly against the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:        tx,
		otg:       s.Ontology,
		otgWriter: s.Ontology.NewWriter(tx),
		group:     s.group,
	}
}

// NewRetrieve opens a new Retrieve query to fetch ranges from the database.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{gorp: gorp.NewRetrieve[uuid.UUID, Range](), baseTX: s.DB, otg: s.Ontology}
}
