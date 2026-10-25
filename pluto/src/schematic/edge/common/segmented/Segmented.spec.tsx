// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render } from "@testing-library/react";
import { type Node, ReactFlowProvider, useStoreApi } from "@xyflow/react";
import { Profiler, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Jumps } from "@/schematic/edge/common/jumps";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { diagram } from "@/vis/diagram/aether";

type Store = ReturnType<typeof useStoreApi>;

const SIZE = { width: 40, height: 40 };

// The target sits behind the source, so the edge routes around the source's box.
const NODES: Node[] = [
  { id: "a", position: { x: 0, y: 0 }, measured: SIZE, data: {} },
  { id: "b", position: { x: -200, y: 100 }, measured: SIZE, data: {} },
  { id: "c", position: { x: 400, y: 400 }, measured: SIZE, data: {} },
];

const updateNode = (store: Store, id: string, change: Partial<Node>): void => {
  const { nodes, setNodes } = store.getState();
  setNodes(nodes.map((n) => (n.id === id ? { ...n, ...change } : n)));
};

const Line = ({ points, crossings }: Segmented.PathProps): ReactElement => (
  <path d={Path.rounded(points, crossings)} />
);

const spec = Segmented.createSpec("electric", "Electric", Line);

const StoreProbe = ({ onStore }: { onStore: (s: Store) => void }): null => {
  onStore(useStoreApi());
  return null;
};

const renderEdge = () => {
  let store!: Store;
  let renders = 0;
  const jumps = Jumps.createStore();
  const ui = (edgeKey: string): ReactElement => (
    <ReactFlowProvider defaultNodes={NODES}>
      <StoreProbe onStore={(s) => (store = s)} />
      <Jumps.Context value={jumps}>
        <svg>
          <Profiler id="edge" onRender={() => renders++}>
            <spec.Edge
              key={edgeKey}
              edgeKey={edgeKey}
              source={diagram.createEndpoint(40, 20, "right")}
              target={diagram.createEndpoint(-160, 120, "right")}
              sourceNode="a"
              targetNode="b"
              selected={false}
              config={spec.defaultConfig()}
              onChange={vi.fn()}
            />
          </Profiler>
        </svg>
      </Jumps.Context>
    </ReactFlowProvider>
  );
  const { container, rerender } = render(ui("first"));
  const path = (): string | null =>
    container.querySelector("path")?.getAttribute("d") ?? null;
  return {
    store: () => store,
    path,
    renders: () => renders,
    remount: () => rerender(ui("second")),
  };
};

describe("Segmented", () => {
  it("should draw the edge for the current zoom after the viewport zooms", () => {
    const { store, path, remount } = renderEdge();
    const atStart = path();
    act(() => store().setState({ transform: [0, 0, 2] }));
    const zoomed = path();
    remount();
    expect(zoomed).toEqual(path());
    expect(zoomed).not.toEqual(atStart);
  });

  it("should route around a node's new box after the node resizes", () => {
    const { store, path, remount } = renderEdge();
    const atStart = path();
    act(() => updateNode(store(), "a", { measured: { width: 40, height: 120 } }));
    const resized = path();
    remount();
    expect(resized).toEqual(path());
    expect(resized).not.toEqual(atStart);
  });

  it("should not re-render when a node it does not connect moves", () => {
    const { store, renders } = renderEdge();
    const baseline = renders();
    act(() => updateNode(store(), "c", { position: { x: 500, y: 500 } }));
    expect(renders()).toBe(baseline);
  });

  it("should not re-render when a node it connects keeps the same box", () => {
    const { store, renders } = renderEdge();
    const baseline = renders();
    act(() => updateNode(store(), "a", { measured: { ...SIZE }, selected: true }));
    expect(renders()).toBe(baseline);
  });
});
