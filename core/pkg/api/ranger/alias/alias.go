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
	"maps"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the alias API service.
type ServiceConfig struct {
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
	c.Alias = override.Nil(c.Alias, other.Alias)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.ranger.alias")
	v.NotNil("alias", c.Alias)
	return v.Error()
}

type Service struct {
	alias *alias.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		alias: cfg.Alias,
	}, nil
}

type SetRequest struct {
	Aliases map[channel.Key]string `json:"aliases" msgpack:"aliases"`
	Range   ranger.Key             `json:"range"   msgpack:"range"`
}

// Set creates or updates the aliases of the channels on the range. It returns an error
// wrapping query.ErrNotFound when a channel does not exist.
func (s *Service) Set(
	ctx context.Context,
	tx gorp.Tx,
	req SetRequest,
) (struct{}, error) {
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

// Resolve returns the channel of each alias on the range. It leaves out an alias that
// names no channel or that the subject cannot retrieve.
func (s *Service) Resolve(
	ctx context.Context,
	snap bind.Snapshot,
	req ResolveRequest,
) (ResolveResponse, error) {
	reader := s.alias.NewReader(snap)
	aliases := make(map[string]channel.Key, len(req.Aliases))
	for _, a := range req.Aliases {
		ch, err := reader.Resolve(ctx, req.Range, a)
		if err != nil && !errors.Is(err, query.ErrNotFound) {
			return ResolveResponse{}, err
		}
		if ch != 0 && snap.Allowed(alias.ResourceID(req.Range, ch)) {
			aliases[a] = ch
		}
	}
	return ResolveResponse{Aliases: aliases}, nil
}

type DeleteRequest struct {
	Channels []channel.Key `json:"channels" msgpack:"channels"`
	Range    ranger.Key    `json:"range"    msgpack:"range"`
}

// Delete deletes the aliases of the channels on the range. A channel with no alias is
// ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
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

// List returns the alias of each channel on the range that the subject can retrieve.
func (s *Service) List(
	ctx context.Context,
	snap bind.Snapshot,
	req ListRequest,
) (ListResponse, error) {
	aliases, err := s.alias.NewReader(snap).List(ctx, req.Range)
	if err != nil {
		return ListResponse{}, err
	}
	maps.DeleteFunc(aliases, func(ch channel.Key, _ string) bool {
		return !snap.Allowed(alias.ResourceID(req.Range, ch))
	})
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

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It names the alias of each channel on the range.
func (r RetrieveRequest) Named() []resource.ID {
	return alias.ResourceIDs(r.Range, r.Channels)
}

// Retrieve returns the alias of each channel on the range. It leaves out a channel with
// no alias.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	reader := s.alias.NewReader(snap)
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
