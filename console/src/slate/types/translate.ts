import { type slate } from "@synnaxlabs/client";
import { color, xy } from "@synnaxlabs/x";

import { type State } from "@/slate/types";

export const translateSlateForward = (slate: slate.Slate): State => ({
  key: slate.key,
  nodes: slate.graph.nodes.map((n) => ({
    key: n.key,
    position: (n.data.position as xy.XY) ?? xy.ZERO,
    selected: false,
    zIndex: 1,
  })),
  edges: slate.graph.edges.map((e) => ({
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
    slate.graph.nodes.map((n) => [
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

export const translateSlateBackward = (slate: State): slate.Slate => ({
  key: slate.key,
  graph: {
    nodes: slate.nodes.map((n) => ({
      key: n.key,
      type: slate.props[n.key].key,
      data: slate.props[n.key],
    })),
    edges: slate.edges.map((e) => ({
      source: { key: e.sourceHandle as string, node: e.source },
      sink: { key: e.targetHandle as string, node: e.target },
    })),
  },
});
