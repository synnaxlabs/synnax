// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package node defines the node, the unit that leases channels and other resources. A
// Core runs one node, the host. The service that serves it as a resource lives in
// pkg/service/node.
package node

import (
	"strconv"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/types"
)

// Key is a 12-bit unsigned integer that identifies a node. Channel keys embed the key
// of the node that leases them.
type Key types.Uint12

const (
	// KeyBootstrapper is the key of the host node of a Core.
	KeyBootstrapper Key = 1
	// KeyFree is the key of no node. Resources that no node leases, such as free
	// virtual channels, carry it.
	KeyFree = Key(math.MaxUint12)
)

// Free returns true if the key is KeyFree.
func (k Key) Free() bool { return k == KeyFree }

// IsBootstrapper returns true if the key is KeyBootstrapper.
func (k Key) IsBootstrapper() bool { return k == KeyBootstrapper }

// String implements fmt.Stringer.
func (k Key) String() string { return strconv.Itoa(int(k)) }

// Node is a node of a Core, reached at Address.
type Node struct {
	// Address is the network address of the node.
	Address address.Address
	// Key identifies the node.
	Key Key
}
