// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the resource API service.
type ServiceConfig struct {
	// Resource searches and names records of every type.
	//
	// [REQUIRED]
	Resource *resource.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the resource API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Resource = override.Nil(c.Resource, other.Resource)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.resource")
	v.NotNil("resource", c.Resource)
	return v.Error()
}

// Service is the API for searching and naming records of every type.
type Service struct {
	internal *resource.Service
}

// NewService opens a Service with the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Resource}, nil
}

type (
	// SearchRequest selects the resources that Search returns.
	SearchRequest struct {
		// SearchTerm is the text that a resource must match. An empty term lists
		// resources in order of type.
		SearchTerm string `json:"search_term" msgpack:"search_term"`
		// Types limits the search to resources of these types. Empty searches every
		// type.
		Types []resource.Type `json:"types" msgpack:"types"`
		// Limit is the maximum number of resources to return, at most 1000. Zero
		// returns at most 100.
		Limit int `json:"limit" msgpack:"limit"`
		// Offset is the number of closest matches to skip, at most 10000.
		Offset int `json:"offset" msgpack:"offset"`
	}
	// SearchResponse holds the resources that match a SearchRequest, closest match
	// first.
	SearchResponse struct {
		Resources []resource.Resource `json:"resources" msgpack:"resources"`
	}
)

// Search returns the resources that match req and that the caller can retrieve,
// closest match first. It leaves out the rest and still fills each page.
func (s *Service) Search(
	ctx context.Context,
	snap bind.Snapshot,
	req SearchRequest,
) (SearchResponse, error) {
	resources, err := s.internal.Search(ctx, snap, resource.SearchQuery{
		Term:   req.SearchTerm,
		Types:  req.Types,
		Limit:  req.Limit,
		Offset: req.Offset,
		Except: func(id resource.ID) bool { return !snap.Allowed(id) },
	})
	if err != nil {
		return SearchResponse{}, err
	}
	return SearchResponse{Resources: resources}, nil
}

type (
	// RetrieveRequest names the resources that Retrieve returns.
	RetrieveRequest struct {
		IDs []resource.ID `json:"ids" msgpack:"ids"`
	}
	// RetrieveResponse holds the resources that a RetrieveRequest names, in the order
	// of its IDs. It leaves out each ID that names no resource.
	RetrieveResponse struct {
		Resources []resource.Resource `json:"resources" msgpack:"resources"`
	}
)

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named.
func (r RetrieveRequest) Named() []resource.ID { return r.IDs }

// Retrieve returns the resources that req names.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	resources, err := s.internal.Retrieve(ctx, snap, req.IDs...)
	if err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Resources: resources}, nil
}
