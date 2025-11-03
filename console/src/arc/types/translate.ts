// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { color, xy } from "@synnaxlabs/x";

import { type GraphState } from "@/arc/types";

export const translateGraphToConsole = (module: arc.Graph): GraphState => ({
  nodes: module.nodes.map((n) => ({
    key: n.key,
    position: n.position,
    selected: false,
    zIndex: 1,
  })),
  edges: module.edges.map((e) => ({
    id: `${e.source.node}-${e.target.node}`,
    key: `${e.source.node}-${e.target.node}`,
    source: e.source.node,
    target: e.target.node,
    sourceHandle: e.source.param,
    targetHandle: e.target.param,
    segments: [],
    color: color.ZERO,
    selected: false,
  })),
  props: Object.fromEntries(
    module.nodes.map((n) => [n.key, { key: n.type, ...n.config }]),
  ),
  viewport: {
    position: xy.ZERO,
    zoom: 1,
  },
  editable: false,
  fitViewOnResize: false,
});

export const translateGraphToServer = (arc: GraphState): arc.Graph => ({
  nodes: arc.nodes.map((n) => {
    const { key: type, ...config } = arc.props[n.key];
    return { key: n.key, type, config, position: n.position };
  }),
  edges: arc.edges.map((e) => ({
    source: { param: e.sourceHandle as string, node: e.source },
    target: { param: e.targetHandle as string, node: e.target },
  })),
});
