// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"context"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the panel API service.
type ServiceConfig struct {
	// Panel stores and retrieves panels.
	//
	// [REQUIRED]
	Panel *panel.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the panel API service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Panel = override.Nil(c.Panel, other.Panel)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.panel")
	v.NotNil("panel", c.Panel)
	return v.Error()
}

// Service is the API-layer panel service. Its write handlers run in the transaction
// of the write binder, which checks their access.
type Service struct {
	internal *panel.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		internal: cfg.Panel,
	}, nil
}

type (
	CreateRequest struct {
		// Panels are the panels to create. Each panel belongs to exactly one of a
		// project and a range.
		Panels []panel.Panel `json:"panels" msgpack:"panels"`
		// DispatchKey is stamped onto the create broadcasts so the sender skips its
		// echo.
		DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	}
	CreateResponse struct {
		Panels []panel.Panel `json:"panels" msgpack:"panels"`
	}
)

// Create creates the panels. It returns a validation error when a panel does not belong
// to exactly one project or range, or when a panel with its key exists.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	if err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		CreateMany(ctx, &req.Panels); err != nil {
		return CreateResponse{}, err
	}
	return CreateResponse{Panels: req.Panels}, nil
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the panels that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, panel.ResourceID)
}

// Retrieve returns the panels that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.internal.NewRetrieve().Entries(&res.Panels))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err = q.Where(bind.Filter(snap, panel.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	return res, nil
}

// DispatchRequest carries an action sequence to apply to a single panel.
// DispatchKey is a client-generated identifier for the batch, registered as
// outstanding by the client before the request so it can recognize and skip
// its own broadcast echo.
type DispatchRequest = actions.DispatchRequest[panel.Key, panel.Action]

// DispatchResponse carries the actions that the Core added to a dispatch, such as a set
// of the project that a move into a group of another project needs.
type DispatchResponse = actions.DispatchResponse[panel.Action]

// Dispatch applies the action sequence to the target panel in tx. Subscribers to the
// panel action signals receive the sequence after the transaction commits.
func (s *Service) Dispatch(
	ctx context.Context,
	tx gorp.Tx,
	req DispatchRequest,
) (DispatchResponse, error) {
	added, err := s.internal.NewWriter(tx).
		WithDispatchKey(req.DispatchKey).
		Apply(ctx, req.Key, req.Actions...)
	if err != nil {
		return DispatchResponse{}, err
	}
	return DispatchResponse{Actions: added}, nil
}

type DeleteRequest struct {
	Keys []panel.Key `json:"keys" msgpack:"keys"`
}

// Delete deletes the panels with the given keys. A key that matches no panel is
// ignored.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.internal.NewWriter(tx).Delete(ctx, req.Keys...)
}
