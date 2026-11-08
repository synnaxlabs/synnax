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

import "github.com/synnaxlabs/aspen"

type (
	// Key is a 12-bit unsigned integer that identifies a node. Channel keys embed the
	// key of the node that leases them.
	Key = aspen.NodeKey
	// Node is a node of a Core, reached at its address.
	Node = aspen.Node
)

const (
	// KeyBootstrapper is the key of the host node of a Core.
	KeyBootstrapper = aspen.NodeKeyBootstrapper
	// KeyFree is the key of no node. Resources that no node leases, such as free
	// virtual channels, carry it.
	KeyFree = aspen.NodeKeyFree
)
