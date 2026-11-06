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
	"slices"

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
	enforcer := s.access.NewEnforcer(tx)
	if err := enforcer.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []resource.ID{{Type: resource.TypeGroup}},
	}); err != nil {
		return CreateResponse{}, err
	}
	if req.Parent != nil {
		if err := enforcer.Enforce(ctx, access.Request{
			Subject: auth.GetSubject(ctx),
			Action:  access.ActionUpdate,
			Objects: []resource.ID{*req.Parent},
		}); err != nil {
			return CreateResponse{}, err
		}
	}
	g := group.Group{Key: req.Key, Name: req.Name, Scope: req.Scope, Parent: req.Parent}
	if err := s.internal.NewWriter(tx).Create(ctx, &g); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Group: g}, nil
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := req.Apply(s.internal.NewRetrieve().Entries(&res.Groups))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: group.ResourceIDsFromGroups(res.Groups),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type (
	RetrieveAncestorsRequest struct {
		Keys []group.Key `json:"keys" msgpack:"keys"`
	}
	RetrieveAncestorsResponse struct {
		Groups []group.Group `json:"groups" msgpack:"groups"`
	}
)

// RetrieveAncestors returns every group above the groups with the requested keys.
func (s *Service) RetrieveAncestors(
	ctx context.Context,
	req RetrieveAncestorsRequest,
) (RetrieveAncestorsResponse, error) {
	var (
		res RetrieveAncestorsResponse
		err error
	)
	if len(req.Keys) > 0 {
		if res.Groups, err = s.internal.Ancestors(ctx, nil, req.Keys...); err != nil {
			return RetrieveAncestorsResponse{}, err
		}
	}
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: append(
			group.ResourceIDs(req.Keys),
			group.ResourceIDsFromGroups(res.Groups)...,
		),
	}); err != nil {
		return RetrieveAncestorsResponse{}, err
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

// SetParent moves the groups. It requires update access on the groups, on the parents
// they leave, and on the parent they join.
func (s *Service) SetParent(
	ctx context.Context,
	tx gorp.Tx,
	req SetParentRequest,
) (struct{}, error) {
	var groups []group.Group
	if err := s.internal.NewRetrieve().
		Where(group.MatchKeys(req.Keys...)).
		Entries(&groups).
		Exec(ctx, tx); err != nil {
		return struct{}{}, err
	}
	objects := group.ResourceIDs(req.Keys)
	for _, g := range append(groups, group.Group{Parent: req.Parent}) {
		if g.Parent != nil && !slices.Contains(objects, *g.Parent) {
			objects = append(objects, *g.Parent)
		}
	}
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: objects,
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.internal.NewWriter(tx).SetParent(ctx, req.Parent, req.Keys...)
}
