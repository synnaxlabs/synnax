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
import * as Groups from "@/schematic/groups";
import { type FluxSubStore, useDispatch } from "@/schematic/queries";
import { type DiagramClipboardHandler } from "@/vis/diagram/Diagram";

// The "web " prefix is required: Chrome silently drops custom MIME types from
// the clipboard without it.
const MIME = "web application/synnax-schematic+json";
const VERSION = 1;

interface Payload {
  version: number;
  nodes: schematic.Node[];
  edges: schematic.Edge[];
  configs: Record<string, record.Unknown>;
  anchor: xy.XY;
}

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

interface BuiltPayload {
  payload: Payload;
  expandedKeys: Set<string>;
}

/** Computes the clipboard payload for the current selection, expanded across
 * groups. */
const buildPayload = (
  schem: schematic.Schematic,
  selectedKeys: readonly string[],
): BuiltPayload | null => {
  if (selectedKeys.length === 0) return null;
  const expanded = Groups.expandSelectionToGroups(
    selectedKeys,
    schem.nodes,
    schem.configs,
  );
  const expandedSet = new Set(expanded);
  const ns = schem.nodes.filter((n) => expandedSet.has(n.key));
  const includedEdges = schem.edges.filter((e) => expandedSet.has(e.key));
  if (ns.length === 0 && includedEdges.length === 0) return null;
  const configs: Record<string, record.Unknown> = {};
  for (const k of [
    ...ns.map((n) => n.key),
    ...includedEdges.map((edge) => edge.key),
  ]) {
    const c = schem.configs[k];
    if (c != null) configs[k] = c as record.Unknown;
  }
  return {
    payload: {
      version: VERSION,
      nodes: ns,
      edges: includedEdges,
      configs,
      anchor: centroid(ns),
    },
    expandedKeys: expandedSet,
  };
};

export interface UseClipboardArgs {
  key: schematic.Key;
  selected?: string[];
  onPaste?: (newKeys: string[]) => void;
}

export interface UseClipboardReturn {
  onCopy: DiagramClipboardHandler;
  onCut: DiagramClipboardHandler;
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
      // Defer to the browser if the user has a real text selection.
      const text = window.getSelection()?.toString();
      if (text != null && text.length > 0) return;
      const schem = store.schematics.get(key);
      if (schem == null) return;
      const built = buildPayload(schem, selectedRef.current);
      if (built == null) return;
      e.preventDefault();
      e.clipboardData.setData(MIME, JSON.stringify(built.payload));
      e.clipboardData.setData("text/plain", describe(built.payload));
    },
    [key, store],
  );

  const handleCut = useCallback<DiagramClipboardHandler>(
    (e) => {
      const text = window.getSelection()?.toString();
      if (text != null && text.length > 0) return;
      const schem = store.schematics.get(key);
      if (schem == null) return;
      const built = buildPayload(schem, selectedRef.current);
      if (built == null) return;
      e.preventDefault();
      e.clipboardData.setData(MIME, JSON.stringify(built.payload));
      e.clipboardData.setData("text/plain", describe(built.payload));
      const actions: schematic.Action[] = [];
      for (const edge of built.payload.edges)
        actions.push(schematic.removeEdge({ key: edge.key }));
      for (const k of built.expandedKeys)
        actions.push(schematic.removeNode({ key: k }));
      if (actions.length > 0) dispatch({ key, actions });
    },
    [key, store, dispatch],
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
      const actions: schematic.Action[] = [];
      for (const node of payload.nodes) {
        const newKey = uuid.create();
        remap[node.key] = newKey;
      }
      for (const node of payload.nodes) {
        const remapped = Groups.remapGroupId(node, remap);
        actions.push(
          schematic.setNode({
            node: {
              ...remapped,
              key: remap[node.key],
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

  return { onCopy: handleCopy, onCut: handleCut, onPaste: handlePaste };
};
