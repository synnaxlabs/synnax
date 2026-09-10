// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type Node, ReactFlowProvider, useStoreApi } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";

import { type Config } from "@/schematic/node/general/line/config";
import { defaultConfig } from "@/schematic/node/general/line/external";
import { Symbol } from "@/schematic/node/general/line/Symbol";
import { type NodeProps } from "@/schematic/node/spec";

type Store = ReturnType<typeof useStoreApi>;

const POSITION = { x: 100, y: 100 };
const NODES: Node[] = [{ id: "l1", position: POSITION, data: {} }];

const StoreProbe = ({ onStore }: { onStore: (s: Store) => void }): null => {
  onStore(useStoreApi());
  return null;
};

// A horizontal line from (100, 100) to (200, 100) in flow coordinates.
const renderSymbol = (props: Partial<NodeProps<Config>> = {}) => {
  const onConfigChange = vi.fn();
  let store!: Store;
  const { container } = render(
    <ReactFlowProvider defaultNodes={NODES}>
      <StoreProbe onStore={(s) => (store = s)} />
      <Symbol
        nodeKey="l1"
        selected
        onConfigChange={onConfigChange}
        position={POSITION}
        config={defaultConfig()}
        {...props}
      />
    </ReactFlowProvider>,
  );
  const handles = Array.from(
    container.querySelectorAll<HTMLElement>(".pluto-line__end"),
  );
  return { handles, onConfigChange, store };
};

const nodePosition = (store: Store): xy.XY | undefined =>
  store.getState().nodes.find((n) => n.id === "l1")?.position;

const POINTER = { pointerId: 1, button: 0, isPrimary: true };

// Moves are coalesced and delivered on the next animation frame.
const frame = async (): Promise<void> =>
  await new Promise((resolve) => requestAnimationFrame(() => resolve()));

const drag = async (el: HTMLElement, by: xy.XY, shiftKey = false): Promise<void> => {
  fireEvent.pointerDown(el, { ...POINTER, clientX: 0, clientY: 0 });
  fireEvent.pointerMove(window, { ...POINTER, shiftKey, clientX: by.x, clientY: by.y });
  await frame();
  fireEvent.pointerUp(window, { ...POINTER, clientX: by.x, clientY: by.y });
};

describe("Line.Symbol", () => {
  describe("handles", () => {
    it("should place a handle on each endpoint when selected", () => {
      const { handles } = renderSymbol();
      expect(handles).toHaveLength(2);
      expect(handles[0].style.left).toBe("0px");
      expect(handles[1].style.left).toBe("100px");
      expect(handles[1].style.top).toBe("0px");
    });

    it("should hide the handles when not selected", () => {
      expect(renderSymbol({ selected: false }).handles).toHaveLength(0);
    });

    it("should hide the handles when the node is not draggable", () => {
      expect(renderSymbol({ draggable: false }).handles).toHaveLength(0);
    });
  });

  describe("drag", () => {
    it("should move only the dragged endpoint", async () => {
      const { handles, onConfigChange, store } = renderSymbol();
      await drag(handles[1], { x: 20, y: 30 });
      expect(onConfigChange).toHaveBeenLastCalledWith({
        start: { x: 0, y: 0 },
        end: { x: 120, y: 30 },
      });
      expect(nodePosition(store)).toEqual(POSITION);
    });

    it("should re-anchor the node when an endpoint passes the origin", async () => {
      const { handles, onConfigChange, store } = renderSymbol();
      await drag(handles[0], { x: -20, y: -30 });
      expect(nodePosition(store)).toEqual({ x: 80, y: 70 });
      expect(onConfigChange).toHaveBeenLastCalledWith({
        start: { x: 0, y: 0 },
        end: { x: 120, y: 30 },
      });
    });

    it("should snap to horizontal with shift when the x delta dominates", async () => {
      const { handles, onConfigChange } = renderSymbol();
      await drag(handles[1], { x: 20, y: 6 }, true);
      expect(onConfigChange).toHaveBeenLastCalledWith({
        start: { x: 0, y: 0 },
        end: { x: 120, y: 0 },
      });
    });

    it("should snap to vertical with shift when the y delta dominates", async () => {
      const { handles, onConfigChange } = renderSymbol();
      await drag(handles[1], { x: -94, y: 30 }, true);
      expect(onConfigChange).toHaveBeenLastCalledWith({
        start: { x: 0, y: 0 },
        end: { x: 0, y: 30 },
      });
    });

    it("should divide the pointer delta by the viewport zoom", async () => {
      const { handles, onConfigChange, store } = renderSymbol();
      store.setState({ transform: [0, 0, 2] });
      await drag(handles[1], { x: 40, y: 20 });
      expect(onConfigChange).toHaveBeenLastCalledWith({
        start: { x: 0, y: 0 },
        end: { x: 120, y: 10 },
      });
    });
  });
});
