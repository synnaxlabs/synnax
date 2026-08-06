// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v6 holds the frozen wire format for the Console schematic export at version
// 6. Unlike v0-v5 this is not a persisted state: from v0.56 the Console retrieves the
// typed schematic from the Core and writes it out directly, stamping its own version.
// The body is therefore the current Schematic shape in camelCase, which is why v6 has
// no Migrate into the storage chain — only the ImEx decode path reads it.
package v6

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/x/encoding/msgpack"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
)

// Version is the version the Console stamped on this format.
const Version imex.Version = 6

// Node is a schematic node as the Console serializes it.
type Node struct {
	// Key is the node's unique key.
	Key string `json:"key" msgpack:"key"`
	// Position is the node position on the canvas.
	Position spatial.XY `json:"position" msgpack:"position"`
	// ZIndex is the node stacking order.
	ZIndex int16 `json:"zIndex" msgpack:"zIndex"`
}

// Handle is one end of an edge, naming the node and the parameter it binds.
type Handle struct {
	// Node is the key of the node this end attaches to.
	Node string `json:"node" msgpack:"node"`
	// Param is the parameter on that node.
	Param string `json:"param" msgpack:"param"`
}

// Edge connects two node handles.
type Edge struct {
	// Key is the edge's unique key.
	Key string `json:"key" msgpack:"key"`
	// Source is the originating handle.
	Source Handle `json:"source" msgpack:"source"`
	// Target is the terminating handle.
	Target Handle `json:"target" msgpack:"target"`
}

// Data is the Console schematic export at version 6. Configs values are opaque user
// JSON and pass through as-is.
type Data struct {
	// Snapshot marks the schematic as a range snapshot.
	Snapshot bool `json:"snapshot" msgpack:"snapshot"`
	// Nodes are the schematic nodes.
	Nodes []Node `json:"nodes" msgpack:"nodes"`
	// Edges are the schematic edges.
	Edges []Edge `json:"edges" msgpack:"edges"`
	// Configs holds per-symbol configuration keyed by node key.
	Configs map[string]msgpack.EncodedJSON `json:"configs" msgpack:"configs"`
}
