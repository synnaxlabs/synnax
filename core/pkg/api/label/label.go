// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the label API service.
type ServiceConfig struct {
	// Label stores and retrieves labels.
	//
	// [REQUIRED]
	Label *label.Service
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the label API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Label = override.Nil(c.Label, other.Label)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.label")
	v.NotNil("label", c.Label)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *label.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Label,
		access:   cfg.RBAC,
	}, nil
}

type Label = label.Label

// CreateRequest is a request to create a Label in the cluster.
type CreateRequest struct {
	// Labels are the labels to create.
	Labels []Label `json:"labels" msgpack:"labels"`
}

// CreateResponse is a response to a CreateRequest.
type CreateResponse struct {
	// Labels are the labels that were created.
	Labels []Label `json:"labels" msgpack:"labels"`
}

// Create creates the labels in the cluster.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{{Type: ontology.ResourceTypeLabel}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Labels); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse(req), nil
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.internal.NewRetrieve().Entries(&res.Labels))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	objects := label.OntologyIDsFromLabels(res.Labels)
	// The labels of a resource describe it, so a for line needs access to the resource.
	if req.Where != nil {
		objects = append(objects, req.Where.ForIDs()...)
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: objects,
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []label.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: label.OntologyIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

type AddRequest struct {
	ID      ontology.ID `json:"id"      msgpack:"id"`
	Labels  []label.Key `json:"labels"  msgpack:"labels"`
	Replace bool        `json:"replace" msgpack:"replace"`
}

func (s *Service) Add(
	ctx context.Context,
	tx gorp.Tx,
	req AddRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(label.OntologyIDs(req.Labels), req.ID),
	}); err != nil {
		return struct{}{}, err
	}
	w := s.internal.NewWriter(tx)
	if req.Replace {
		return struct{}{}, w.Replace(ctx, req.ID, req.Labels)
	}
	return struct{}{}, w.Label(ctx, req.ID, req.Labels)
}

type RemoveRequest struct {
	ID     ontology.ID `json:"id"     msgpack:"id"`
	Labels []label.Key `json:"labels" msgpack:"labels"`
}

func (s *Service) Remove(
	ctx context.Context,
	tx gorp.Tx,
	req RemoveRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(label.OntologyIDs(req.Labels), req.ID),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).RemoveLabel(ctx, req.ID, req.Labels)
}
