// Copyright 2025 Synnax Labs, Inc.
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
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/telem"
)

type Service struct {
	cluster cluster.Cluster
}

func NewService(cfg config.Config) *Service {
	return &Service{cluster: cfg.Distribution.Cluster}
}

type CheckResponse = auth.ClusterInfo

func (s *Service) Check(
	context.Context,
	types.Nil,
) (CheckResponse, error) {
	return CheckResponse{
		ClusterKey:  s.cluster.Key().String(),
		NodeVersion: version.Get(),
		NodeKey:     s.cluster.HostKey(),
		NodeTime:    telem.Now(),
	}, nil
}
