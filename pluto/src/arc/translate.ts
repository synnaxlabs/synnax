// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";

import { type Diagram } from "@/vis/diagram";

// edgeToDiagram converts a server graph edge to a keyed diagram edge.
export const edgeToDiagram = (e: arc.ir.Edge): Diagram.Edge => ({
  key: arc.ir.edgeKey(e.source, e.target),
  source: { node: e.source.node, param: e.source.param },
  target: { node: e.target.node, param: e.target.param },
});

// edgesToDiagram converts every server graph edge to a keyed diagram edge.
export const edgesToDiagram = (edges: arc.ir.Edge[]): Diagram.Edge[] =>
  edges.map(edgeToDiagram);
