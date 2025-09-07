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

export const raiseGraph = (module: arc.Module): GraphState => ({
  nodes: module.nodes.map((n) => ({
    key: n.key,
    position: (n.config.position as xy.XY) ?? xy.ZERO,
    selected: false,
    zIndex: 1,
  })),
  edges: module.edges.map((e) => ({
    id: `${e.source.key}-${e.sink.key}`,
    key: `${e.source.key}-${e.sink.key}`,
    source: e.source.node,
    target: e.sink.node,
    sourceHandle: e.source.key,
    targetHandle: e.sink.key,
    segments: [],
    color: color.ZERO,
    selected: false,
  })),
  props: Object.fromEntries(
    module.nodes.map((n) => [n.key, { key: n.key, ...n.config }]),
  ),
  viewport: {
    position: xy.ZERO,
    zoom: 1,
  },
  editable: true,
  fitViewOnResize: false,
});

export const lowerGraph = (arc: GraphState): arc.Module => ({
  nodes: arc.nodes.map((n) => ({
    key: n.key,
    type: arc.props[n.key].key,
    config: arc.props[n.key],
  })),
  edges: arc.edges.map((e) => ({
    source: { key: e.sourceHandle as string, node: e.source },
    sink: { key: e.targetHandle as string, node: e.target },
  })),
});
