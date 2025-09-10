// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/text"
)

type (
	IR       = ir.IR
	Stage    = ir.Stage
	Node     = ir.Node
	Edge     = ir.Edge
	Function = ir.Function
	Graph    = graph.Graph
	Text     = text.Text
)

type Module struct {
	ir.IR
	WASM []byte
}
