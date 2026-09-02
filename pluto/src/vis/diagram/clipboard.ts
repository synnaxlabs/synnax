// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record, uuid, xy } from "@synnaxlabs/x";
import { type ClipboardEvent, type RefObject, useCallback, useRef } from "react";

import { useSyncedRef } from "@/hooks";
import { type ClipboardHandler } from "@/vis/diagram/Diagram";

const VERSION = 1;

interface Payload<N, E> {
  version: number;
  nodes: N[];
  edges: E[];
  configs: Record<string, record.Unknown>;
  anchor: xy.XY;
}

/** ClipboardNode is the minimal shape the clipboard needs from a diagram node. */
export interface ClipboardNode {
  key: string;
  position: xy.XY;
}

/** ClipboardEdge is the minimal shape the clipboard needs from a diagram edge. */
export interface ClipboardEdge {
  source: { node: string };
  target: { node: string };
}

/** Snapshot is the current set of nodes, edges, and configs for a diagram. */
export interface Snapshot<N extends ClipboardNode, E extends ClipboardEdge> {
  nodes: N[];
  edges: E[];
  configs: Record<string, record.Unknown>;
}

/** PastedNode is a node remapped for paste, with its copied config attached. */
export interface PastedNode<N extends ClipboardNode> {
  node: N;
  config: record.Unknown | undefined;
}

/** PastedEdge is an edge remapped for paste, with its copied config attached. */
export interface PastedEdge<E extends ClipboardEdge> {
  edge: E;
  config: record.Unknown | undefined;
}

/**
 * PasteResult is the outcome of a paste, as plain data. Node keys are freshly
 * generated and positions are offset to the cursor; edge endpoints are remapped
 * onto the new node keys and edges whose endpoints did not survive are dropped.
 * It is up to the consumer to persist these values however it sees fit.
 */
export interface PasteResult<N extends ClipboardNode, E extends ClipboardEdge> {
  nodes: PastedNode<N>[];
  edges: PastedEdge<E>[];
  /** newKeys are the freshly generated node keys, e.g. to select what was pasted. */
  newKeys: string[];
}

/** SelectionKeys identifies clipboard items by key, split by kind. */
export interface SelectionKeys {
  nodes: string[];
  edges: string[];
}

/**
 * ClipboardAdapter binds the generic cut/copy/paste mechanics to a diagram domain.
 * It deals only in plain diagram data: the adapter reads a snapshot to copy and
 * receives a paste result to apply. The diagram has no knowledge of how that data
 * is stored or persisted.
 */
export interface ClipboardAdapter<N extends ClipboardNode, E extends ClipboardEdge> {
  /** mime is the clipboard MIME type used to read and write the payload. */
  mime: string;
  /**
   * edgeKey returns an edge's identity, used for selection membership and for
   * associating an edge with its copied config.
   */
  edgeKey: (edge: E) => string;
  /** getSnapshot returns the current diagram contents, or null if unavailable. */
  getSnapshot: () => Snapshot<N, E> | null;
  /** apply persists a paste result. Called only when at least one item pasted. */
  apply: (result: PasteResult<N, E>) => void;
  /** remove deletes the given items from the diagram. Called by cut after it copies. */
  remove: (keys: SelectionKeys) => void;
}

export interface UseClipboardParams<N extends ClipboardNode, E extends ClipboardEdge> {
  adapter: ClipboardAdapter<N, E>;
  selected?: string[];
  /** onCut receives the selection that survives a cut. */
  onCut?: (remaining: string[]) => void;
  container?: RefObject<HTMLDivElement | null>;
}

export interface UseClipboardReturn {
  onCopy: ClipboardHandler;
  onCut: ClipboardHandler;
  onPaste: ClipboardHandler;
  copy: () => void;
  cut: () => void;
  paste: () => void;
}

const describe = (nodeCount: number, edgeCount: number): string =>
  `${nodeCount} node${nodeCount === 1 ? "" : "s"}, ` +
  `${edgeCount} edge${edgeCount === 1 ? "" : "s"}`;

const centroid = (nodes: ClipboardNode[]): xy.XY => {
  if (nodes.length === 0) return xy.ZERO;
  return xy.scale(
    nodes.reduce((acc, n) => xy.translate(acc, n.position), xy.ZERO),
    1 / nodes.length,
  );
};

/**
 * useClipboard provides copy, cut, and paste handlers for a node/edge diagram. It
 * owns the clipboard mechanics (MIME plumbing, version gating, anchor geometry, key
 * remapping); the adapter supplies the diagram data and persists the result.
 */
export const useClipboard = <N extends ClipboardNode, E extends ClipboardEdge>({
  adapter,
  selected,
  onCut: onCutProp,
  container,
}: UseClipboardParams<N, E>): UseClipboardReturn => {
  const adapterRef = useSyncedRef(adapter);
  const selectedRef = useSyncedRef(selected ?? []);
  const onCutRef = useSyncedRef(onCutProp);
  const cutPendingRef = useRef(false);
  const lastCopiedRef = useRef<string | null>(null);

  // Writes the selection to the event's clipboard data. Returns the written keys,
  // or null when nothing was written.
  const writeSelection = useCallback(
    (e: ClipboardEvent<HTMLDivElement>): SelectionKeys | null => {
      // Defer to the browser if the user has a real text selection.
      const text = window.getSelection()?.toString();
      if (text != null && text.length > 0) return null;
      const { mime, edgeKey, getSnapshot } = adapterRef.current;
      const snapshot = getSnapshot();
      if (snapshot == null) return null;
      const sel = new Set(selectedRef.current);
      if (sel.size === 0) return null;
      const nodes = snapshot.nodes.filter((n) => sel.has(n.key));
      const edges = snapshot.edges.filter((edge) => sel.has(edgeKey(edge)));
      if (nodes.length === 0 && edges.length === 0) return null;
      const keys: SelectionKeys = {
        nodes: nodes.map((n) => n.key),
        edges: edges.map(edgeKey),
      };
      const configs: Record<string, record.Unknown> = {};
      for (const k of [...keys.nodes, ...keys.edges]) {
        const c = snapshot.configs[k];
        if (c != null) configs[k] = c;
      }
      const payload: Payload<N, E> = {
        version: VERSION,
        nodes,
        edges,
        configs,
        anchor: centroid(nodes),
      };
      e.preventDefault();
      const json = JSON.stringify(payload);
      e.clipboardData.setData(mime, json);
      e.clipboardData.setData("text/plain", describe(nodes.length, edges.length));
      lastCopiedRef.current = json;
      return keys;
    },
    [],
  );

  // Removes the written keys and reports the surviving selection.
  const removeWritten = useCallback((keys: SelectionKeys) => {
    adapterRef.current.remove(keys);
    const removed = new Set([...keys.nodes, ...keys.edges]);
    onCutRef.current?.(selectedRef.current.filter((k) => !removed.has(k)));
  }, []);

  const onCopy = useCallback<ClipboardHandler>(
    (e) => {
      const keys = writeSelection(e);
      if (cutPendingRef.current && keys != null) removeWritten(keys);
    },
    [writeSelection, removeWritten],
  );

  const onCut = useCallback<ClipboardHandler>(
    (e) => {
      const keys = writeSelection(e);
      if (keys != null) removeWritten(keys);
    },
    [writeSelection, removeWritten],
  );

  const exec = useCallback(
    (command: "copy" | "paste"): boolean => {
      const el = container?.current;
      if (el == null) return false;
      window.getSelection()?.removeAllRanges();
      el.focus({ preventScroll: true });
      // Monaco's clipboard menu actions call execCommand the same way. It is deprecated
      // with no replacement: WebKit's clipboard API rejects custom MIME types. Only
      // these menu triggers use it; the keyboard path fires native events.
      // eslint-disable-next-line @typescript-eslint/no-deprecated
      return document.execCommand(command);
    },
    [container],
  );

  const copy = useCallback(() => void exec("copy"), [exec]);

  const cut = useCallback(() => {
    // Native cut events only fire in editable DOM, so cut copies, then removes.
    cutPendingRef.current = true;
    try {
      exec("copy");
    } finally {
      cutPendingRef.current = false;
    }
  }, [exec]);

  const paste = useCallback(() => {
    // Chromium refuses execCommand("paste"). Replay the last copied payload as a
    // synthetic event so it flows through the normal handler and its cursor math.
    if (exec("paste")) return;
    const el = container?.current;
    const raw = lastCopiedRef.current;
    if (el == null || raw == null) return;
    const data = new DataTransfer();
    data.setData(adapterRef.current.mime, raw);
    el.dispatchEvent(
      new globalThis.ClipboardEvent("paste", { clipboardData: data, bubbles: true }),
    );
  }, [exec, container]);

  const onPaste = useCallback<ClipboardHandler>((e, cursor) => {
    const { mime, edgeKey, apply } = adapterRef.current;
    const raw = e.clipboardData.getData(mime);
    if (raw === "") return;
    let payload: Payload<N, E>;
    try {
      payload = JSON.parse(raw) as Payload<N, E>;
    } catch {
      return;
    }
    if (payload.version !== VERSION) return;
    e.preventDefault();
    const offset = xy.translation(payload.anchor, cursor);
    const remap: Record<string, string> = {};
    const nodes: PastedNode<N>[] = payload.nodes.map((node) => {
      const newKey = uuid.create();
      remap[node.key] = newKey;
      return {
        node: { ...node, key: newKey, position: xy.translate(node.position, offset) },
        config: payload.configs[node.key],
      };
    });
    const edges: PastedEdge<E>[] = [];
    for (const edge of payload.edges) {
      const source = remap[edge.source.node];
      const target = remap[edge.target.node];
      if (source == null || target == null) continue;
      edges.push({
        edge: {
          ...edge,
          source: { ...edge.source, node: source },
          target: { ...edge.target, node: target },
        },
        config: payload.configs[edgeKey(edge)],
      });
    }
    if (nodes.length === 0 && edges.length === 0) return;
    apply({ nodes, edges, newKeys: Object.values(remap) });
  }, []);

  return { onCopy, onCut, onPaste, copy, cut, paste };
};
