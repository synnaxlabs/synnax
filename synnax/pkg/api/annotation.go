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
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/annotation"
	"github.com/synnaxlabs/x/gorp"
)

type AnnotationService struct {
	dbProvider
	accessProvider
	internal *annotation.Service
}

func NewAnnotationService(p Provider) *AnnotationService {
	return &AnnotationService{
		dbProvider:     p.db,
		accessProvider: p.access,
		internal:       p.Service.Annotation,
	}
}

type (
	AnnotationCreateRequest struct {
		Annotations []annotation.Annotation `json:"annotations" msgpack:"annotations"`
	}
	AnnotationCreateResponse = AnnotationCreateRequest
)

func (s *AnnotationService) Create(ctx context.Context, req AnnotationCreateRequest) (res AnnotationCreateResponse, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Create,
		Objects: annotation.OntologyIDsFromAnnotations(req.Annotations),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		for i, annotation_ := range req.Annotations {
			if err = w.Create(ctx, &annotation_); err != nil {
				return err
			}
			req.Annotations[i] = annotation_
		}
		res.Annotations = req.Annotations
		return nil
	})
}

type AnnotationDeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *AnnotationService) Delete(ctx context.Context, req AnnotationDeleteRequest) (res types.Nil, err error) {
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Delete,
		Objects: annotation.OntologyIDs(req.Keys),
	}); err != nil {
		return res, err
	}
	return res, s.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
	})
}

type (
	AnnotationRetrieveRequest struct {
		Keys []uuid.UUID `json:"keys" msgpack:"keys"`
	}
	AnnotationRetrieveResponse struct {
		Annotations []annotation.Annotation `json:"annotations" msgpack:"annotations"`
	}
)

func (s *AnnotationService) Retrieve(ctx context.Context, req AnnotationRetrieveRequest) (res AnnotationRetrieveResponse, err error) {
	if err = s.internal.NewRetrieve().WhereKeys(req.Keys...).Entries(&res.Annotations).Exec(ctx, nil); err != nil {
		return AnnotationRetrieveResponse{}, err
	}
	if err = s.access.Enforce(ctx, access.Request{
		Subject: getSubject(ctx),
		Action:  access.Retrieve,
		Objects: annotation.OntologyIDsFromAnnotations(res.Annotations),
	}); err != nil {
		return AnnotationRetrieveResponse{}, err
	}
	return res, err
}
