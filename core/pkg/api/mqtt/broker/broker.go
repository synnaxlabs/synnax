// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package broker serves the MQTT broker routes.
package broker

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/broker"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the MQTT broker API service.
type ServiceConfig struct {
	// Broker stores and retrieves brokers.
	// [REQUIRED]
	Broker *broker.Service
	// Status retrieves broker statuses.
	// [REQUIRED]
	Status *status.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Broker = override.Nil(c.Broker, other.Broker)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.mqtt.broker")
	v.NotNil("broker", c.Broker)
	v.NotNil("status", c.Status)
	return v.Error()
}

// Service serves the MQTT broker routes.
type Service struct {
	cfg ServiceConfig
}

// NewService returns a Service. It returns a validation error when the configuration
// is invalid.
func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

type (
	// CreateRequest carries the brokers to create.
	CreateRequest struct {
		// Brokers are the brokers to store. A nil key mints a new one.
		Brokers []broker.Broker `json:"brokers" msgpack:"brokers"`
	}
	// CreateResponse carries the created brokers with their keys.
	CreateResponse struct {
		// Brokers are the stored brokers, without their statuses.
		Brokers []broker.Broker `json:"brokers" msgpack:"brokers"`
	}
)

// Create stores the given brokers.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	w := s.cfg.Broker.NewWriter(tx)
	for i := range req.Brokers {
		req.Brokers[i].Status = nil
		if err := w.Create(ctx, &req.Brokers[i]); err != nil {
			return CreateResponse{}, err
		}
	}
	return CreateResponse(req), nil
}

// UpdateRequest carries one field group to write to a stored broker.
type UpdateRequest struct {
	// Update is the field group to write.
	Update broker.Update `json:"update" msgpack:"update"`
	// Key selects the broker to update.
	Key broker.Key `json:"key" msgpack:"key"`
}

// Update writes the field group of req to the broker stored under req.Key, as
// broker.Writer.Apply does. It returns a validation error when the request carries no
// group.
func (s *Service) Update(
	ctx context.Context,
	tx gorp.Tx,
	req UpdateRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Broker.NewWriter(tx).Apply(ctx, req.Key, req.Update)
}

// DeleteRequest carries the keys of the brokers to delete.
type DeleteRequest struct {
	// Keys selects the brokers to delete.
	Keys []broker.Key `json:"keys" msgpack:"keys"`
}

// Delete removes the given brokers with their statuses.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Broker.NewWriter(tx).Delete(ctx, req.Keys...)
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the brokers that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, broker.ResourceID)
}

// Retrieve returns the brokers that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.cfg.Broker.NewRetrieve().Entries(&res.Brokers))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Where(bind.Filter(snap, broker.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if !req.IncludeStatus {
		return res, nil
	}
	ids := lo.Map(res.Brokers, func(srv broker.Broker, _ int) resource.ID {
		return srv.ResourceID()
	})
	statuses, err := s.cfg.Status.Join[broker.StatusDetails](ctx, snap, ids)
	if err != nil {
		return RetrieveResponse{}, err
	}
	for i, st := range statuses {
		res.Brokers[i].Status = st
	}
	return res, nil
}
