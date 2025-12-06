// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/signals"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for the label service. Config is provided to the
// OpenService method.
type Config struct {
	// DB specifies the database that the label service will use to store and retrieve
	// labels.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is the ontology service that the label service will use to manage
	// resources and relationships between other objects.
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Group is used to create and manage a root group for holding all labels.
	// [REQUIRED]
	Group *group.Service
	// Signals is the signal service used to propagate changes to labels.
	// [OPTIONAL]
	Signals *signals.Provider
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default for the label service. This configuration is not
	// valid, and must be overridden with a valid configuration before the service can
	// be opened.
	DefaultConfig = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("label")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "group", c.Group)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Group = override.Nil(c.Group, other.Group)
	c.Signals = override.Nil(c.Signals, other.Signals)
	return c
}

// Service is the main entry point for managing labels within Synnax. It provides
// mechanisms for creating, deleting, retrieving, and listening to changes on labels.
type Service struct {
	Config
	signals      io.Closer
	entryManager *gorp.EntryManager[uuid.UUID, Label]
}

// OpenService opens a new label service using the provided configuration. If error
// is nil, the service is ready for use and must be closed by calling Close in order
// to prevent resource leaks.
func OpenService(ctx context.Context, cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	entryManager, err := gorp.OpenEntryManager[uuid.UUID, Label](ctx, cfg.DB)
	if err != nil {
		return nil, err
	}
	s := &Service{Config: cfg, entryManager: entryManager}
	cfg.Ontology.RegisterService(s)
	if cfg.Signals != nil {
		s.signals, err = signals.PublishFromGorp(ctx, cfg.Signals, signals.GorpPublisherConfigUUID[Label](cfg.DB))
		if err != nil {
			return nil, err
		}
	}
	return s, err
}

// Close closes the label service and releases any resources that it may have acquired.
// Close must be called when the service is no longer needed to prevent resource leaks.
func (s *Service) Close() error {
	if s.signals != nil {
		return errors.Combine(s.signals.Close(), s.entryManager.Close())
	}
	return s.entryManager.Close()
}

// NewRetrieve opens a new Retrieve query to fetch labels.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		baseTx: s.DB,
		gorp:   gorp.NewRetrieve[uuid.UUID, Label](),
		otg:    s.Ontology,
	}
}

// NewWriter opens a new Writer to create, update, and delete labels. If tx is not nil
// the writer will use it, otherwise it will execute operations directly against the
// underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{tx: tx, otg: s.Ontology.NewWriter(tx)}
}
