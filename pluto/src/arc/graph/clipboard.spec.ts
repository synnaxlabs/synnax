// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useClipboard } from "@/arc/graph/clipboard";
import { useDispatch } from "@/arc/queries";
import { Flux } from "@/flux";
import { Diagram } from "@/vis/diagram";

vi.mock("@/arc/queries", () => ({ useDispatch: vi.fn() }));
vi.mock("@/flux", () => ({ Flux: { useStore: vi.fn() } }));
vi.mock("@/vis/diagram", () => ({ Diagram: { useClipboard: vi.fn() } }));

type Adapter = Diagram.ClipboardAdapter<arc.graph.Node, arc.ir.Edge>;

const KEY = "arc-1";

const handle = (node: string, param: string): arc.ir.Handle => ({ node, param });

const node = (key: string, x = 0, y = 0): arc.graph.Node => ({
  key,
  position: { x, y },
});

const edge = (source: arc.ir.Handle, target: arc.ir.Handle): arc.ir.Edge => ({
  source,
  target,
  kind: arc.ir.EdgeKind.continuous,
});

const dispatch = vi.fn();
const get = vi.fn();
const onPaste = vi.fn();

/**
 * captureAdapter renders the hook and returns the adapter it hands to the
 * generic Diagram clipboard, which is where all of this module's logic lives.
 */
const captureAdapter = (
  args: Partial<Parameters<typeof useClipboard>[0]> = {},
): Adapter => {
  renderHook(() => useClipboard({ key: KEY, onPaste, ...args }));
  return vi.mocked(Diagram.useClipboard).mock.calls[0][0].adapter as unknown as Adapter;
};

describe("arc graph clipboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useDispatch).mockReturnValue({
      dispatch,
    } as unknown as ReturnType<typeof useDispatch>);
    vi.mocked(Flux.useStore).mockReturnValue({
      arcs: { get },
    } as unknown as ReturnType<typeof Flux.useStore>);
    vi.mocked(Diagram.useClipboard).mockReturnValue({
      onCopy: vi.fn(),
      onPaste: vi.fn(),
    });
  });

  it("should forward the selected keys to the generic clipboard", () => {
    captureAdapter({ selected: ["a", "b"] });
    expect(vi.mocked(Diagram.useClipboard).mock.calls[0][0].selected).toEqual([
      "a",
      "b",
    ]);
  });

  describe("edgeKey", () => {
    it("should derive an edge's identity from its endpoints", () => {
      const adapter = captureAdapter();
      const source = handle("a", "out");
      const target = handle("b", "in");
      expect(adapter.edgeKey(edge(source, target))).toEqual(
        arc.ir.edgeKey(source, target),
      );
    });
  });

  describe("getSnapshot", () => {
    it("should return null when the arc is not in the store", () => {
      get.mockReturnValue(undefined);
      const adapter = captureAdapter();
      expect(adapter.getSnapshot()).toBeNull();
      expect(get).toHaveBeenCalledWith(KEY);
    });

    it("should project the arc's graph into a snapshot", () => {
      const nodes = [node("a"), node("b")];
      const edges = [edge(handle("a", "out"), handle("b", "in"))];
      const configs = { a: { label: "A" } };
      get.mockReturnValue({ graph: { nodes, edges, configs } });
      const adapter = captureAdapter();
      expect(adapter.getSnapshot()).toEqual({ nodes, edges, configs });
    });
  });

  describe("apply", () => {
    it("should dispatch a set_node action for each pasted node", () => {
      const adapter = captureAdapter();
      const a = node("a", 1, 2);
      const b = node("b", 3, 4);
      adapter.apply({
        nodes: [
          { node: a, config: undefined },
          { node: b, config: undefined },
        ],
        edges: [],
        newKeys: ["a", "b"],
      });
      expect(dispatch).toHaveBeenCalledWith({
        key: KEY,
        actions: [arc.setNode({ node: a }), arc.setNode({ node: b })],
      });
    });

    it("should dispatch a set_node_config action when a node carries a config", () => {
      const adapter = captureAdapter();
      const a = node("a");
      adapter.apply({
        nodes: [{ node: a, config: { label: "A" } }],
        edges: [],
        newKeys: ["a"],
      });
      expect(dispatch).toHaveBeenCalledWith({
        key: KEY,
        actions: [
          arc.setNode({ node: a }),
          arc.setNodeConfig({ key: "a", config: { label: "A" } }),
        ],
      });
    });

    it("should dispatch an add_edge action for each pasted edge", () => {
      const adapter = captureAdapter();
      const e = edge(handle("a", "out"), handle("b", "in"));
      adapter.apply({
        nodes: [],
        edges: [{ edge: e, config: undefined }],
        newKeys: [],
      });
      expect(dispatch).toHaveBeenCalledWith({
        key: KEY,
        actions: [arc.addEdge({ edge: e })],
      });
    });

    it("should order node actions before edge actions", () => {
      const adapter = captureAdapter();
      const a = node("a");
      const b = node("b");
      const e = edge(handle("a", "out"), handle("b", "in"));
      adapter.apply({
        nodes: [
          { node: a, config: { label: "A" } },
          { node: b, config: undefined },
        ],
        edges: [{ edge: e, config: undefined }],
        newKeys: ["a", "b"],
      });
      expect(dispatch).toHaveBeenCalledWith({
        key: KEY,
        actions: [
          arc.setNode({ node: a }),
          arc.setNodeConfig({ key: "a", config: { label: "A" } }),
          arc.setNode({ node: b }),
          arc.addEdge({ edge: e }),
        ],
      });
    });

    it("should notify the onPaste callback with the freshly generated keys", () => {
      const adapter = captureAdapter();
      adapter.apply({
        nodes: [{ node: node("a"), config: undefined }],
        edges: [],
        newKeys: ["a", "b"],
      });
      expect(onPaste).toHaveBeenCalledWith(["a", "b"]);
    });

    it("should not throw when no onPaste callback is provided", () => {
      const adapter = captureAdapter({ onPaste: undefined });
      expect(() =>
        adapter.apply({
          nodes: [{ node: node("a"), config: undefined }],
          edges: [],
          newKeys: ["a"],
        }),
      ).not.toThrow();
      expect(dispatch).toHaveBeenCalledOnce();
    });
  });
});
