// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package cluster names a Core and resolves the address of its host node. The node
// domain type itself lives in pkg/distribution/node.
package cluster

import (
	"uuid"

	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
)

// Identity names a Core: the cluster it belongs to and its own node in that cluster.
type Identity struct {
	// Key is the key of the cluster. Clients use it to tell Cores apart.
	Key uuid.UUID
	// HostKey is the key of the host node. Channel, rack, and task keys embed it.
	HostKey node.Key
}

type (
	// Resolver maps a node key to the network address at which that node can be
	// reached.
	Resolver = aspen.Resolver
	// HostResolver is a Resolver that also exposes information about the host node it
	// is running on (its key and full node record).
	HostResolver = aspen.HostResolver
)

// ErrNodeNotFound is returned when a key names no node.
var ErrNodeNotFound = aspen.ErrNodeNotFound

// Host is the host node of a Core, the only node it resolves.
type Host struct{ Node node.Node }

var _ HostResolver = Host{}

// Host implements HostResolver.
func (h Host) Host() node.Node { return h.Node }

// HostKey implements HostResolver.
func (h Host) HostKey() node.Key { return h.Node.Key }

// Resolve implements HostResolver. It returns ErrNodeNotFound for any key other than
// the host's.
func (h Host) Resolve(key node.Key) (address.Address, error) {
	if key != h.Node.Key {
		return "", errors.Wrapf(ErrNodeNotFound, "node %d", key)
	}
	return h.Node.Address, nil
}
