// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { type record, uuid, xy } from "@synnaxlabs/x";
import { useCallback } from "react";

import { type FluxSubStore, useDispatch } from "@/arc/queries";
import { edgeKey } from "@/arc/translate";
import { Flux } from "@/flux";
import { useSyncedRef } from "@/hooks";
import { type DiagramClipboardHandler } from "@/vis/diagram/Diagram";

// The "web " prefix is required: Chrome silently drops custom MIME types from
// the clipboard without it.
const MIME = "web application/synnax-arc+json";
const VERSION = 1;

interface Payload {
  version: number;
  nodes: arc.graph.Node[];
  configs: Record<string, record.Unknown>;
  edges: arc.ir.Edge[];
  anchor: xy.XY;
}

const describe = (p: Payload): string => {
  const n = p.nodes.length;
  const e = p.edges.length;
  return `${n} node${n === 1 ? "" : "s"}, ${e} edge${e === 1 ? "" : "s"}`;
};

const centroid = (nodes: arc.graph.Node[]): xy.XY => {
  if (nodes.length === 0) return xy.ZERO;
  return xy.scale(
    nodes.reduce((acc, n) => xy.translate(acc, n.position), xy.ZERO),
    1 / nodes.length,
  );
};

export interface UseClipboardArgs {
  key: arc.Key;
  selected?: string[];
  onPaste?: (newKeys: string[]) => void;
}

export interface UseClipboardReturn {
  onCopy: DiagramClipboardHandler;
  onPaste: DiagramClipboardHandler;
}

export const useClipboard = ({
  key,
  selected,
  onPaste,
}: UseClipboardArgs): UseClipboardReturn => {
  const { dispatch } = useDispatch();
  const store = Flux.useStore<FluxSubStore>();
  const selectedRef = useSyncedRef(selected ?? []);

  const handleCopy = useCallback<DiagramClipboardHandler>(
    (e) => {
      const text = window.getSelection()?.toString();
      if (text != null && text.length > 0) return;
      const a = store.arcs.get(key);
      if (a == null) return;
      const sel = new Set(selectedRef.current);
      if (sel.size === 0) return;
      const nodes = a.graph.nodes.filter((n) => sel.has(n.key));
      const edges = a.graph.edges.filter((edge) =>
        sel.has(edgeKey(edge.source, edge.target)),
      );
      if (nodes.length === 0 && edges.length === 0) return;
      const configs: Record<string, record.Unknown> = {};
      for (const n of nodes) {
        const c = a.graph.configs[n.key];
        if (c != null) configs[n.key] = c;
      }
      const payload: Payload = {
        version: VERSION,
        nodes,
        configs,
        edges,
        anchor: centroid(nodes),
      };
      e.preventDefault();
      e.clipboardData.setData(MIME, JSON.stringify(payload));
      e.clipboardData.setData("text/plain", describe(payload));
    },
    [key, store],
  );

  const handlePaste = useCallback<DiagramClipboardHandler>(
    (e, cursor) => {
      const raw = e.clipboardData.getData(MIME);
      if (raw === "") return;
      let payload: Payload;
      try {
        payload = JSON.parse(raw) as Payload;
      } catch {
        return;
      }
      if (payload.version !== VERSION) return;
      e.preventDefault();
      const offset = xy.translation(payload.anchor, cursor);
      const remap: Record<string, string> = {};
      const actions: arc.Action[] = [];
      for (const node of payload.nodes) {
        const newKey = uuid.create();
        remap[node.key] = newKey;
        actions.push(
          arc.setNode({
            node: {
              ...node,
              key: newKey,
              position: xy.translate(node.position, offset),
            },
          }),
        );
        const config = payload.configs[node.key];
        if (config != null) actions.push(arc.setNodeConfig({ key: newKey, config }));
      }
      for (const edge of payload.edges) {
        const src = remap[edge.source.node];
        const tgt = remap[edge.target.node];
        if (src == null || tgt == null) continue;
        actions.push(
          arc.addEdge({
            edge: {
              ...edge,
              source: { ...edge.source, node: src },
              target: { ...edge.target, node: tgt },
            },
          }),
        );
      }
      dispatch({ key, actions });
      if (actions.length > 0) onPaste?.(Object.values(remap));
    },
    [key, dispatch, onPaste],
  );

  return { onCopy: handleCopy, onPaste: handlePaste };
};
