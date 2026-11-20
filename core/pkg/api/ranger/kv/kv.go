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
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Pair = kv.Pair

// ServiceConfig is the configuration for the range key-value API service.
type ServiceConfig struct {
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
	c.KV = override.Nil(c.KV, other.KV)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ranger.kv")
	v.NotNil("kv", c.KV)
	return v.Error()
}

type Service struct {
	kv *kv.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		kv: cfg.KV,
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

var _ bind.Named = GetRequest{}

// Named implements bind.Named. It names the range that holds the pairs.
func (r GetRequest) Named() []resource.ID {
	return []resource.ID{ranger.ResourceID(r.Range)}
}

// Get returns the pairs of the range with the given keys, or every pair of the range
// when Keys is empty.
func (s *Service) Get(
	ctx context.Context,
	snap bind.Snapshot,
	req GetRequest,
) (GetResponse, error) {
	reader := s.kv.NewReader(snap)
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
	Pairs []kv.Pair `json:"pairs" msgpack:"pairs"`
}

// Set creates or updates the key-value pairs. Each pair names its own range.
func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
) (struct{}, error) {
	return struct{}{}, s.kv.NewWriter(tx).SetMany(ctx, req.Pairs)
}

// DispatchRequest carries a sequence of actions to apply to the pair with the Gorp key
// Key.
type DispatchRequest = actions.DispatchRequest[string, kv.Action]

// Dispatch applies the action sequence to the pair in tx.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.kv.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

type DeleteRequest struct {
	Keys  []string   `json:"keys"  msgpack:"keys"`
	Range ranger.Key `json:"range" msgpack:"range"`
}

// Delete deletes the pairs with the given keys from the range. A key that matches no
// pair is ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	w := s.kv.NewWriter(tx)
	for _, key := range req.Keys {
		if err := w.Delete(ctx, req.Range, key); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}
