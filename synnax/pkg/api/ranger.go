package api

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/query"
	"go/types"

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

type RangeRenameRequest struct {
	Key  uuid.UUID `json:"key" msgpack:"key"`
	Name string    `json:"name" msgpack:"name"`
}

func (s *RangeService) Rename(ctx context.Context, req RangeRenameRequest) (res types.Nil, _ errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		return errors.MaybeQuery(s.internal.NewWriter(tx).Rename(ctx, req.Key, req.Name))
	})
}

type RangeDeleteRequest struct {
	Keys  []uuid.UUID `json:"keys" msgpack:"keys"`
	Names []string    `json:"names" msgpack:"names"`
}

func (s *RangeService) Delete(ctx context.Context, req RangeDeleteRequest) (res types.Nil, _ errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		for _, key := range req.Keys {
			if err := errors.MaybeQuery(s.internal.NewWriter(tx).Delete(ctx, key)); err.Occurred() {
				return err
			}
		}
		return errors.Nil
	})
}

type RangeKVGetRequest struct {
	Range uuid.UUID `json:"range" msgpack:"range"`
	Keys  []string  `json:"keys" msgpack:"keys"`
}

type RangeKVGetResponse struct {
	Pairs map[string]string `json:"pairs" msgpack:"pairs"`
}

func (s *RangeService) KVGet(ctx context.Context, req RangeKVGetRequest) (res RangeKVGetResponse, _ errors.Typed) {
	var r ranger.Range
	if err := errors.MaybeQuery(s.internal.NewRetrieve().Entry(&r).
		WhereKeys(req.Range).
		Exec(ctx, nil)); err.Occurred() {
		return res, err
	}
	pairs := make(map[string]string, len(req.Keys))
	for _, key := range req.Keys {
		value, err := r.Get(ctx, []byte(key))
		if roacherrors.Is(err, query.NotFound) {
			return res, errors.Query(roacherrors.Wrapf(query.NotFound, "key %s not found on range", key))
		} else if err != nil {
			return res, errors.MaybeQuery(err)
		}
		pairs[key] = string(value)
	}
	return RangeKVGetResponse{Pairs: pairs}, errors.Nil
}

type RangeKVSetRequest struct {
	Range uuid.UUID         `json:"range" msgpack:"range"`
	Pairs map[string]string `json:"pairs" msgpack:"pairs"`
}

func (s *RangeService) KVSet(ctx context.Context, req RangeKVSetRequest) (res types.Nil, _ errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		var rng ranger.Range
		if err := errors.MaybeQuery(s.internal.NewRetrieve().Entry(&rng).
			WhereKeys(req.Range).
			Exec(ctx, tx)); err.Occurred() {
			return err
		}
		rng = rng.UseTx(tx)
		for k, v := range req.Pairs {
			if err := rng.Set(ctx, []byte(k), []byte(v)); err != nil {
				return errors.MaybeQuery(err)
			}
		}
		return errors.Nil
	})
}

type RangeKVDeleteRequest struct {
	Range uuid.UUID `json:"range" msgpack:"range"`
	Keys  []string  `json:"keys" msgpack:"keys"`
}

func (s *RangeService) KVDelete(ctx context.Context, req RangeKVDeleteRequest) (res types.Nil, _ errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		var rng ranger.Range
		if err := errors.MaybeQuery(s.internal.NewRetrieve().Entry(&rng).
			WhereKeys(req.Range).
			Exec(ctx, tx)); err.Occurred() {
			return err
		}
		rng = rng.UseTx(tx)
		for _, key := range req.Keys {
			if err := rng.Delete(ctx, []byte(key)); err != nil {
				return errors.MaybeQuery(err)
			}
		}
		return errors.Nil
	})
}

type RangeAliasSetRequest struct {
	Range   uuid.UUID              `json:"range" msgpack:"range"`
	Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
}

func (s *RangeService) AliasSet(ctx context.Context, req RangeAliasSetRequest) (res types.Nil, _ errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		var rng ranger.Range
		if err := errors.MaybeQuery(s.internal.NewRetrieve().Entry(&rng).
			WhereKeys(req.Range).
			Exec(ctx, tx)); err.Occurred() {
			return err
		}
		rng = rng.UseTx(tx)
		for k, v := range req.Aliases {
			if err := rng.SetAlias(ctx, k, v); err != nil {
				return errors.MaybeQuery(err)
			}
		}
		return errors.Nil
	})
}

type RangeAliasResolveRequest struct {
	Range   uuid.UUID `json:"range" msgpack:"range"`
	Aliases []string  `json:"aliases" msgpack:"aliases"`
}

type RangeAliasResolveResponse struct {
	Aliases map[string]channel.Key `json:"aliases" msgpack:"aliases"`
}

func (s *RangeService) AliasResolve(ctx context.Context, req RangeAliasResolveRequest) (res RangeAliasResolveResponse, _ errors.Typed) {
	var r ranger.Range
	if err := errors.MaybeQuery(s.internal.NewRetrieve().Entry(&r).
		WhereKeys(req.Range).
		Exec(ctx, nil)); err.Occurred() {
		return res, err
	}
	aliases := make(map[string]channel.Key, len(req.Aliases))
	for _, alias := range req.Aliases {
		ch, err := r.ResolveAlias(ctx, alias)
		if err != nil {
			return res, errors.MaybeQuery(err)
		}
		aliases[alias] = ch
	}
	return RangeAliasResolveResponse{Aliases: aliases}, errors.Nil
}

type RangeAliasDeleteRequest struct {
	Range   uuid.UUID     `json:"range" msgpack:"range"`
	Aliases []channel.Key `json:"aliases" msgpack:"aliases"`
}

func (s *RangeService) AliasDelete(ctx context.Context, req RangeAliasDeleteRequest) (res types.Nil, _ errors.Typed) {
	return res, s.WithTx(ctx, func(tx gorp.Tx) errors.Typed {
		var rng ranger.Range
		if err := errors.MaybeQuery(s.internal.NewRetrieve().Entry(&rng).
			WhereKeys(req.Range).
			Exec(ctx, tx)); err.Occurred() {
			return err
		}
		rng = rng.UseTx(tx)
		for _, alias := range req.Aliases {
			if err := rng.DeleteAlias(ctx, alias); err != nil {
				return errors.MaybeQuery(err)
			}
		}
		return errors.Nil
	})
}

type RangeAliasListRequest struct {
	Range    uuid.UUID     `json:"range" msgpack:"range"`
	Channels []channel.Key `json:"channels" msgpack:"channels"`
}

type RangeAliasListResponse struct {
	Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
}

func (s *RangeService) AliasList(ctx context.Context, req RangeAliasListRequest) (res RangeAliasListResponse, _ errors.Typed) {
	var r ranger.Range
	if err := errors.MaybeQuery(s.internal.NewRetrieve().Entry(&r).
		WhereKeys(req.Range).
		Exec(ctx, nil)); err.Occurred() {
		return res, err
	}
	aliases, err := r.ListAliases(ctx)
	return RangeAliasListResponse{Aliases: aliases}, errors.MaybeQuery(err)
}
