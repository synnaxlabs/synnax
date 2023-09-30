package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/ranger"
	"github.com/synnaxlabs/x/gorp"
)

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

type Range = ranger.Range

type RangeService struct {
	dbProvider
	internal *ranger.Service
}

func NewRangeService(p Provider) *RangeService {
	return &RangeService{
		dbProvider: p.db,
		internal:   p.Config.Ranger,
	}
}

type RangeCreateRequest struct {
	Ranges []Range `json:"ranges" msgpack:"ranges"`
}

type RangeCreateResponse struct {
	Ranges []Range `json:"ranges" msgpack:"ranges"`
}

func (s *RangeService) Create(ctx context.Context, req RangeCreateRequest) (res RangeCreateResponse, _ errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		err := s.internal.NewWriter(tx).CreateMany(ctx, &req.Ranges)
		res = RangeCreateResponse{Ranges: req.Ranges}
		return errors.MaybeQuery(err)
	})
}

type RangeRetrieveRequest struct {
	Keys  []uuid.UUID `json:"keys" msgpack:"keys"`
	Names []string    `json:"names" msgpack:"names"`
	Term  string      `json:"term" msgpack:"term"`
}

type RangeRetrieveResponse struct {
	Ranges []Range `json:"ranges" msgpack:"ranges"`
}

func (s *RangeService) Retrieve(ctx context.Context, req RangeRetrieveRequest) (res RangeRetrieveResponse, _ errors.Typed) {
	var (
		resRanges []ranger.Range
		q         = s.internal.NewRetrieve().Entries(&resRanges)
		hasNames  = len(req.Names) > 0
		hasKeys   = len(req.Keys) > 0
		hasSearch = req.Term != ""
	)
	if hasNames {
		q = q.WhereNames(req.Names...)
	}
	if hasKeys {
		q = q.WhereKeys(req.Keys...)
	}
	if hasSearch {
		q = q.Search(req.Term)
	}
	err := errors.MaybeQuery(q.Exec(ctx, nil))
	return RangeRetrieveResponse{Ranges: resRanges}, err
}
