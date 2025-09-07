import { type arc } from "@synnaxlabs/client";
import { color, xy } from "@synnaxlabs/x";

import { type State } from "@/arc/types";

export const translateSlateForward = (arc: arc.Arc): State => ({
  key: arc.key,
  nodes: arc.graph.nodes.map((n) => ({
    key: n.key,
    position: (n.data.position as xy.XY) ?? xy.ZERO,
    selected: false,
    zIndex: 1,
  })),
  edges: arc.graph.edges.map((e) => ({
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
    arc.graph.nodes.map((n) => [
      n.key,
      {
        key: n.key,
        ...n.data,
      },
    ]),
  ),
  viewport: {
    position: xy.ZERO,
    zoom: 1,
  },
  fitViewOnResize: false,
  remoteCreated: false,
  editable: true,
  version: "0.0.0",
});

export const translateSlateBackward = (arc: State): arc.Arc => ({
  key: arc.key,
  graph: {
    nodes: arc.nodes.map((n) => ({
      key: n.key,
      type: arc.props[n.key].key,
      config: arc.props[n.key],
    })),
    edges: arc.edges.map((e) => ({
      source: { key: e.sourceHandle as string, node: e.source },
      sink: { key: e.targetHandle as string, node: e.target },
    })),
  },
});
