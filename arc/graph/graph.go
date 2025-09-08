// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph

import (
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/spatial"
)

type (
	Stage = ir.Stage
	Edge  = ir.Edge
)
type Node struct {
	ir.Node
	Position spatial.XY `json:"position"`
}

type Viewport struct {
	Position spatial.XY `json:"position"`
	Zoom     float32    `json:"zoom"`
}

type Graph struct {
	Viewport Viewport `json:"viewport"`
	Stages   []Stage  `json:"stages"`
	Edges    []Edge   `json:"edges"`
}
