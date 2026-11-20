// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, schematic } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import type { RefObject } from "react";

import { useSingleDispatch } from "@/schematic/queries";
import { useKey } from "@/schematic/Suspended";
import { Synnax } from "@/synnax";
import { Diagram } from "@/vis/diagram";

// The "web " prefix is required: Chrome silently drops custom MIME types from
// the clipboard without it.
const MIME = "web application/synnax-schematic+json";

export interface UseClipboardParams {
  selected?: string[];
  onCut?: (remaining: string[]) => void;
  onPaste?: (newKeys: string[]) => void;
  container?: RefObject<HTMLDivElement | null>;
}

export const useClipboard = ({
  selected,
  onCut,
  onPaste,
  container,
}: UseClipboardParams): Diagram.UseClipboardReturn => {
  const key = useKey();
  const dispatch = useSingleDispatch();
  const client = Synnax.use();
  // Nodes and edges carry their own configs, so the clipboard holds no others.
  const adapter: Diagram.ClipboardAdapter<schematic.Node, schematic.Edge, never> = {
    mime: MIME,
    edgeKey: (edge) => edge.key,
    getSnapshot: () => {
      const cached = client?.schematics.watch(key).getCached();
      if (!query.isLive(cached)) return null;
      const { nodes, edges } = cached;
      return { nodes, edges, configs: {} };
    },
    apply: ({ nodes, edges, remap }) => {
      const actions: schematic.Action[] = [];
      // A pasted node keeps its group only when the group was pasted with it.
      for (const { node } of nodes) {
        const { parent, ...rest } = node;
        const pastedParent = parent == null ? undefined : remap[parent];
        actions.push(
          schematic.setNode({
            node: pastedParent == null ? rest : { ...rest, parent: pastedParent },
          }),
        );
      }
      for (const { edge } of edges)
        actions.push(schematic.addEdge({ edge: { ...edge, key: uuid.create() } }));
      dispatch(actions);
      if (actions.length > 0) onPaste?.(Object.values(remap));
    },
    remove: ({ nodes, edges }) =>
      dispatch([
        ...nodes.map((key) => schematic.removeNode({ key })),
        ...edges.map((key) => schematic.removeEdge({ key })),
      ]),
  };
  return Diagram.useClipboard({ adapter, selected, onCut, container });
};
