// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package v1 holds the frozen wire format for Console arc state at version 1.0.0.
// v1 replaced the flat ReactFlow edge form with nested Handle objects; nodes,
// props, text, and mode are unchanged from v0.
package v1

import (
	"github.com/synnaxlabs/arc/ir"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/legacy/v0"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
)

// Version is the ImEx schema version of arc state at this version. The Console
// stamped it on the wire as the semver string "1.0.0", which legacy.MigrateData
// decodes onto this numeric version.
const Version imex.Version = 1

// Data is the wire shape of a Console arc state at version 1.0.0.
type Data struct {
	Graph Graph   `json:"graph"`
	Text  v0.Text `json:"text"`
	Mode  string  `json:"mode"`
}

// Graph is the graph-mode program body with v1's nested-handle edges.
type Graph struct {
	Nodes []v0.Node                 `json:"nodes"`
	Edges []Edge                    `json:"edges"`
	Props map[string]map[string]any `json:"props"`
}

// Edge is the v1 edge form with nested Handle objects.
type Edge struct {
	Key    string    `json:"key"`
	Source ir.Handle `json:"source"`
	Target ir.Handle `json:"target"`
}
