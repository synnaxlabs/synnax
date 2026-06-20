// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";

import { type FluxSubStore, useDispatch } from "@/arc/queries";
import { Flux } from "@/flux";
import { Diagram } from "@/vis/diagram";

// The "web " prefix is required: Chrome silently drops custom MIME types from
// the clipboard without it.
const MIME = "web application/synnax-arc+json";

export interface UseClipboardArgs {
  key: arc.Key;
  selected?: string[];
  onPaste?: (newKeys: string[]) => void;
}

export const useClipboard = ({
  key,
  selected,
  onPaste,
}: UseClipboardArgs): Diagram.UseClipboardReturn => {
  const { dispatch } = useDispatch();
  const store = Flux.useStore<FluxSubStore>();
  const adapter: Diagram.ClipboardAdapter<arc.graph.Node, arc.ir.Edge> = {
    mime: MIME,
    edgeKey: (edge) => arc.ir.edgeKey(edge.source, edge.target),
    getSnapshot: () => {
      const a = store.arcs.get(key);
      if (a == null) return null;
      return { nodes: a.graph.nodes, edges: a.graph.edges, configs: a.graph.configs };
    },
    apply: ({ nodes, edges, newKeys }) => {
      const actions: arc.Action[] = [];
      for (const { node, config } of nodes) {
        actions.push(arc.setNode({ node }));
        if (config != null) actions.push(arc.setNodeConfig({ key: node.key, config }));
      }
      for (const { edge } of edges) actions.push(arc.addEdge({ edge }));
      dispatch({ key, actions });
      onPaste?.(newKeys);
    },
  };
  return Diagram.useClipboard({ adapter, selected });
};
