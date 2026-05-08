// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type record, uuid, xy } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Flux } from "@/flux";
import { useSyncedRef } from "@/hooks";
import { type FluxSubStore, useDispatch } from "@/schematic/queries";

// The "web " prefix is required: Chrome silently drops custom MIME types from
// the clipboard without it.
const MIME = "web application/synnax-schematic+json";
const VERSION = 1;

export interface Payload {
  version: number;
  nodes: schematic.Node[];
  edges: schematic.Edge[];
  configs: Record<string, record.Unknown>;
  anchor: xy.XY;
}

export const write = async (
  nodes: schematic.Node[],
  edges: schematic.Edge[],
  configs: Record<string, record.Unknown>,
  anchor: xy.XY,
): Promise<void> => {
  const payload: Payload = { version: VERSION, nodes, edges, configs, anchor };
  await navigator.clipboard.write([
    new ClipboardItem({
      [MIME]: new Blob([JSON.stringify(payload)], { type: MIME }),
      "text/plain": new Blob([describe(payload)], { type: "text/plain" }),
    }),
  ]);
};

export const read = async (): Promise<Payload | null> => {
  let items: ClipboardItem[];
  try {
    items = await navigator.clipboard.read();
  } catch {
    return null;
  }
  for (const item of items) {
    if (!item.types.includes(MIME)) continue;
    try {
      const blob = await item.getType(MIME);
      const parsed = JSON.parse(await blob.text()) as Payload;
      if (parsed.version !== VERSION) return null;
      return parsed;
    } catch {
      return null;
    }
  }
  return null;
};

const describe = (p: Payload): string => {
  const n = p.nodes.length;
  const e = p.edges.length;
  return `${n} node${n === 1 ? "" : "s"}, ${e} edge${e === 1 ? "" : "s"}`;
};

const centroid = (nodes: schematic.Node[]): xy.XY => {
  if (nodes.length === 0) return xy.ZERO;
  return xy.scale(
    nodes.reduce((acc, n) => xy.translate(acc, n.position), xy.ZERO),
    1 / nodes.length,
  );
};

export interface UseClipboardArgs {
  key: schematic.Key;
  selected?: string[];
  onPaste?: (newKeys: string[]) => void;
}

export interface UseClipboardReturn {
  copy: () => void;
  paste: (target: xy.XY) => Promise<void>;
}

export const useClipboard = ({
  key,
  selected,
  onPaste,
}: UseClipboardArgs): UseClipboardReturn => {
  const { update: dispatch } = useDispatch();
  const store = Flux.useStore<FluxSubStore>();
  const selectedRef = useSyncedRef(selected ?? []);

  const copy = useCallback(() => {
    const schem = store.schematics.get(key);
    if (schem == null) return;
    const sel = new Set(selectedRef.current);
    if (sel.size === 0) return;
    const ns = schem.nodes.filter((n) => sel.has(n.key));
    const es = schem.edges.filter((e) => sel.has(e.key));
    if (ns.length === 0 && es.length === 0) return;
    const configs: Record<string, record.Unknown> = {};
    for (const k of [...ns.map((n) => n.key), ...es.map((e) => e.key)]) {
      const c = schem.configs[k];
      if (c != null) configs[k] = c as record.Unknown;
    }
    void write(ns, es, configs, centroid(ns));
  }, [key, store]);

  const paste = useCallback(
    async (target: xy.XY): Promise<void> => {
      const payload = await read();
      if (payload == null) return;
      const offset = xy.translation(payload.anchor, target);
      const remap: Record<string, string> = {};
      const actions: schematic.Action[] = [];
      for (const node of payload.nodes) {
        const newKey = uuid.create();
        remap[node.key] = newKey;
        actions.push(
          schematic.setNode({
            node: {
              ...node,
              key: newKey,
              position: xy.translate(node.position, offset),
            },
            config: payload.configs[node.key],
          }),
        );
      }
      for (const edge of payload.edges) {
        const src = remap[edge.source.node];
        const tgt = remap[edge.target.node];
        if (src == null || tgt == null) continue;
        const newKey = uuid.create();
        actions.push(
          schematic.addEdge({
            edge: {
              ...edge,
              key: newKey,
              source: { ...edge.source, node: src },
              target: { ...edge.target, node: tgt },
            },
          }),
        );
        const cfg = payload.configs[edge.key];
        if (cfg != null)
          actions.push(schematic.setConfig({ key: newKey, config: cfg }));
      }
      if (actions.length === 0) return;
      dispatch({ key, actions });
      onPaste?.(Object.values(remap));
    },
    [key, dispatch, onPaste],
  );

  return { copy, paste };
};
