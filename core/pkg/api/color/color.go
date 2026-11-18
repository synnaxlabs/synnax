// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the named color API service.
type ServiceConfig struct {
	// Color stores and retrieves named colors.
	//
	// [REQUIRED]
	Color *color.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the named color API
	// service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Color = override.Nil(c.Color, other.Color)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.color")
	v.NotNil("color", c.Color)
	return v.Error()
}

// Service is the API for named colors.
type Service struct{ internal *color.Service }

// NewService opens a Service with the given configuration.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{internal: cfg.Color}, nil
}

// CreateRequest holds the named colors to create.
type CreateRequest struct {
	// Colors are the named colors to create.
	Colors []color.Named `json:"colors" msgpack:"colors"`
	// DispatchKey is stamped onto the create broadcasts so the sender skips its echo.
	DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
}

// CreateResponse holds the created named colors.
type CreateResponse struct {
	// Colors are the created named colors, each with its key.
	Colors []color.Named `json:"colors" msgpack:"colors"`
}

// Create creates the named colors. It returns a validation error when a color is not
// valid, its group cannot hold a named color, or a named color with its key exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, req.Colors); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Colors: req.Colors}, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the named colors that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, color.ResourceID)
}

// Retrieve returns the named colors that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Colors))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, color.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// RetrieveBuiltinsResponse holds the built-in named colors.
type RetrieveBuiltinsResponse struct {
	// Colors are the built-in named colors with their shipped values.
	Colors []color.Named `json:"colors" msgpack:"colors"`
}

// RetrieveBuiltins returns the named colors that the Core ships, with their shipped
// values. Any signed-in subject can read them.
func (s *Service) RetrieveBuiltins(
	context.Context,
	struct{},
) (RetrieveBuiltinsResponse, error) {
	return RetrieveBuiltinsResponse{Colors: color.Builtins()}, nil
}

// DispatchRequest carries an action sequence to apply to one named color. DispatchKey
// is echoed on the broadcast so the sender recognizes its own echo.
type DispatchRequest = actions.DispatchRequest[color.Key, color.Action]

// Dispatch applies the actions to the named color atomically. Subscribers to the named
// color action signals receive them after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
}

// DeleteRequest selects the named colors to delete.
type DeleteRequest struct {
	// Keys are the keys of the named colors to delete.
	Keys []color.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the named colors. A key that matches no named color is ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
