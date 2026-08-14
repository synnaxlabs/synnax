// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record, xy } from "@synnaxlabs/x";
import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  type ClipboardAdapter,
  type ClipboardEdge,
  type ClipboardNode,
  type PasteResult,
  type Snapshot,
  useClipboard,
} from "@/vis/diagram/clipboard";
import { type ClipboardHandler } from "@/vis/diagram/Diagram";

type ClipboardEvent = Parameters<ClipboardHandler>[0];

const MIME = "application/test-diagram";

interface Node extends ClipboardNode {
  type: string;
}

interface Edge extends ClipboardEdge {
  key: string;
}

const node = (key: string, position: xy.XY, type = "value"): Node => ({
  key,
  position,
  type,
});

const edge = (key: string, source: string, target: string): Edge => ({
  key,
  source: { node: source },
  target: { node: target },
});

const edgeKey = (e: Edge): string => e.key;

/**
 * fakeClipboardEvent builds a minimal ReactClipboardEvent backed by an in-memory
 * data store, recording whether preventDefault was called.
 */
const fakeClipboardEvent = (initial: Record<string, string> = {}) => {
  const store = new Map<string, string>(Object.entries(initial));
  const preventDefault = vi.fn();
  const event = {
    preventDefault,
    clipboardData: {
      getData: (mime: string) => store.get(mime) ?? "",
      setData: (mime: string, data: string) => {
        store.set(mime, data);
      },
    },
  } as unknown as ClipboardEvent;
  return { event, store, preventDefault };
};

const makeAdapter = (
  snapshot: Snapshot<Node, Edge> | null,
): ClipboardAdapter<Node, Edge> => ({
  mime: MIME,
  edgeKey,
  getSnapshot: () => snapshot,
  apply: vi.fn(),
});

const pasteResultOf = (
  adapter: ClipboardAdapter<Node, Edge>,
): PasteResult<Node, Edge> => vi.mocked(adapter.apply).mock.calls[0][0];

const renderClipboard = (adapter: ClipboardAdapter<Node, Edge>, selected?: string[]) =>
  renderHook(() => useClipboard({ adapter, selected })).result.current;

const readPayload = (store: Map<string, string>) => JSON.parse(store.get(MIME) ?? "");

describe("clipboard", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("onCopy", () => {
    it("should write the selected nodes and edges to the clipboard", () => {
      const snapshot: Snapshot<Node, Edge> = {
        nodes: [node("a", xy.construct(0, 0)), node("b", xy.construct(10, 20))],
        edges: [edge("e1", "a", "b")],
        configs: {},
      };
      const { onCopy } = renderClipboard(makeAdapter(snapshot), ["a", "b", "e1"]);
      const { event, store, preventDefault } = fakeClipboardEvent();
      onCopy(event, xy.ZERO);
      expect(preventDefault).toHaveBeenCalled();
      const payload = readPayload(store);
      expect(payload.version).toEqual(1);
      expect(payload.nodes.map((n: Node) => n.key)).toEqual(["a", "b"]);
      expect(payload.edges.map((e: Edge) => e.key)).toEqual(["e1"]);
    });

    it("should write a human-readable description to text/plain", () => {
      const snapshot: Snapshot<Node, Edge> = {
        nodes: [node("a", xy.ZERO), node("b", xy.ZERO)],
        edges: [edge("e1", "a", "b")],
        configs: {},
      };
      const { onCopy } = renderClipboard(makeAdapter(snapshot), ["a", "b", "e1"]);
      const { event, store } = fakeClipboardEvent();
      onCopy(event, xy.ZERO);
      expect(store.get("text/plain")).toEqual("2 nodes, 1 edge");
    });

    it("should singularize a single node and pluralize zero edges", () => {
      const snapshot: Snapshot<Node, Edge> = {
        nodes: [node("a", xy.ZERO)],
        edges: [],
        configs: {},
      };
      const { onCopy } = renderClipboard(makeAdapter(snapshot), ["a"]);
      const { event, store } = fakeClipboardEvent();
      onCopy(event, xy.ZERO);
      expect(store.get("text/plain")).toEqual("1 node, 0 edges");
    });

    it("should only copy items that are selected", () => {
      const snapshot: Snapshot<Node, Edge> = {
        nodes: [node("a", xy.ZERO), node("b", xy.ZERO), node("c", xy.ZERO)],
        edges: [edge("e1", "a", "b"), edge("e2", "b", "c")],
        configs: {},
      };
      const { onCopy } = renderClipboard(makeAdapter(snapshot), ["a", "b", "e1"]);
      const { event, store } = fakeClipboardEvent();
      onCopy(event, xy.ZERO);
      const payload = readPayload(store);
      expect(payload.nodes.map((n: Node) => n.key)).toEqual(["a", "b"]);
      expect(payload.edges.map((e: Edge) => e.key)).toEqual(["e1"]);
    });

    it("should include configs for the copied nodes and edges only", () => {
      const configs: Record<string, record.Unknown> = {
        a: { label: "A" },
        b: { label: "B" },
        e1: { stroke: "red" },
      };
      const snapshot: Snapshot<Node, Edge> = {
        nodes: [node("a", xy.ZERO), node("b", xy.ZERO)],
        edges: [edge("e1", "a", "b")],
        configs,
      };
      const { onCopy } = renderClipboard(makeAdapter(snapshot), ["a", "e1"]);
      const { event, store } = fakeClipboardEvent();
      onCopy(event, xy.ZERO);
      const payload = readPayload(store);
      expect(payload.configs).toEqual({ a: { label: "A" }, e1: { stroke: "red" } });
    });

    it("should compute the anchor as the centroid of the copied nodes", () => {
      const snapshot: Snapshot<Node, Edge> = {
        nodes: [node("a", xy.construct(0, 0)), node("b", xy.construct(10, 30))],
        edges: [],
        configs: {},
      };
      const { onCopy } = renderClipboard(makeAdapter(snapshot), ["a", "b"]);
      const { event, store } = fakeClipboardEvent();
      onCopy(event, xy.ZERO);
      const payload = readPayload(store);
      expect(payload.anchor).toEqual({ x: 5, y: 15 });
    });

    it("should defer to the browser when a text selection exists", () => {
      vi.spyOn(window, "getSelection").mockReturnValue({
        toString: () => "selected text",
      } as Selection);
      const snapshot: Snapshot<Node, Edge> = {
        nodes: [node("a", xy.ZERO)],
        edges: [],
        configs: {},
      };
      const { onCopy } = renderClipboard(makeAdapter(snapshot), ["a"]);
      const { event, store, preventDefault } = fakeClipboardEvent();
      onCopy(event, xy.ZERO);
      expect(preventDefault).not.toHaveBeenCalled();
      expect(store.has(MIME)).toBe(false);
    });

    it("should do nothing when the snapshot is unavailable", () => {
      const { onCopy } = renderClipboard(makeAdapter(null), ["a"]);
      const { event, store, preventDefault } = fakeClipboardEvent();
      onCopy(event, xy.ZERO);
      expect(preventDefault).not.toHaveBeenCalled();
      expect(store.has(MIME)).toBe(false);
    });

    it("should do nothing when nothing is selected", () => {
      const snapshot: Snapshot<Node, Edge> = {
        nodes: [node("a", xy.ZERO)],
        edges: [],
        configs: {},
      };
      const { onCopy } = renderClipboard(makeAdapter(snapshot));
      const { event, store, preventDefault } = fakeClipboardEvent();
      onCopy(event, xy.ZERO);
      expect(preventDefault).not.toHaveBeenCalled();
      expect(store.has(MIME)).toBe(false);
    });

    it("should do nothing when the selection matches no nodes or edges", () => {
      const snapshot: Snapshot<Node, Edge> = {
        nodes: [node("a", xy.ZERO)],
        edges: [edge("e1", "a", "a")],
        configs: {},
      };
      const { onCopy } = renderClipboard(makeAdapter(snapshot), ["missing"]);
      const { event, store, preventDefault } = fakeClipboardEvent();
      onCopy(event, xy.ZERO);
      expect(preventDefault).not.toHaveBeenCalled();
      expect(store.has(MIME)).toBe(false);
    });
  });

  describe("onPaste", () => {
    const writePayload = (
      store: Map<string, string>,
      payload: Record<string, unknown>,
    ) => store.set(MIME, JSON.stringify(payload));

    it("should remap node keys and offset positions to the cursor", () => {
      const adapter = makeAdapter(null);
      const { onPaste } = renderClipboard(adapter);
      const { event, store } = fakeClipboardEvent();
      writePayload(store, {
        version: 1,
        nodes: [node("a", xy.construct(0, 0)), node("b", xy.construct(10, 10))],
        edges: [],
        configs: {},
        anchor: { x: 5, y: 5 },
      });
      onPaste(event, xy.construct(105, 205));
      expect(adapter.apply).toHaveBeenCalledOnce();
      const result = pasteResultOf(adapter);
      expect(result.nodes).toHaveLength(2);
      result.nodes.forEach((pasted) => {
        expect(pasted.node.key).not.toEqual("a");
        expect(pasted.node.key).not.toEqual("b");
      });
      // offset = cursor - anchor = (100, 200)
      expect(result.nodes[0].node.position).toEqual({ x: 100, y: 200 });
      expect(result.nodes[1].node.position).toEqual({ x: 110, y: 210 });
      expect(result.newKeys).toEqual(result.nodes.map((n) => n.node.key));
    });

    it("should preserve other node fields while replacing the key", () => {
      const adapter = makeAdapter(null);
      const { onPaste } = renderClipboard(adapter);
      const { event, store } = fakeClipboardEvent();
      writePayload(store, {
        version: 1,
        nodes: [node("a", xy.ZERO, "valve")],
        edges: [],
        configs: {},
        anchor: xy.ZERO,
      });
      onPaste(event, xy.ZERO);
      const result = pasteResultOf(adapter);
      expect(result.nodes[0].node.type).toEqual("valve");
    });

    it("should remap edge endpoints onto the freshly generated node keys", () => {
      const adapter = makeAdapter(null);
      const { onPaste } = renderClipboard(adapter);
      const { event, store } = fakeClipboardEvent();
      writePayload(store, {
        version: 1,
        nodes: [node("a", xy.ZERO), node("b", xy.ZERO)],
        edges: [edge("e1", "a", "b")],
        configs: {},
        anchor: xy.ZERO,
      });
      onPaste(event, xy.ZERO);
      const result = pasteResultOf(adapter);
      const [keyA, keyB] = result.nodes.map((n) => n.node.key);
      expect(result.edges).toHaveLength(1);
      expect(result.edges[0].edge.source.node).toEqual(keyA);
      expect(result.edges[0].edge.target.node).toEqual(keyB);
    });

    it("should drop edges whose endpoints did not survive the paste", () => {
      const adapter = makeAdapter(null);
      const { onPaste } = renderClipboard(adapter);
      const { event, store } = fakeClipboardEvent();
      writePayload(store, {
        version: 1,
        nodes: [node("a", xy.ZERO)],
        edges: [edge("e1", "a", "ghost")],
        configs: {},
        anchor: xy.ZERO,
      });
      onPaste(event, xy.ZERO);
      const result = pasteResultOf(adapter);
      expect(result.nodes).toHaveLength(1);
      expect(result.edges).toHaveLength(0);
    });

    it("should attach copied configs to pasted nodes and edges", () => {
      const adapter = makeAdapter(null);
      const { onPaste } = renderClipboard(adapter);
      const { event, store } = fakeClipboardEvent();
      writePayload(store, {
        version: 1,
        nodes: [node("a", xy.ZERO), node("b", xy.ZERO)],
        edges: [edge("e1", "a", "b")],
        configs: { a: { label: "A" }, e1: { stroke: "red" } },
        anchor: xy.ZERO,
      });
      onPaste(event, xy.ZERO);
      const result = pasteResultOf(adapter);
      expect(result.nodes[0].config).toEqual({ label: "A" });
      expect(result.nodes[1].config).toBeUndefined();
      expect(result.edges[0].config).toEqual({ stroke: "red" });
    });

    it("should do nothing when there is no payload for the MIME type", () => {
      const adapter = makeAdapter(null);
      const { onPaste } = renderClipboard(adapter);
      const { event, preventDefault } = fakeClipboardEvent();
      onPaste(event, xy.ZERO);
      expect(adapter.apply).not.toHaveBeenCalled();
      expect(preventDefault).not.toHaveBeenCalled();
    });

    it("should do nothing when the payload is not valid JSON", () => {
      const adapter = makeAdapter(null);
      const { onPaste } = renderClipboard(adapter);
      const { event, preventDefault } = fakeClipboardEvent({ [MIME]: "not json {" });
      onPaste(event, xy.ZERO);
      expect(adapter.apply).not.toHaveBeenCalled();
      expect(preventDefault).not.toHaveBeenCalled();
    });

    it("should ignore payloads with a mismatched version", () => {
      const adapter = makeAdapter(null);
      const { onPaste } = renderClipboard(adapter);
      const { event, store, preventDefault } = fakeClipboardEvent();
      writePayload(store, {
        version: 999,
        nodes: [node("a", xy.ZERO)],
        edges: [],
        configs: {},
        anchor: xy.ZERO,
      });
      onPaste(event, xy.ZERO);
      expect(adapter.apply).not.toHaveBeenCalled();
      expect(preventDefault).not.toHaveBeenCalled();
    });

    it("should not apply when nothing survives the paste", () => {
      const adapter = makeAdapter(null);
      const { onPaste } = renderClipboard(adapter);
      const { event, store, preventDefault } = fakeClipboardEvent();
      writePayload(store, {
        version: 1,
        nodes: [],
        edges: [edge("e1", "a", "b")],
        configs: {},
        anchor: xy.ZERO,
      });
      onPaste(event, xy.ZERO);
      expect(adapter.apply).not.toHaveBeenCalled();
      expect(preventDefault).toHaveBeenCalled();
    });
  });

  describe("round trip", () => {
    it("should copy a selection and paste it back as fresh nodes and edges", () => {
      const snapshot: Snapshot<Node, Edge> = {
        nodes: [node("a", xy.construct(0, 0)), node("b", xy.construct(20, 0))],
        edges: [edge("e1", "a", "b")],
        configs: { a: { label: "A" }, e1: { stroke: "red" } },
      };
      const adapter = makeAdapter(snapshot);
      const { onCopy, onPaste } = renderClipboard(adapter, ["a", "b", "e1"]);
      const { event } = fakeClipboardEvent();
      onCopy(event, xy.ZERO);
      onPaste(event, xy.construct(100, 100));
      const result = pasteResultOf(adapter);
      expect(result.nodes).toHaveLength(2);
      expect(result.edges).toHaveLength(1);
      const originalKeys = new Set(["a", "b"]);
      const UUID_REGEX =
        /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
      result.nodes.forEach((n) => {
        expect(originalKeys.has(n.node.key)).toBe(false);
        expect(n.node.key).toMatch(UUID_REGEX);
      });
      const [keyA, keyB] = result.nodes.map((n) => n.node.key);
      expect(result.edges[0].edge.source.node).toEqual(keyA);
      expect(result.edges[0].edge.target.node).toEqual(keyB);
      // anchor is centroid (10, 0); offset to cursor (100, 100) -> (90, 100).
      expect(result.nodes[0].node.position).toEqual({ x: 90, y: 100 });
      expect(result.nodes[1].node.position).toEqual({ x: 110, y: 100 });
    });
  });
});
