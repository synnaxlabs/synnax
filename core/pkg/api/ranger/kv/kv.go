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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Pair = kv.Pair

// ServiceConfig is the configuration for the range key-value API service.
type ServiceConfig struct {
	// RBAC enforces access control on every request.
	//
	// [REQUIRED]
	RBAC *rbac.Service
	// KV stores and retrieves key-value pairs on ranges.
	//
	// [REQUIRED]
	KV *kv.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the range key-value API
	// service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.RBAC = override.Nil(c.RBAC, other.RBAC)
	c.KV = override.Nil(c.KV, other.KV)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ranger.kv")
	v.NotNil("rbac", c.RBAC)
	v.NotNil("kv", c.KV)
	return v.Error()
}

type Service struct {
	access *rbac.Service
	kv     *kv.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		access: cfg.RBAC,
		kv:     cfg.KV,
	}, nil
}

type (
	GetRequest struct {
		Keys  []string   `json:"keys"  msgpack:"keys"`
		Range ranger.Key `json:"range" msgpack:"range"`
	}
	GetResponse struct {
		Pairs []kv.Pair `json:"pairs" msgpack:"pairs"`
	}
)

func (s *Service) Get(ctx context.Context, req GetRequest) (GetResponse, error) {
	if err := s.access.NewEnforcer(nil).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionRetrieve,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return GetResponse{}, err
	}
	reader := s.kv.NewReader(nil)
	var (
		pairs []kv.Pair
		err   error
	)
	if len(req.Keys) == 0 {
		pairs, err = reader.List(ctx, req.Range)
	} else {
		pairs, err = reader.GetMany(ctx, req.Range, req.Keys)
	}
	if err != nil {
		return GetResponse{}, err
	}
	return GetResponse{Pairs: pairs}, nil
}

type SetRequest struct {
	Pairs []kv.Pair  `json:"pairs" msgpack:"pairs"`
	Range ranger.Key `json:"range" msgpack:"range"`
}

func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, s.kv.NewWriter(tx).SetMany(ctx, req.Pairs)
}

type DeleteRequest struct {
	Keys  []string   `json:"keys"  msgpack:"keys"`
	Range ranger.Key `json:"range" msgpack:"range"`
}

func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	if err := s.access.NewEnforcer(tx).Enforce(ctx, access.Request{
		Subject: auth.GetSubject(ctx),
		Action:  access.ActionUpdate,
		Objects: []ontology.ID{ranger.OntologyID(req.Range)},
	}); err != nil {
		return struct{}{}, err
	}
	w := s.kv.NewWriter(tx)
	for _, key := range req.Keys {
		if err := w.Delete(ctx, req.Range, key); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}
