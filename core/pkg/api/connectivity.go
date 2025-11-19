// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/telem"
)

type ConnectivityService struct{ clusterProvider }

func NewConnectivityService(p Provider) *ConnectivityService {
	return &ConnectivityService{clusterProvider: p.cluster}
}

// ClusterInfo is general information about the cluster and node that the request was
// sent to.
type ClusterInfo struct {
	// ClusterKey is the key of the cluster.
	ClusterKey string `json:"cluster_key" msgpack:"cluster_key"`
	// NodeVersion is the current version of the Synnax Core being used.
	NodeVersion string `json:"node_version" msgpack:"node_version"`
	// NodeKey is the key of the node in the cluster that the request was sent to.
	NodeKey cluster.NodeKey `json:"node_key" msgpack:"node_key"`
	// NodeTime is the time of the node that the request was sent to.
	NodeTime telem.TimeStamp `json:"node_time" msgpack:"node_time"`
}

type ConnectivityCheckResponse = ClusterInfo

func (s *ConnectivityService) Check(
	context.Context,
	types.Nil,
) (ConnectivityCheckResponse, error) {
	return ConnectivityCheckResponse{
		ClusterKey:  s.clusterProvider.cluster.Key().String(),
		NodeVersion: version.Get(),
		NodeKey:     s.cluster.HostKey(),
		NodeTime:    telem.Now(),
	}, nil
}
