// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";

import { useDispatch } from "@/arc/queries";
import { Synnax } from "@/synnax";
import { Diagram } from "@/vis/diagram";

// The "web " prefix is required: Chrome silently drops custom MIME types from
// the clipboard without it.
const MIME = "web application/synnax-arc+json";

export interface UseClipboardParams {
  key: arc.Key;
  selected?: string[];
  onPaste?: (newKeys: string[]) => void;
}

export const useClipboard = ({
  key,
  selected,
  onPaste,
}: UseClipboardParams): Diagram.UseClipboardReturn => {
  const { dispatch } = useDispatch();
  const client = Synnax.use();
  const adapter: Diagram.ClipboardAdapter<arc.graph.Node, arc.graph.Edge> = {
    mime: MIME,
    edgeKey: (edge) => edge.key,
    getSnapshot: () => {
      const cached = client?.arcs.getCached({ key });
      if (cached == null || cached.variant === "deleted") return null;
      const {
        graph: { nodes, edges, inputs },
      } = cached.data;
      return { nodes, edges, configs: inputs };
    },
    apply: ({ nodes, edges, newKeys }) => {
      const actions: arc.Action[] = [];
      for (const { node, config } of nodes) {
        actions.push(arc.setNode({ node }));
        if (config != null)
          actions.push(arc.setNodeInputs({ key: node.key, inputs: config }));
      }
      for (const { edge } of edges)
        actions.push(arc.addEdge({ edge: { ...edge, key: uuid.create() } }));
      dispatch({ key, actions });
      onPaste?.(newKeys);
    },
  };
  return Diagram.useClipboard({ adapter, selected });
};
