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

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the alias API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// Alias stores and retrieves channel aliases on ranges.
	//
	// [REQUIRED]
	Alias *alias.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the alias API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.Alias = override.Nil(c.Alias, other.Alias)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ranger.alias")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("alias", c.Alias)
	return v.Error()
}

type Service struct {
	access *rbac.Service
	alias  *alias.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access: cfg.RBAC,
		alias:  cfg.Alias,
	}, nil
}

type SetRequest struct {
	Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
	Range   ranger.Key             `json:"range"   msgpack:"range"`
}

func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
) (struct{}, error) {
	keys := lo.Keys(req.Aliases)
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionCreate,
		Objects: alias.OntologyIDs(req.Range, keys),
	}); err != nil {
		return struct{}{}, err
	}
	w := s.alias.NewWriter(tx)
	for k, v := range req.Aliases {
		if err := w.Set(ctx, req.Range, k, v); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

type (
	ResolveRequest struct {
		Aliases []string   `json:"aliases" msgpack:"aliases"`
		Range   ranger.Key `json:"range"   msgpack:"range"`
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
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
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
	Range    ranger.Key    `json:"range"    msgpack:"range"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionDelete,
		Objects: alias.OntologyIDs(req.Range, req.Channels),
	}); err != nil {
		return struct{}{}, err
	}
	w := s.alias.NewWriter(tx)
	for _, ch := range req.Channels {
		if err := w.Delete(ctx, req.Range, ch); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}

type (
	ListRequest struct {
		Range ranger.Key `json:"range" msgpack:"range"`
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
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
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
		Range    ranger.Key    `json:"range"    msgpack:"range"`
	}
	RetrieveResponse struct {
		Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
	}
)

func (s *Service) Retrieve(
	ctx context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
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
