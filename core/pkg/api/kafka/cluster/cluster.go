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
