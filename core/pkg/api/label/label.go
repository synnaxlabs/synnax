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
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

type Service struct {
	db       *gorp.DB
	access   *rbac.Service
	internal *label.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Service.Label,
		db:       cfg.Distribution.DB,
		access:   cfg.Service.RBAC,
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
	req CreateRequest,
) (res CreateResponse, err error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: label.OntologyIDsFromLabels(req.Labels),
	}); err != nil {
		return res, err
	}
	return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Labels)
		if err != nil {
			return err
		}
		res.Labels = req.Labels
		return nil
	})
}

type RetrieveRequest struct {
	For        ontology.ID `json:"for" msgpack:"for"`
	SearchTerm string      `json:"search_term" msgpack:"search_term"`
	Keys       []uuid.UUID `json:"keys" msgpack:"keys"`
	Names      []string    `json:"names" msgpack:"names"`
	Limit      int         `json:"limit" msgpack:"limit"`
	Offset     int         `json:"offset" msgpack:"offset"`
}

type RetrieveResponse struct {
	// Labels are the labels that were retrieved.
	Labels []Label `json:"labels" msgpack:"labels"`
}

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (res RetrieveResponse, err error) {
	if !req.For.IsZero() {
		res.Labels, err = s.internal.RetrieveFor(ctx, req.For, nil)
		return
	}

	q := s.internal.NewRetrieve()

	if req.SearchTerm != "" {
		q = q.Search(req.SearchTerm)
	}
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}
	if req.Offset > 0 {
		q = q.Offset(req.Offset)
	}
	if len(req.Keys) != 0 {
		q = q.WhereKeys(req.Keys...)
	}
	if len(req.Names) != 0 {
		q = q.WhereNames(req.Names...)
	}

	if err = q.Entries(&res.Labels).Exec(ctx, nil); err != nil {
		return RetrieveResponse{}, err
	}
	if err = s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: label.OntologyIDsFromLabels(res.Labels),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

type DeleteRequest struct {
	Keys []uuid.UUID `json:"keys" msgpack:"keys"`
}

func (s *Service) Delete(
	ctx context.Context,
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: label.OntologyIDs(req.Keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).DeleteMany(ctx, req.Keys)
	})
}

type AddRequest struct {
	ID      ontology.ID `json:"id" msgpack:"id" validate:"required"`
	Labels  []uuid.UUID `json:"labels" msgpack:"labels" validate:"required"`
	Replace bool        `json:"replace" msgpack:"replace"`
}

func (s *Service) Add(
	ctx context.Context,
	req AddRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(label.OntologyIDs(req.Labels), req.ID),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.internal.NewWriter(tx)
		if req.Replace {
			if err := w.Clear(ctx, req.ID); err != nil {
				return err
			}
		}
		return w.Label(ctx, req.ID, req.Labels)
	})
}

type RemoveRequest struct {
	ID     ontology.ID `json:"id" msgpack:"id" validate:"required"`
	Labels []uuid.UUID `json:"labels" msgpack:"labels" validate:"required"`
}

func (s *Service) Remove(
	ctx context.Context,
	req RemoveRequest,
) (types.Nil, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: append(label.OntologyIDs(req.Labels), req.ID),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.internal.NewWriter(tx).RemoveLabel(ctx, req.ID, req.Labels)
	})
}
