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

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	ontology *ontology.Ontology
	access   *rbac.Service
}

func NewService(cfg config.Config) *Service {
	return &Service{
		ontology: cfg.Distribution.Ontology,
		access:   cfg.Service.RBAC,
		db:       cfg.Distribution.DB,
	}
}

type (
	RetrieveRequest struct {
		SearchTerm       string          `json:"search_term" msgpack:"search_term"`
		IDs              []ontology.ID   `json:"ids" msgpack:"ids" validate:"required"`
		Types            []ontology.Type `json:"types" msgpack:"types"`
		Limit            int             `json:"limit" msgpack:"limit"`
		Offset           int             `json:"offset" msgpack:"offset"`
		Children         bool            `json:"children" msgpack:"children"`
		Parents          bool            `json:"parents" msgpack:"parents"`
		ExcludeFieldData bool            `json:"exclude_field_data" msgpack:"exclude_field_data"`
	}
	RetrieveResponse struct {
		Resources []ontology.Resource `json:"resources" msgpack:"resources"`
	}
)

func (o *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	if req.SearchTerm != "" {
		resources, err := o.ontology.Search(ctx, ontology.SearchRequest{
			Term: req.SearchTerm,
		})
		if err != nil {
			return RetrieveResponse{}, err
		}
		return RetrieveResponse{Resources: resources}, nil
	}
	q := o.ontology.NewRetrieve()
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
	var resources []ontology.Resource
	if err := q.Entries(&resources).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err := o.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: ontology.ResourceIDs(resources),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return RetrieveResponse{Resources: resources}, nil
}

type AddChildrenRequest struct {
	ID       ontology.ID   `json:"id" msgpack:"id" validate:"required"`
	Children []ontology.ID `json:"children" msgpack:"children" validate:"required"`
}

func (o *Service) AddChildren(
	ctx context.Context,
	req AddChildrenRequest,
) (res types.Nil, err error) {
	if err = o.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.ID),
	}); err != nil {
		return res, err
	}
	return res, o.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := o.ontology.NewWriter(tx)
		for _, child := range req.Children {
			if err := w.DefineRelationship(ctx, req.ID, ontology.RelationshipTypeParentOf, child); err != nil {
				return err
			}
		}
		return nil
	})
}

type RemoveChildrenRequest struct {
	ID       ontology.ID   `json:"id" msgpack:"id" validate:"required"`
	Children []ontology.ID `json:"children" msgpack:"children" validate:"required"`
}

func (o *Service) RemoveChildren(
	ctx context.Context,
	req RemoveChildrenRequest,
) (res types.Nil, err error) {
	if err = o.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.ID),
	}); err != nil {
		return res, err
	}
	return res, o.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := o.ontology.NewWriter(tx)
		for _, child := range req.Children {
			if err := w.DeleteRelationship(ctx, req.ID, ontology.RelationshipTypeParentOf, child); err != nil {
				return err
			}
		}
		return nil
	})
}

type MoveChildrenRequest struct {
	From     ontology.ID   `json:"from" msgpack:"from" validate:"required"`
	To       ontology.ID   `json:"to" msgpack:"to" validate:"required"`
	Children []ontology.ID `json:"children" msgpack:"children" validate:"required"`
}

func (o *Service) MoveChildren(
	ctx context.Context,
	req MoveChildrenRequest,
) (res types.Nil, err error) {
	if err = o.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.From, req.To),
	}); err != nil {
		return res, err
	}
	return res, o.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := o.ontology.NewWriter(tx)
		for _, child := range req.Children {
			if err = w.DeleteRelationship(ctx, req.From, ontology.RelationshipTypeParentOf, child); err != nil {
				return err
			}
			if err = w.DefineRelationship(ctx, req.To, ontology.RelationshipTypeParentOf, child); err != nil {
				return err
			}
		}
		return nil
	})
}
