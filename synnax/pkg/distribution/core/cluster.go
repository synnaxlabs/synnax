// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import (
	"github.com/synnaxlabs/aspen"
)

type (
	Node         = aspen.Node
	NodeID       = aspen.NodeKey
	NodeState    = aspen.NodeState
	Cluster      = aspen.Cluster
	HostResolver = aspen.HostResolver
	Resolver     = aspen.Resolver
	ClusterState = aspen.ClusterState
)
