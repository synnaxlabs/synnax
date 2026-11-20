// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  color,
  DisconnectedError,
  query,
  schematic,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { type record, uuid } from "@synnaxlabs/x";
import type { RefObject } from "react";

import { useSingleDispatch } from "@/schematic/queries";
import { useKey } from "@/schematic/Suspended";
import { Synnax } from "@/synnax";
import { Diagram } from "@/vis/diagram";

// The "web " prefix is required: Chrome silently drops custom MIME types from
// the clipboard without it.
const MIME = "web application/synnax-schematic+json";

const namedKey = (value: unknown): color.Key | null => {
  const ref = color.namedRefZ.safeParse(value);
  return ref.success ? ref.data.key : null;
};

/**
 * @returns a function that drops each named color the Core does not hold from a config,
 * such as one copied from another Core. A symbol paints its fallback in its place.
 */
const dropMissingColors = async (
  client: Client,
  configs: record.Unknown[],
): Promise<(config: record.Unknown) => record.Unknown> => {
  const keys = configs.flatMap((c) =>
    Object.values(c).flatMap((v) => namedKey(v) ?? []),
  );
  if (keys.length === 0) return (config) => config;
  const held = new Set(
    (await client.colors.retrieve({ where: { key: { in: keys } } })).map((n) => n.key),
  );
  return (config) =>
    Object.fromEntries(
      Object.entries(config).filter(([, v]) => {
        const key = namedKey(v);
        return key == null || held.has(key);
      }),
    );
};

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
  const handleError = Status.useErrorHandler();
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
    apply: ({ nodes, edges, remap }) =>
      handleError(async () => {
        if (client == null)
          throw new DisconnectedError("Cannot paste: no Core connected.");
        const drop = await dropMissingColors(client, [
          ...nodes.map(({ node }) => node.config),
          ...edges.map(({ edge }) => edge.config),
        ]);
        const actions: schematic.Action[] = [];
        // A pasted node keeps its group only when the group was pasted with it.
        for (const { node } of nodes) {
          const { parent, ...rest } = node;
          const pastedParent = parent == null ? undefined : remap[parent];
          const config = schematic.nodeConfigZ.parse(drop(rest.config));
          actions.push(
            schematic.setNode({
              node:
                pastedParent == null
                  ? { ...rest, config }
                  : { ...rest, config, parent: pastedParent },
            }),
          );
        }
        for (const { edge } of edges) {
          const config = schematic.edgeConfigZ.parse(drop(edge.config));
          actions.push(
            schematic.addEdge({ edge: { ...edge, config, key: uuid.create() } }),
          );
        }
        dispatch(actions);
        if (actions.length > 0) onPaste?.(Object.values(remap));
      }, "Failed to paste"),
    remove: ({ nodes, edges }) =>
      dispatch([
        ...nodes.map((key) => schematic.removeNode({ key })),
        ...edges.map((key) => schematic.removeEdge({ key })),
      ]),
  };
  return Diagram.useClipboard({ adapter, selected, onCut, container });
};
