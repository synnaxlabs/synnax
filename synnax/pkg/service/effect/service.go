// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package effect

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/annotation"
	"github.com/synnaxlabs/synnax/pkg/service/slate"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for opening the effect service.
type ServiceConfig struct {
	alamos.Instrumentation
	// DB is the database that the effect service will store effects in.
	// [REQUIRED]
	DB *gorp.DB
	// Ontology is used to define relationships between effects and other entities in
	// the Synnax resource graph.
	// [REQUIRED]
	Ontology   *ontology.Ontology
	Framer     *framer.Service
	Slate      *slate.Service
	Channel    channel.Service
	Annotation *annotation.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultConfig is the default configuration for opening a effect service.
	DefaultConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.DB = override.Nil(c.DB, other.DB)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Slate = override.Nil(c.Slate, other.Slate)
	c.Annotation = override.Nil(c.Annotation, other.Annotation)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("effect")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "slate", c.Slate)
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "annotation", c.Annotation)
	return v.Error()
}

// Service is the primary service for retrieving and modifying effects from Synnax.
type Service struct {
	cfg ServiceConfig
	mu  struct {
		sync.Mutex
		entries map[uuid.UUID]*entry
	}
	effectStateChannelKey channel.Key
	effectStateWriter     *framer.Writer
}

func (s *Service) Close() error { return nil }

// OpenService instantiates a new effect service using the provided configurations. Each
// configuration will be used as an override for the previous configuration in the list.
// See the ServiceConfig struct for information on which fields should be set.
func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	s.mu.entries = make(map[uuid.UUID]*entry)

	cfg.Ontology.RegisterService(ctx, s)

	obs := gorp.Observe[uuid.UUID, Effect](cfg.DB)
	obs.OnChange(s.handleChange)

	effectStateCh := channel.Channel{
		Name:     "sy_effect_state",
		DataType: telem.JSONT,
		Virtual:  true,
		Internal: true,
	}
	if err = cfg.Channel.Create(
		ctx,
		&effectStateCh,
		channel.OverwriteIfNameExistsAndDifferentProperties(),
		channel.RetrieveIfNameExists(true),
	); err != nil {
		return nil, err
	}
	s.effectStateChannelKey = effectStateCh.Key()
	if s.effectStateWriter, err = cfg.Framer.OpenWriter(ctx, framer.WriterConfig{
		Keys:  []channel.Key{s.effectStateChannelKey},
		Start: telem.Now(),
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// NewWriter opens a new writer for creating, updating, and deleting effects in Synnax. If
// tx is provided, the writer will use that transaction. If tx is nil, the Writer
// will execute the operations directly on the underlying gorp.DB.
func (s *Service) NewWriter(tx gorp.Tx) Writer {
	return Writer{
		tx:        gorp.OverrideTx(s.cfg.DB, tx),
		otgWriter: s.cfg.Ontology.NewWriter(tx),
		otg:       s.cfg.Ontology,
	}
}

// NewRetrieve opens a new query builder for retrieving effects from Synnax.
func (s *Service) NewRetrieve() Retrieve {
	return Retrieve{
		gorp:   gorp.NewRetrieve[uuid.UUID, Effect](),
		baseTX: s.cfg.DB,
	}
}
