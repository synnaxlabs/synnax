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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the group API service.
type ServiceConfig struct {
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
	c.Group = override.Nil(c.Group, other.Group)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.group")
	v.NotNil("group", c.Group)
	return v.Error()
}

type Service struct {
	internal *group.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
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

// Create creates a group. An existing group with the key keeps its stored parent and
// scope.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	g := group.Group{Key: req.Key, Name: req.Name, Scope: req.Scope, Parent: req.Parent}
	if err := s.internal.NewWriter(tx).Create(ctx, &g); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Group: g}, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the groups that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, group.ResourceID)
}

// Retrieve returns the groups that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Groups))
	if err != nil {
		return RetrieveResponse{}, err
	}
	return res, q.Where(bind.Filter(snap, group.Match)).Exec(ctx, snap)
}

type (
	RetrieveAncestorsRequest struct {
		Keys []group.Key `json:"keys" msgpack:"keys"`
	}
	RetrieveAncestorsResponse struct {
		Groups []group.Group `json:"groups" msgpack:"groups"`
	}
)

var _ bind.Named = RetrieveAncestorsRequest{}

// Named implements bind.Named.
func (r RetrieveAncestorsRequest) Named() []resource.ID {
	return group.ResourceIDs(r.Keys)
}

// RetrieveAncestors returns every group above the groups with the requested keys that
// the subject can retrieve.
func (s *Service) RetrieveAncestors(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveAncestorsRequest,
) (RetrieveAncestorsResponse, error) {
	if len(req.Keys) == 0 {
		return RetrieveAncestorsResponse{}, nil
	}
	ancestors, err := s.internal.Ancestors(ctx, snap, req.Keys...)
	if err != nil {
		return RetrieveAncestorsResponse{}, err
	}
	return RetrieveAncestorsResponse{Groups: lo.Filter(
		ancestors,
		func(g group.Group, _ int) bool { return snap.Allowed(g.ResourceID()) },
	)}, nil
}

type DeleteRequest struct {
	Keys []group.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the groups with the given keys. It returns a validation error when a
// group holds a member or a group that the request does not name.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}

type RenameRequest struct {
	Name string    `json:"name" msgpack:"name"`
	Key  group.Key `json:"key"  msgpack:"key"`
}

// Rename renames the group with the given key.
func (s *Service) Rename(
	ctx context.Context,
	tx gorp.Tx,
	req RenameRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name)
}

// SetParentRequest moves groups under Parent, or to the top level when Parent is nil.
type SetParentRequest struct {
	Parent *resource.ID `json:"parent" msgpack:"parent"`
	Keys   []group.Key  `json:"keys"   msgpack:"keys"`
}

// SetParent moves the groups under a parent. A move under a new top parent moves the
// visualizations in the groups to the project of that top parent.
func (s *Service) SetParent(
	ctx context.Context,
	tx gorp.Tx,
	req SetParentRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).SetParent(ctx, req.Parent, req.Keys...)
}
