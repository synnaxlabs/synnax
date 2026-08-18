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
	"go/types"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type Service struct {
	ontology *ontology.Ontology
	search   *search.Index
	access   *rbac.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		ontology: cfg.Service.Ontology,
		search:   cfg.Service.Search,
		access:   cfg.Service.RBAC,
	}, nil
}

type (
	RetrieveRequest struct {
		SearchTerm          string                  `json:"search_term"            msgpack:"search_term"`
		IDs                 []ontology.ID           `json:"ids"                    msgpack:"ids"                    validate:"required"`
		Types               []ontology.ResourceType `json:"types"                  msgpack:"types"`
		Limit               int                     `json:"limit"                  msgpack:"limit"`
		Offset              int                     `json:"offset"                 msgpack:"offset"`
		Children            bool                    `json:"children"               msgpack:"children"`
		Parents             bool                    `json:"parents"                msgpack:"parents"`
		ExcludeFieldData    bool                    `json:"exclude_field_data"     msgpack:"exclude_field_data"`
		IgnoreNotFoundError bool                    `json:"ignore_not_found_error" msgpack:"ignore_not_found_error"`
	}
	RetrieveResponse struct {
		Resources []ontology.Resource `json:"resources" msgpack:"resources"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	resources := make([]ontology.Resource, 0)
	if req.SearchTerm != "" {
		ids, err := s.search.Search(ctx, search.Request{Term: req.SearchTerm})
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
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: ontology.ResourceIDs(resources),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Resources: resources}, nil
}

type AddChildrenRequest struct {
	ID       ontology.ID   `json:"id"       msgpack:"id"       validate:"required"`
	Children []ontology.ID `json:"children" msgpack:"children" validate:"required"`
}

func (s *Service) AddChildren(
	ctx context.Context,
	tx gorp.Tx,
	req AddChildrenRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.ID),
	}); err != nil {
		return types.Nil{}, err
	}
	if err := s.ontology.NewWriter(tx).DefineRelationships(
		ctx,
		req.ID,
		ontology.RelationshipTypeParentOf,
		req.Children...,
	); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, nil
}

type RemoveChildrenRequest struct {
	ID       ontology.ID   `json:"id"       msgpack:"id"       validate:"required"`
	Children []ontology.ID `json:"children" msgpack:"children" validate:"required"`
}

func (s *Service) RemoveChildren(
	ctx context.Context,
	tx gorp.Tx,
	req RemoveChildrenRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.ID),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.ontology.NewWriter(tx).DeleteRelationships(
		ctx,
		lo.Map(req.Children, func(child ontology.ID, _ int) ontology.Relationship {
			return ontology.Relationship{
				From: req.ID,
				Type: ontology.RelationshipTypeParentOf,
				To:   child,
			}
		})...,
	)
}

type MoveChildrenRequest struct {
	From     ontology.ID   `json:"from"     msgpack:"from"     validate:"required"`
	To       ontology.ID   `json:"to"       msgpack:"to"       validate:"required"`
	Children []ontology.ID `json:"children" msgpack:"children" validate:"required"`
}

func (s *Service) MoveChildren(
	ctx context.Context,
	tx gorp.Tx,
	req MoveChildrenRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.From, req.To),
	}); err != nil {
		return types.Nil{}, err
	}
	w := s.ontology.NewWriter(tx)
	for _, child := range req.Children {
		if err := w.DeleteRelationships(ctx, ontology.Relationship{
			From: req.From,
			Type: ontology.RelationshipTypeParentOf,
			To:   child,
		}); err != nil {
			return types.Nil{}, err
		}
		if err := w.DefineRelationships(
			ctx,
			req.To,
			ontology.RelationshipTypeParentOf,
			child,
		); err != nil {
			return types.Nil{}, err
		}
	}
	return types.Nil{}, nil
}
