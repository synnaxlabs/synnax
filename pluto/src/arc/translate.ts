// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";

import { type Diagram } from "@/vis/diagram";

// EDGE_KEY_SEP joins the four handle fields of an edge into a single diagram edge
// key. Arc edges carry no key on the wire, so the diagram key is derived from the
// endpoints and parsed back into handles when a gesture targets the edge. Node
// keys (generated ids) and parameter names (identifiers) never contain a colon.
const EDGE_KEY_SEP = "::";

// edgeKey derives the diagram edge key for the given handles.
export const edgeKey = (source: arc.ir.Handle, target: arc.ir.Handle): string =>
  [source.node, source.param, target.node, target.param].join(EDGE_KEY_SEP);

// parseEdgeKey recovers the source and target handles from a diagram edge key.
export const parseEdgeKey = (
  key: string,
): { source: arc.ir.Handle; target: arc.ir.Handle } => {
  const [sn, sp, tn, tp] = key.split(EDGE_KEY_SEP);
  return { source: { node: sn, param: sp }, target: { node: tn, param: tp } };
};

// nodeProps returns the renderer props for a server graph node: the function type
// under the reserved `key` field, merged with the node's config values. Graph
// nodes are otherwise consumed as diagram nodes directly, since graph.Node is a
// structural superset of Diagram.Node.
export const nodeProps = (n: arc.graph.Node): record.Unknown => ({
  key: n.type,
  ...n.config,
});

// edgeToDiagram converts a server graph edge to a keyed diagram edge.
export const edgeToDiagram = (e: arc.ir.Edge): Diagram.Edge => ({
  key: edgeKey(e.source, e.target),
  source: { node: e.source.node, param: e.source.param },
  target: { node: e.target.node, param: e.target.param },
});

// edgesToDiagram converts every server graph edge to a keyed diagram edge.
export const edgesToDiagram = (edges: arc.ir.Edge[]): Diagram.Edge[] =>
  edges.map(edgeToDiagram);
