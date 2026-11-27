// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package cluster serves the Kafka cluster routes.
package cluster

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/kafka/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the Kafka cluster API service.
type ServiceConfig struct {
	// Cluster stores and retrieves clusters.
	//
	// [REQUIRED]
	Cluster *cluster.Service
	// Status retrieves cluster statuses.
	//
	// [REQUIRED]
	Status *status.Service
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Cluster = override.Nil(c.Cluster, other.Cluster)
	c.Status = override.Nil(c.Status, other.Status)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.kafka.cluster")
	v.NotNil("cluster", c.Cluster)
	v.NotNil("status", c.Status)
	return v.Error()
}

// Service serves the Kafka cluster routes.
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
	// CreateRequest carries the clusters to create.
	CreateRequest struct {
		// Clusters are the clusters to store. A nil key mints a new one.
		Clusters []cluster.Cluster `json:"clusters" msgpack:"clusters"`
	}
	// CreateResponse carries the created clusters with their keys.
	CreateResponse struct {
		// Clusters are the stored clusters, without their statuses.
		Clusters []cluster.Cluster `json:"clusters" msgpack:"clusters"`
	}
)

// Create stores the given clusters.
func (s *Service) Create(
	ctx context.Context,
	tx gorp.Tx,
	req CreateRequest,
) (CreateResponse, error) {
	w := s.cfg.Cluster.NewWriter(tx)
	for i := range req.Clusters {
		req.Clusters[i].Status = nil
		if err := w.Create(ctx, &req.Clusters[i]); err != nil {
			return CreateResponse{}, err
		}
	}
	return CreateResponse(req), nil
}

// UpdateRequest carries one field group to write to a stored cluster.
type UpdateRequest struct {
	// Update is the field group to write.
	Update cluster.Update `json:"update" msgpack:"update"`
	// Key selects the cluster to update.
	Key cluster.Key `json:"key" msgpack:"key"`
}

// Update writes the field group of req to the cluster stored under req.Key, as
// cluster.Writer.Apply does. It returns a validation error when the request carries no
// group.
func (s *Service) Update(
	ctx context.Context,
	tx gorp.Tx,
	req UpdateRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Cluster.NewWriter(tx).Apply(ctx, req.Key, req.Update)
}

// DeleteRequest carries the keys of the clusters to delete.
type DeleteRequest struct {
	// Keys selects the clusters to delete.
	Keys []cluster.Key `json:"keys" msgpack:"keys"`
}

// Delete removes the given clusters with their statuses.
func (s *Service) Delete(
	ctx context.Context,
	tx gorp.Tx,
	req DeleteRequest,
) (struct{}, error) {
	return struct{}{}, s.cfg.Cluster.NewWriter(tx).Delete(ctx, req.Keys...)
}

var _ bind.Named = RetrieveRequest{}

// Named implements bind.Named. It returns the clusters that the top level of Where
// selects by key.
func (r RetrieveRequest) Named() []resource.ID {
	if r.Where == nil {
		return nil
	}
	return bind.Keys(r.Where.Key, cluster.ResourceID)
}

// Retrieve returns the clusters that req selects and that the subject can retrieve.
func (s *Service) Retrieve(
	ctx context.Context,
	snap bind.Snapshot,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	var res RetrieveResponse
	q, err := ApplyRetrieve(req, s.cfg.Cluster.NewRetrieve().Entries(&res.Clusters))
	if err != nil {
		return RetrieveResponse{}, err
	}
	if err := q.Where(bind.Filter(snap, cluster.Match)).Exec(ctx, snap); err != nil {
		return RetrieveResponse{}, err
	}
	if !req.IncludeStatus {
		return res, nil
	}
	ids := lo.Map(res.Clusters, func(srv cluster.Cluster, _ int) resource.ID {
		return srv.ResourceID()
	})
	statuses, err := s.cfg.Status.Join[cluster.StatusDetails](ctx, snap, ids)
	if err != nil {
		return RetrieveResponse{}, err
	}
	for i, st := range statuses {
		res.Clusters[i].Status = st
	}
	return res, nil
}
