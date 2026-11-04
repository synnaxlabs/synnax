// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the group API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Group stores and retrieves groups.
	//
	// [REQUIRED]
	Group *group.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the group API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.group")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("group", c.Group)
	return v.Error()
}

type Service struct {
	access   *rbac.Service
	internal *group.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access:   cfg.RBAC,
		internal: cfg.Group,
	}, nil
}

type (
	CreateRequest struct {
		Parent *resource.ID  `json:"parent" msgpack:"parent"`
		Name   string        `json:"name"   msgpack:"name"`
		Scope  resource.Type `json:"scope"  msgpack:"scope"`
		Key    group.Key     `json:"key"    msgpack:"key"`
	}
	CreateResponse struct {
		Group group.Group `json:"group" msgpack:"group"`
	}
)

func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []resource.ID{{Type: resource.TypeGroup}},
	}); err != nil {
		return CreateResponse{}, err
	}
	g := group.Group{Key: req.Key, Name: req.Name, Scope: req.Scope, Parent: req.Parent}
	if err := s.internal.NewWriter(tx).Create(ctx, &g); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Group: g}, nil
}

type (
	RetrieveRequest struct {
		// Nested keeps only groups with a parent when true, and only groups with no
		// parent when false. Nil applies no filter.
		Nested     *bool           `json:"nested"      msgpack:"nested"`
		SearchTerm string          `json:"search_term" msgpack:"search_term"`
		Keys       []group.Key     `json:"keys"        msgpack:"keys"`
		Parents    []resource.ID   `json:"parents"     msgpack:"parents"`
		Scopes     []resource.Type `json:"scopes"      msgpack:"scopes"`
		Limit      int             `json:"limit"       msgpack:"limit"`
		Offset     int             `json:"offset"      msgpack:"offset"`
		// Ancestors adds every group above the returned groups to the response.
		Ancestors bool `json:"ancestors" msgpack:"ancestors"`
	}
	RetrieveResponse struct {
		Groups    []group.Group `json:"groups"              msgpack:"groups"`
		Ancestors []group.Group `json:"ancestors,omitempty" msgpack:"ancestors,omitempty"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var (
		res RetrieveResponse
		q   = s.internal.NewRetrieve().Search(req.SearchTerm)
	)
	if len(req.Keys) > 0 {
		q = q.Where(group.MatchKeys(req.Keys...))
	}
	if len(req.Parents) > 0 {
		q = q.Where(group.MatchParents(req.Parents...))
	}
	if len(req.Scopes) > 0 {
		q = q.Where(group.MatchScopes(req.Scopes...))
	}
	if req.Nested != nil {
		q = q.Where(group.MatchNested(*req.Nested))
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if err := q.Entries(&res.Groups).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if req.Ancestors && len(res.Groups) > 0 {
		var err error
		if res.Ancestors, err = s.internal.Ancestors(
			ctx,
			nil,
			group.KeysFromGroups(res.Groups)...,
		); err != nil {
			return RetrieveResponse{}, err
		}
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: group.ResourceIDsFromGroups(append(res.Groups, res.Ancestors...)),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []group.Key `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: group.ResourceIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

type RenameRequest struct {
	Name string    `json:"name" msgpack:"name"`
	Key  group.Key `json:"key"  msgpack:"key"`
}

func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []resource.ID{group.ResourceID(req.Key)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

// SetParentRequest moves groups under Parent, or to the top level when Parent is nil.
type SetParentRequest struct {
	Parent *resource.ID `json:"parent" msgpack:"parent"`
	Keys   []group.Key  `json:"keys"   msgpack:"keys"`
}

func (s *Service) SetParent(
	ctx context.Context,
	tx gorp.Tx,
	req SetParentRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: group.ResourceIDs(req.Keys),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).SetParent(ctx, req.Parent, req.Keys...)
}
