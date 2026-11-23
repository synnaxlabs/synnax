// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ gorp.Entry[Key] = Cluster{}

// CustomTypeName gives clusters a table prefix that cannot collide with the legacy
// device table or another integration's resource.
func (Cluster) CustomTypeName() string { return "kafka_cluster" }

// GorpKey returns the key of the cluster.
func (c Cluster) GorpKey() Key { return c.Key }

// ResourceID returns the resource ID of the cluster.
func (c Cluster) ResourceID() resource.ID {
	return resource.ID{Type: resource.TypeKafkaCluster, Key: c.Key.String()}
}
