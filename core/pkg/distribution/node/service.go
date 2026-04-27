// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig configures a node Service.
type ServiceConfig struct {
	alamos.Instrumentation
	// Cluster provides cluster-membership information about the host node and its
	// peers.
	//
	// [REQUIRED]
	Cluster Cluster
	// Ontology is the ontology to which node resources are published.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// Search is the search index to which node resources are registered.
	//
	// [REQUIRED]
	Search *search.Index
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Cluster = override.Nil(c.Cluster, other.Cluster)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Search = override.Nil(c.Search, other.Search)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("node")
	validate.NotNil(v, "cluster", c.Cluster)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "search", c.Search)
	return v.Error()
}

// Service publishes the cluster's nodes as resources in the Synnax ontology and search
// index.
type Service struct{ cfg ServiceConfig }

// NewService creates a Service, registers it with the ontology and search index, and
// defines the sentinel free-node resource (KeyFree) that free channels parent under.
func NewService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{cfg: cfg}
	cfg.Ontology.RegisterService(s)
	cfg.Search.RegisterService(s)
	if err := cfg.Ontology.NewWriter(nil).DefineResource(ctx, OntologyID(KeyFree)); err != nil {
		return nil, errors.Wrap(err, "define free node ontology resource: ")
	}
	return s, nil
}
