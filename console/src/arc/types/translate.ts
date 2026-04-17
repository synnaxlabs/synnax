// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { xy } from "@synnaxlabs/x";

import { type GraphState } from "@/arc/types";

export const translateGraphToConsole = (module: arc.graph.Graph): GraphState => ({
  nodes: module.nodes.map((n) => ({
    key: n.key,
    position: n.position,
    zIndex: 1,
  })),
  edges: module.edges.map((e) => ({
    key: `${e.source.node}-${e.target.node}`,
    source: { node: e.source.node, param: e.source.param },
    target: { node: e.target.node, param: e.target.param },
  })),
  props: Object.fromEntries(
    module.nodes.map((n) => [n.key, { key: n.type, ...n.config }]),
  ),
  viewport: {
    position: xy.ZERO,
    zoom: 1,
  },
  selected: [],
  editable: false,
  fitViewOnResize: false,
});

export const translateGraphToServer = (arc: GraphState): arc.graph.Graph => ({
  nodes: arc.nodes.map((n) => {
    const { key: type, ...config } = arc.props[n.key];
    return { key: n.key, type, config, position: n.position };
  }),
  edges: arc.edges.map((e) => ({
    source: { node: e.source.node, param: e.source.param },
    target: { node: e.target.node, param: e.target.param },
  })),
  viewport: { position: xy.ZERO, zoom: 1 },
  functions: [],
});
