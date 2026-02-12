// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
)

type Pair = kv.Pair

type Service struct {
	db     *gorp.DB
	access *rbac.Service
	kv     *kv.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:     cfg.Distribution.DB,
		access: cfg.Service.RBAC,
		kv:     cfg.Service.KV,
	}, nil
}

type (
	GetRequest struct {
		Keys  []string  `json:"keys" msgpack:"keys"`
		Range uuid.UUID `json:"range" msgpack:"range"`
	}
	GetResponse struct {
		Pairs []kv.Pair `json:"pairs" msgpack:"pairs"`
	}
)

func (s *Service) Get(
	ctx context.Context,
	req GetRequest,
) (GetResponse, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return GetResponse{}, err
	}
	reader := s.kv.NewReader(nil)
	var (
		res GetResponse
		err error
	)
	if len(req.Keys) == 0 {
		res.Pairs, err = reader.List(ctx, req.Range)
		if err != nil {
			return GetResponse{}, err
		}
		return res, nil
	}
	res.Pairs, err = reader.GetMany(ctx, req.Range, req.Keys)
	if err != nil {
		return GetResponse{}, err
	}
	return res, nil
}

type SetRequest struct {
	Pairs []kv.Pair `json:"pairs" msgpack:"pairs"`
	Range uuid.UUID `json:"range" msgpack:"range"`
}

func (s *Service) Set(
	ctx context.Context,
	req SetRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		return s.kv.NewWriter(tx).SetMany(ctx, req.Range, req.Pairs)
	})
}

type DeleteRequest struct {
	Keys  []string  `json:"keys" msgpack:"keys"`
	Range uuid.UUID `json:"range" msgpack:"range"`
}

func (s *Service) Delete(
	ctx context.Context,
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.kv.NewWriter(tx)
		for _, key := range req.Keys {
			if err := w.Delete(ctx, req.Range, key); err != nil {
				return err
			}
		}
		return nil
	})
}
