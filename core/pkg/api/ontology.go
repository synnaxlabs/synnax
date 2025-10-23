// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/search"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/x/gorp"
)

type OntologyService struct {
	dbProvider
	OntologyProvider
	accessProvider
	group *group.Service
}

func NewOntologyService(p Provider) *OntologyService {
	return &OntologyService{
		OntologyProvider: p.ontology,
		accessProvider:   p.access,
		dbProvider:       p.db,
		group:            p.Distribution.Group,
	}
}

type (
	OntologyRetrieveRequest struct {
		IDs              []ontology.ID   `json:"ids" msgpack:"ids" validate:"required"`
		Children         *bool           `json:"children" msgpack:"children"`
		Parents          *bool           `json:"parents" msgpack:"parents"`
		ExcludeFieldData bool            `json:"exclude_field_data" msgpack:"exclude_field_data"`
		Types            []ontology.Type `json:"types" msgpack:"types"`
		SearchTerm       string          `json:"search_term" msgpack:"search_term"`
		Limit            int             `json:"limit" msgpack:"limit"`
		Offset           int             `json:"offset" msgpack:"offset"`
	}
	OntologyRetrieveResponse struct {
		Resources []ontology.Resource `json:"resources" msgpack:"resources"`
	}
)

func (o *OntologyService) Retrieve(
	ctx context.Context,
	req OntologyRetrieveRequest,
) (OntologyRetrieveResponse, error) {
	if req.SearchTerm != "" {
		resources, err := o.Ontology.Search(ctx, search.Request{Term: req.SearchTerm})
		if err != nil {
			return OntologyRetrieveResponse{}, err
		}
		return OntologyRetrieveResponse{Resources: resources}, nil
	}
	q := o.Ontology.NewRetrieve()
	if len(req.IDs) > 0 {
		q = q.WhereIDs(req.IDs...)
	}
	if req.Children != nil && *req.Children {
		q = q.TraverseTo(ontology.Children)
	}
	if req.Parents != nil && *req.Parents {
		q = q.TraverseTo(ontology.Parents)
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
		return OntologyRetrieveResponse{}, err
	}
	if err := o.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: ontology.ResourceIDs(resources),
	}); err != nil {
		return OntologyRetrieveResponse{}, err
	}
	return OntologyRetrieveResponse{Resources: resources}, nil
}

type (
	OntologyCreateGroupRequest struct {
		Name   string      `json:"name" msgpack:"name" validate:"required"`
		Key    uuid.UUID   `json:"key" msgpack:"key"`
		Parent ontology.ID `json:"parent" msgpack:"parent"`
	}
	OntologyCreateGroupResponse struct {
		Group group.Group `json:"group" msgpack:"group"`
	}
)

func (o *OntologyService) CreateGroup(
	ctx context.Context,
	req OntologyCreateGroupRequest,
) (res OntologyCreateGroupResponse, err error) {
	if err = o.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionCreate,
		Objects: []ontology.ID{group.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, o.WithTx(ctx, func(tx gorp.Tx) error {
		w := o.group.NewWriter(tx)
		g, err_ := w.CreateWithKey(ctx, req.Key, req.Name, req.Parent)
		if err_ != nil {
			return err_
		}
		res.Group = g
		return nil
	})
}

type OntologyDeleteGroupRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys" validate:"required"`
}

func (o *OntologyService) DeleteGroup(
	ctx context.Context,
	req OntologyDeleteGroupRequest,
) (res types.Nil, err error) {
	if err = o.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionDelete,
		Objects: group.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, o.WithTx(ctx, func(tx gorp.Tx) error {
		w := o.group.NewWriter(tx)
		return w.Delete(ctx, req.Keys...)
	})
}

type OntologyRenameGroupRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key" validate:"required"`
	Name string    `json:"name" msgpack:"name" validate:"required"`
}

func (o *OntologyService) RenameGroup(
	ctx context.Context,
	req OntologyRenameGroupRequest,
) (res types.Nil, err error) {
	if err = o.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{group.OntologyID(req.Key)},
	}); err != nil {
		return res, err
	}
	return res, o.WithTx(ctx, func(tx gorp.Tx) error {
		w := o.group.NewWriter(tx)
		return w.Rename(ctx, req.Key, req.Name)
	})
}

type OntologyAddChildrenRequest struct {
	ID       ontology.ID   `json:"id" msgpack:"id" validate:"required"`
	Children []ontology.ID `json:"children" msgpack:"children" validate:"required"`
}

func (o *OntologyService) AddChildren(
	ctx context.Context,
	req OntologyAddChildrenRequest,
) (res types.Nil, err error) {
	if err = o.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.ID),
	}); err != nil {
		return res, err
	}
	return res, o.WithTx(ctx, func(tx gorp.Tx) error {
		w := o.Ontology.NewWriter(tx)
		for _, child := range req.Children {
			if err := w.DefineRelationship(ctx, req.ID, ontology.ParentOf, child); err != nil {
				return err
			}
		}
		return nil
	})
}

type OntologyRemoveChildrenRequest struct {
	ID       ontology.ID   `json:"id" msgpack:"id" validate:"required"`
	Children []ontology.ID `json:"children" msgpack:"children" validate:"required"`
}

func (o *OntologyService) RemoveChildren(
	ctx context.Context,
	req OntologyRemoveChildrenRequest,
) (res types.Nil, err error) {
	if err = o.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.ID),
	}); err != nil {
		return res, err
	}
	return res, o.WithTx(ctx, func(tx gorp.Tx) error {
		w := o.Ontology.NewWriter(tx)
		for _, child := range req.Children {
			if err := w.DeleteRelationship(ctx, req.ID, ontology.ParentOf, child); err != nil {
				return err
			}
		}
		return nil
	})
}

type OntologyMoveChildrenRequest struct {
	From     ontology.ID   `json:"from" msgpack:"from" validate:"required"`
	To       ontology.ID   `json:"to" msgpack:"to" validate:"required"`
	Children []ontology.ID `json:"children" msgpack:"children" validate:"required"`
}

func (o *OntologyService) MoveChildren(
	ctx context.Context,
	req OntologyMoveChildrenRequest,
) (res types.Nil, err error) {
	if err = o.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(req.Children, req.From, req.To),
	}); err != nil {
		return res, err
	}
	return res, o.WithTx(ctx, func(tx gorp.Tx) error {
		w := o.Ontology.NewWriter(tx)
		for _, child := range req.Children {
			if err = w.DeleteRelationship(ctx, req.From, ontology.ParentOf, child); err != nil {
				return err
			}
			if err = w.DefineRelationship(ctx, req.To, ontology.ParentOf, child); err != nil {
				return err
			}
		}
		return nil
	})
}
