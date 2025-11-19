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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/gorp"
)

type LabelService struct {
	dbProvider
	accessProvider
	internal *label.Service
}

func NewLabelService(p Provider) *LabelService {
	return &LabelService{
		internal:       p.Service.Label,
		dbProvider:     p.db,
		accessProvider: p.access,
	}
}

type Label = label.Label

// LabelCreateRequest is a request to create a Label in the cluster.
type LabelCreateRequest struct {
	// Labels are the labels to create.
	Labels []Label `json:"labels" msgpack:"labels"`
}

// LabelCreateResponse is a response to a LabelCreateRequest.
type LabelCreateResponse struct {
	// Labels are the labels that were created.
	Labels []Label `json:"labels" msgpack:"labels"`
}

// Create creates the labels in the cluster.
func (s *LabelService) Create(
	ctx context.Context,
	req LabelCreateRequest,
) (res LabelCreateResponse, err error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: label.OntologyIDsFromLabels(req.Labels),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Labels)
		if err != nil {
			return err
		}
		res.Labels = req.Labels
		return nil
	})
}

type LabelRetrieveRequest struct {
	Keys       []uuid.UUID `json:"keys" msgpack:"keys"`
	Names      []string    `json:"names" msgpack:"names"`
	For        ontology.ID `json:"for" msgpack:"for"`
	SearchTerm string      `json:"search_term" msgpack:"search_term"`
	Limit      int         `json:"limit" msgpack:"limit"`
	Offset     int         `json:"offset" msgpack:"offset"`
}

type LabelRetrieveResponse struct {
	// Labels are the labels that were retrieved.
	Labels []Label `json:"labels" msgpack:"labels"`
}

func (s *LabelService) Retrieve(
	ctx context.Context,
	req LabelRetrieveRequest,
) (res LabelRetrieveResponse, err error) {
	if !req.For.IsZero() {
		res.Labels, err = s.internal.RetrieveFor(ctx, req.For, nil)
		return
	}

	q := s.internal.NewRetrieve()

	if req.SearchTerm != "" {
		q = q.Search(req.SearchTerm)
	}
	if req.Limit != 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset != 0 {
		q = q.Offset(req.Offset)
	}
	if len(req.Keys) != 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if len(req.Names) != 0 {
		q = q.WhereNames(req.Names...)
	}

	if err = q.Entries(&res.Labels).Exec(ctx, nil); err != nil {
		return LabelRetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: label.OntologyIDsFromLabels(res.Labels),
	}); err != nil {
		return LabelRetrieveResponse{}, err
	}
	return res, nil
}

type LabelDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *LabelService) Delete(
	ctx context.Context,
	req LabelDeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: label.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).DeleteMany(ctx, req.Keys)
	})
}

type LabelAddRequest struct {
	Labels  []uuid.UUID `json:"labels" msgpack:"labels" validate:"required"`
	Replace bool        `json:"replace" msgpack:"replace"`
	ID      ontology.ID `json:"id" msgpack:"id" validate:"required"`
}

func (s *LabelService) Add(
	ctx context.Context,
	req LabelAddRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Update,
		Objects: append(label.OntologyIDs(req.Labels), req.ID),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		if req.Replace {
			if err := w.Clear(ctx, req.ID); err != nil {
				return err
			}
		}
		return w.Label(ctx, req.ID, req.Labels)
	})
}

type LabelRemoveRequest struct {
	ID     ontology.ID `json:"id" msgpack:"id" validate:"required"`
	Labels []uuid.UUID `json:"labels" msgpack:"labels" validate:"required"`
}

func (s *LabelService) Remove(
	ctx context.Context,
	req LabelRemoveRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Update,
		Objects: append(label.OntologyIDs(req.Labels), req.ID),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).RemoveLabel(ctx, req.ID, req.Labels)
	})
}
