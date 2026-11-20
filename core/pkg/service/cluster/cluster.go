// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package cluster re-exports the identity of a Core from pkg/distribution/cluster, so
// that the service and API layers depend on one cluster package.
package cluster

import "github.com/synnaxlabs/synnax/pkg/distribution/cluster"

// Identity names a Core: the cluster it belongs to and its own node in that cluster.
type Identity = cluster.Identity
