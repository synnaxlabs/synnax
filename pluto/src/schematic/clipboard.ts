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
import { type RefObject } from "react";

import { Group } from "@/schematic/group";
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
  const adapter: Diagram.ClipboardAdapter<schematic.Node, schematic.Edge> = {
    mime: MIME,
    edgeKey: (edge) => edge.key,
    getSnapshot: () => {
      const cached = client?.schematics.getCached(key);
      if (!query.isLive(cached)) return null;
      const { nodes, edges, configs } = cached;
      return { nodes, edges, configs };
    },
    apply: ({ nodes, edges, remap }) => {
      const actions: schematic.Action[] = [];
      for (const { node, config } of nodes)
        actions.push(
          schematic.setNode({ node, config: Group.remapMembers(config, remap) }),
        );
      for (const { edge, config } of edges) {
        const edgeKey = uuid.create();
        actions.push(schematic.addEdge({ edge: { ...edge, key: edgeKey } }));
        if (config != null) actions.push(schematic.setConfig({ key: edgeKey, config }));
      }
      dispatch(actions);
      if (actions.length > 0) onPaste?.(Object.values(remap));
    },
    remove: ({ nodes, edges }) => {
      // removeNode does not cascade, so cut unselected connected edges too;
      // they would otherwise persist as invisible dangling edges.
      const cut = new Set(nodes);
      const cached = client?.schematics.getCached(key);
      const connected = query.isLive(cached)
        ? cached.edges
            .filter((e) => cut.has(e.source.node) || cut.has(e.target.node))
            .map((e) => e.key)
        : [];
      dispatch([
        ...nodes.map((key) => schematic.removeNode({ key })),
        ...[...new Set([...edges, ...connected])].map((key) =>
          schematic.removeEdge({ key }),
        ),
      ]);
    },
  };
  return Diagram.useClipboard({ adapter, selected, onCut, container });
};
