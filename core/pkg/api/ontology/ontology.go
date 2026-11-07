// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the ontology API service.
type ServiceConfig struct {
	// Ontology stores the relationships between resources.
	//
	// [REQUIRED]
	Ontology *ontology.Ontology
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the ontology API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ontology")
	v.NotNil("ontology", c.Ontology)
	v.NotNil("rbac", c.RBAC)
	return v.Error()
}

type Service struct {
	ontology *ontology.Ontology
	access   *rbac.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		ontology: cfg.Ontology,
		access:   cfg.RBAC,
	}, nil
}

type (
	RetrieveRequest struct {
		SearchTerm          string          `json:"search_term"            msgpack:"search_term"`
		IDs                 []resource.ID   `json:"ids"                    msgpack:"ids"`
		Types               []resource.Type `json:"types"                  msgpack:"types"`
		Limit               int             `json:"limit"                  msgpack:"limit"`
		Offset              int             `json:"offset"                 msgpack:"offset"`
		Children            bool            `json:"children"               msgpack:"children"`
		Parents             bool            `json:"parents"                msgpack:"parents"`
		ExcludeFieldData    bool            `json:"exclude_field_data"     msgpack:"exclude_field_data"`
		IgnoreNotFoundError bool            `json:"ignore_not_found_error" msgpack:"ignore_not_found_error"`
		// Root limits a search to the descendants of Root, and ends the Ancestors walk
		// at Root.
		Root resource.ID `json:"root" msgpack:"root"`
		// Ancestors adds the resources and parent relationships on every path from the
		// retrieved resources up to Root, or up to the ontology root.
		Ancestors bool `json:"ancestors" msgpack:"ancestors"`
	}
	RetrieveResponse struct {
		Resources     []ontology.Resource     `json:"resources"               msgpack:"resources"`
		Ancestors     []ontology.Resource     `json:"ancestors,omitempty"     msgpack:"ancestors,omitempty"`
		Relationships []ontology.Relationship `json:"relationships,omitempty" msgpack:"relationships,omitempty"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	resources := make([]ontology.Resource, 0)
	if req.SearchTerm != "" {
		ids, err := s.ontology.Search(ctx, nil, ontology.SearchQuery{
			Term:   req.SearchTerm,
			Types:  req.Types,
			Root:   req.Root,
			Limit:  req.Limit,
			Offset: req.Offset,
		})
		if err != nil {
			return RetrieveResponse{}, err
		}
		resources = make([]ontology.Resource, 0, len(ids))
		err = s.ontology.NewRetrieve().
			WhereIDs(ids...).
			Entries(&resources).
			Exec(ctx, nil)
		if errors.Is(err, query.ErrNotFound) {
			err = nil
		}
		if err != nil {
			return RetrieveResponse{}, err
		}
	} else {
		q := s.ontology.NewRetrieve()
		if len(req.IDs) > 0 {
			q = q.WhereIDs(req.IDs...)
		}
		if req.Children {
			q = q.TraverseTo(ontology.ChildrenTraverser)
		}
		if req.Parents {
			q = q.TraverseTo(ontology.ParentsTraverser)
		}
		if len(req.Types) > 0 {
			q = q.WhereTypes(req.Types...)
		}
		q.ExcludeFieldData(req.ExcludeFieldData)
		if req.Limit > 0 {
			q = q.Limit(req.Limit)
		}
		if req.Offset > 0 {
			q = q.Offset(req.Offset)
		}
		err := q.Entries(&resources).Exec(ctx, nil)
		if req.IgnoreNotFoundError && err != nil {
			err = errors.Skip(err, query.ErrNotFound)
		}
		if err != nil {
			return RetrieveResponse{}, err
		}
	}
	res := RetrieveResponse{Resources: resources}
	if req.Ancestors {
		var err error
		if res.Relationships, err = s.ontology.Ancestors(
			nil,
			req.Root,
			ontology.ResourceIDs(resources)...,
		); err != nil {
			return RetrieveResponse{}, err
		}
		if res.Ancestors, err = s.retrieveAncestors(
			ctx,
			req,
			res.Relationships,
		); err != nil {
			return RetrieveResponse{}, err
		}
	}
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: append(
			ontology.ResourceIDs(resources),
			ontology.ResourceIDs(res.Ancestors)...,
		),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// retrieveAncestors returns the resources that rels lead up to, leaving out the end of
// the walk: req.Root, or the ontology root.
func (s *Service) retrieveAncestors(
	ctx context.Context,
	req RetrieveRequest,
	rels []ontology.Relationship,
) ([]ontology.Resource, error) {
	ids := make([]resource.ID, 0, len(rels))
	seen := make(set.Set[resource.ID], len(rels))
	for _, rel := range rels {
		if rel.From == req.Root || rel.From == ontology.RootID ||
			seen.Contains(rel.From) {
			continue
		}
		seen.Add(rel.From)
		ids = append(ids, rel.From)
	}
	ancestors := make([]ontology.Resource, 0, len(ids))
	if len(ids) == 0 {
		return ancestors, nil
	}
	if err := s.ontology.NewRetrieve().
		WhereIDs(ids...).
		ExcludeFieldData(req.ExcludeFieldData).
		Entries(&ancestors).
		Exec(ctx, nil); err != nil {
		return nil, err
	}
	return ancestors, nil
}

type AddChildrenRequest struct {
	ID       resource.ID   `json:"id"       msgpack:"id"`
	Children []resource.ID `json:"children" msgpack:"children"`
}

func (s *Service) AddChildren(
	ctx context.Context,
	tx gorp.Tx,
	req AddChildrenRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.ID),
	}); err != nil {
		return struct{}{}, err
	}
	if err := s.ontology.NewWriter(tx).DefineRelationships(
		ctx,
		req.ID,
		ontology.RelationshipTypeParentOf,
		req.Children...,
	); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}

type RemoveChildrenRequest struct {
	ID       resource.ID   `json:"id"       msgpack:"id"`
	Children []resource.ID `json:"children" msgpack:"children"`
}

func (s *Service) RemoveChildren(
	ctx context.Context,
	tx gorp.Tx,
	req RemoveChildrenRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.ID),
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.ontology.NewWriter(tx).DeleteRelationships(
		ctx,
		lo.Map(req.Children, func(child resource.ID, _ int) ontology.Relationship {
			return ontology.Relationship{
				From: req.ID,
				Type: ontology.RelationshipTypeParentOf,
				To:   child,
			}
		})...,
	)
}

type MoveChildrenRequest struct {
	From     resource.ID   `json:"from"     msgpack:"from"`
	To       resource.ID   `json:"to"       msgpack:"to"`
	Children []resource.ID `json:"children" msgpack:"children"`
}

func (s *Service) MoveChildren(
	ctx context.Context,
	tx gorp.Tx,
	req MoveChildrenRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.From, req.To),
	}); err != nil {
		return struct{}{}, err
	}
	w := s.ontology.NewWriter(tx)
	for _, child := range req.Children {
		if err := w.DeleteRelationships(ctx, ontology.Relationship{
			From: req.From,
			Type: ontology.RelationshipTypeParentOf,
			To:   child,
		}); err != nil {
			return struct{}{}, err
		}
		if err := w.DefineRelationships(
			ctx,
			req.To,
			ontology.RelationshipTypeParentOf,
			child,
		); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}
