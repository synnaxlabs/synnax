// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	xconfig "github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

type Service struct {
	db     *gorp.DB
	access *rbac.Service
	alias  *alias.Service
}

func NewService(cfgs ...config.LayerConfig) (*Service, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		db:     cfg.Distribution.DB,
		access: cfg.Service.RBAC,
		alias:  cfg.Service.Alias,
	}, nil
}

type SetRequest struct {
	Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
	Range   uuid.UUID              `json:"range" msgpack:"range"`
}

func (s *Service) Set(
	ctx context.Context,
	req SetRequest,
) (types.Nil, error) {
	keys := lo.Keys(req.Aliases)
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: alias.OntologyIDs(req.Range, keys),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.alias.NewWriter(tx)
		for k, v := range req.Aliases {
			if err := w.Set(ctx, req.Range, k, v); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	ResolveRequest struct {
		Aliases []string  `json:"aliases" msgpack:"aliases"`
		Range   uuid.UUID `json:"range" msgpack:"range"`
	}
	ResolveResponse struct {
		Aliases map[string]channel.Key `json:"aliases" msgpack:"aliases"`
	}
)

func (s *Service) Resolve(
	ctx context.Context,
	req ResolveRequest,
) (ResolveResponse, error) {
	reader := s.alias.NewReader(nil)
	aliases := make(map[string]channel.Key, len(req.Aliases))
	for _, a := range req.Aliases {
		ch, err := reader.Resolve(ctx, req.Range, a)
		if err != nil && !errors.Is(err, query.ErrNotFound) {
			return ResolveResponse{}, err
		}
		if ch != 0 {
			aliases[a] = ch
		}
	}
	keys := lo.Values(aliases)
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: alias.OntologyIDs(req.Range, keys),
	}); err != nil {
		return ResolveResponse{}, err
	}
	return ResolveResponse{Aliases: aliases}, nil
}

type DeleteRequest struct {
	Channels []channel.Key `json:"channels" msgpack:"channels"`
	Range    uuid.UUID     `json:"range" msgpack:"range"`
}

func (s *Service) Delete(
	ctx context.Context,
	req DeleteRequest,
) (types.Nil, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: alias.OntologyIDs(req.Range, req.Channels),
	}); err != nil {
		return types.Nil{}, err
	}
	return types.Nil{}, s.db.WithTx(ctx, func(tx gorp.Tx) error {
		w := s.alias.NewWriter(tx)
		for _, ch := range req.Channels {
			if err := w.Delete(ctx, req.Range, ch); err != nil {
				return err
			}
		}
		return nil
	})
}

type (
	ListRequest struct {
		Range uuid.UUID `json:"range" msgpack:"range"`
	}
	ListResponse struct {
		Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
	}
)

func (s *Service) List(
	ctx context.Context,
	req ListRequest,
) (ListResponse, error) {
	reader := s.alias.NewReader(nil)
	aliases, err := reader.List(ctx, req.Range)
	if err != nil {
		return ListResponse{}, err
	}
	keys := lo.Keys(aliases)
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: alias.OntologyIDs(req.Range, keys),
	}); err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Aliases: aliases}, nil
}

type (
	RetrieveRequest struct {
		Channels []channel.Key `json:"channels" msgpack:"channels"`
		Range    uuid.UUID     `json:"range" msgpack:"range"`
	}
	RetrieveResponse struct {
		Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	if err := s.access.Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: alias.OntologyIDs(req.Range, req.Channels),
	}); err != nil {
		return RetrieveResponse{}, err
	}
	reader := s.alias.NewReader(nil)
	aliases := make(map[channel.Key]string)
	for _, ch := range req.Channels {
		al, err := reader.Retrieve(ctx, req.Range, ch)
		if err != nil && !errors.Is(err, query.ErrNotFound) {
			return RetrieveResponse{}, err
		}
		if al != "" {
			aliases[ch] = al
		}
	}
	return RetrieveResponse{Aliases: aliases}, nil
}
