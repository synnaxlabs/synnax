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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
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
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the resource API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Resource = override.Nil(c.Resource, other.Resource)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.resource")
	v.NotNil("resource", c.Resource)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

// Service is the API for searching and naming records of every type.
type Service struct {
	internal *resource.Service
	access   *rbac.Service
}

// NewService opens a Service with the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Resource, access: cfg.RBAC}, nil
}

type (
	// SearchRequest selects the resources that Search returns.
	SearchRequest struct {
		// Term is the text that a resource must match. An empty term lists resources in
		// order of type.
		Term string `json:"term" msgpack:"term"`
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
	req SearchRequest,
) (SearchResponse, error) {
	allowed, err := s.access.NewEnforcer(nil).Allowed(
		ctx,
		auth.GetSubject(ctx),
		access.ActionRetrieve,
	)
	if err != nil {
		return SearchResponse{}, err
	}
	resources, err := s.internal.Search(ctx, nil, resource.SearchQuery{
		Term:   req.Term,
		Types:  req.Types,
		Limit:  req.Limit,
		Offset: req.Offset,
		Except: func(id resource.ID) bool { return !allowed(id) },
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

// Retrieve returns the resources that req names. It returns access.ErrDenied when the
// caller cannot retrieve one of them.
func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	resources, err := s.internal.Retrieve(ctx, nil, req.IDs...)
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: lo.Map(resources, func(r resource.Resource, _ int) resource.ID {
			return r.ID
		}),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Resources: resources}, nil
}
