// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package connectivity

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/license"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig is the configuration for the connectivity API service.
type ServiceConfig struct {
	// Cluster provides information about the cluster topology.
	//
	// [REQUIRED]
	Cluster cluster.Cluster
	// License holds the Core's license.
	//
	// [REQUIRED]
	License *license.Service
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultServiceConfig is the default configuration for the connectivity API
	// service.
	DefaultServiceConfig = ServiceConfig{}
)

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Cluster = override.Nil(c.Cluster, other.Cluster)
	c.License = override.Nil(c.License, other.License)
	return c
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("api.connectivity")
	v.NotNil("cluster", c.Cluster)
	v.NotNil("license", c.License)
	return v.Error()
}

type Service struct {
	cluster cluster.Cluster
	license *license.Service
}

func NewService(cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{
		cluster: cfg.Cluster,
		license: cfg.License,
	}, nil
}

// CheckResponse describes the Core and the state of its license.
type CheckResponse struct {
	auth.ClusterInfo
	// License is the state of the Core's license.
	License license.State `json:"license" msgpack:"license"`
}

func (s *Service) Check(context.Context, struct{}) (CheckResponse, error) {
	return CheckResponse{
		ClusterKey:  s.cluster.Key().String(),
		NodeVersion: version.Get(),
		NodeKey:     s.cluster.HostKey(),
		NodeTime:    telem.Now(),
		License:     s.license.Retrieve().State,
	}, nil
}
