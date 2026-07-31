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

import { useSingleDispatch } from "@/schematic/queries";
import { useKey } from "@/schematic/Suspended";
import { Synnax } from "@/synnax";
import { Diagram } from "@/vis/diagram";

// The "web " prefix is required: Chrome silently drops custom MIME types from
// the clipboard without it.
const MIME = "web application/synnax-schematic+json";

export interface UseClipboardParams {
  selected?: string[];
  onPaste?: (newKeys: string[]) => void;
}

export const useClipboard = ({
  selected,
  onPaste,
}: UseClipboardParams): Diagram.UseClipboardReturn => {
  const key = useKey();
  const dispatch = useSingleDispatch();
  const client = Synnax.use();
  const adapter: Diagram.ClipboardAdapter<schematic.Node, schematic.Edge> = {
    mime: MIME,
    edgeKey: (edge) => edge.key,
    getSnapshot: () => {
      const cached = client?.schematics.getCached({ key });
      if (!query.isLive(cached)) return null;
      const { nodes, edges, configs } = cached;
      return { nodes, edges, configs };
    },
    apply: ({ nodes, edges, newKeys }) => {
      const actions: schematic.Action[] = [];
      for (const { node, config } of nodes)
        actions.push(schematic.setNode({ node, config }));
      for (const { edge, config } of edges) {
        const edgeKey = uuid.create();
        actions.push(schematic.addEdge({ edge: { ...edge, key: edgeKey } }));
        if (config != null) actions.push(schematic.setConfig({ key: edgeKey, config }));
      }
      dispatch(actions);
      if (actions.length > 0) onPaste?.(newKeys);
    },
  };
  return Diagram.useClipboard({ adapter, selected });
};
