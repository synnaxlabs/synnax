// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@xyflow/react/dist/base.css";
import "@/schematic/Schematic.css";

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { color, type xy } from "@synnaxlabs/x";
import {
  type InternalNode,
  type Node,
  type NodeProps,
  ReactFlow,
  type ReactFlowInstance,
  ReactFlowProvider,
  useReactFlow,
  useStoreApi,
} from "@xyflow/react";
import { type ReactElement, useEffect } from "react";
import { describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { DRAG_HANDLE_CLASS } from "@/schematic/node/common/grid/Grid";
import { Value } from "@/schematic/node/general/value/Primitive";
import { resolveEndpoint } from "@/vis/diagram/util";

const LEFT = "1";
const RIGHT = "2";
const WIDENED = 160;
const RIGHT_HANDLE = `.react-flow__node[data-id="a"] [data-handleid="${RIGHT}"]`;
const RING_RADIUS = 4.5;
const ZOOMS = [0.5, 0.75, 1, 1.5];
const COLOR = color.construct("#00ff00");

type Store = ReturnType<typeof useStoreApi>;
type ValueNode = Node<{ width: number }, "custom">;

const ValueNodeRenderer = ({ data }: NodeProps<ValueNode>): ReactElement => (
  <div className={DRAG_HANDLE_CLASS}>
    <Value
      strokeColor={COLOR}
      orientation="left"
      units="mbar"
      inlineSize={data.width}
      dimensions={{ height: 27, width: data.width }}
    />
  </div>
);

const NODE_TYPES = { custom: ValueNodeRenderer };

const NODES: ValueNode[] = [
  { id: "a", type: "custom", position: { x: 50, y: 50 }, data: { width: 70 } },
  { id: "b", type: "custom", position: { x: 500, y: 50 }, data: { width: 70 } },
];

interface Context {
  store: Store;
  flow: ReactFlowInstance<ValueNode>;
}

interface FlowProps {
  zoom: number;
  editable: boolean;
  contextRef: { current: Context | null };
}

const Tap = ({ contextRef }: Pick<FlowProps, "contextRef">): null => {
  const store = useStoreApi();
  const flow = useReactFlow<ValueNode>();
  useEffect(() => {
    contextRef.current = { store, flow };
  }, [store, flow]);
  return null;
};

const SCHEMATIC_CLASS = CSS.B("schematic");
const ZOOM_VARIABLE = CSS.variable("diagram-zoom");

const Flow = ({ zoom, editable, contextRef }: FlowProps): ReactElement => (
  <div
    className={CSS.cls(SCHEMATIC_CLASS, CSS.editable(editable))}
    style={{ width: 900, height: 500, [ZOOM_VARIABLE]: zoom }}
  >
    <ReactFlowProvider>
      <ReactFlow
        defaultNodes={NODES}
        nodeTypes={NODE_TYPES}
        defaultViewport={{ x: 0, y: 0, zoom }}
        minZoom={0.1}
        maxZoom={4}
      >
        <Tap contextRef={contextRef} />
      </ReactFlow>
    </ReactFlowProvider>
  </div>
);

interface Rendered {
  store: Store;
  widen: () => Promise<void>;
  rezoom: (zoom: number) => Promise<void>;
}

const renderFlow = async (zoom: number, editable: boolean): Promise<Rendered> => {
  const contextRef: FlowProps["contextRef"] = { current: null };
  await renderThemed(<Flow zoom={zoom} editable={editable} contextRef={contextRef} />);
  const { store, flow } = await vi.waitFor(() => {
    const context = contextRef.current;
    if (context == null) throw new Error("flow not mounted");
    NODES.forEach(({ id }) => {
      if (internalNode(context.store, id).internals.handleBounds == null)
        throw new Error(`node ${id} not measured`);
    });
    return context;
  });
  return {
    store,
    rezoom: async (zoom) => {
      queryElement(`.${SCHEMATIC_CLASS}`).style.setProperty(ZOOM_VARIABLE, `${zoom}`);
      void flow.setViewport({ x: 0, y: 0, zoom });
      await vi.waitFor(() => expect(zoomOf(store)).toBe(zoom));
    },
    widen: async () => {
      const before = anchor(store, "a", RIGHT).x;
      flow.updateNodeData("a", { width: WIDENED });
      await vi.waitFor(() =>
        expect(anchor(store, "a", RIGHT).x).toBeGreaterThan(before),
      );
    },
  };
};

const queryElement = (selector: string): HTMLElement => {
  const element = document.querySelector<HTMLElement>(selector);
  if (element == null) throw new Error(`${selector} not found`);
  return element;
};

const queryRect = (selector: string): DOMRect =>
  queryElement(selector).getBoundingClientRect();

const nodeBox = (nodeKey: string): DOMRect =>
  queryRect(`.react-flow__node[data-id="${nodeKey}"]`);

const outerBox = (nodeKey: string): DOMRect =>
  queryRect(`.react-flow__node[data-id="${nodeKey}"] .${CSS.B("value")}`);

const internalNode = (store: Store, nodeKey: string): InternalNode => {
  const node = store.getState().nodeLookup.get(nodeKey);
  if (node == null) throw new Error(`node ${nodeKey} not found`);
  return node;
};

/** @returns the point a pipe attaches to, in flow coordinates. */
const anchor = (store: Store, nodeKey: string, handle: string): xy.XY => {
  const { internals } = internalNode(store, nodeKey);
  const endpoint = resolveEndpoint(internals, handle);
  if (endpoint == null) throw new Error(`handle ${handle} not found`);
  return endpoint.position;
};

const handleWidth = (store: Store, nodeKey: string, handle: string): number => {
  const { handleBounds } = internalNode(store, nodeKey).internals;
  const bounds = handleBounds?.source?.find(({ id }) => id === handle);
  if (bounds == null) throw new Error(`handle ${handle} not found`);
  return bounds.width;
};

const zoomOf = (store: Store): number => store.getState().transform[2];

const toFlow = (store: Store, nodeKey: string, screen: xy.XY): xy.XY => {
  const { left, top } = nodeBox(nodeKey);
  const { x, y } = internalNode(store, nodeKey).internals.positionAbsolute;
  const zoom = zoomOf(store);
  return { x: x + (screen.x - left) / zoom, y: y + (screen.y - top) / zoom };
};

const rightEdge = (store: Store, nodeKey: string): xy.XY => {
  const { right, top, bottom } = outerBox(nodeKey);
  return toFlow(store, nodeKey, { x: right, y: (top + bottom) / 2 });
};

const expectOnRightEdge = (store: Store, nodeKey: string): void => {
  const edge = rightEdge(store, nodeKey);
  const point = anchor(store, nodeKey, RIGHT);
  expect(point.x - handleWidth(store, nodeKey, RIGHT) / 2).toBeCloseTo(edge.x, 0);
  expect(point.y).toBeCloseTo(edge.y, 0);
};

describe("value symbol connection points", () => {
  describe.each([
    { editable: false, mode: "viewing" },
    { editable: true, mode: "editing" },
  ])("while $mode", ({ editable }) => {
    it.each(ZOOMS)("should sit on the outer right edge at zoom %s", async (zoom) => {
      const { store } = await renderFlow(zoom, editable);
      expectOnRightEdge(store, "a");
    });

    it.each(ZOOMS)("should follow the widened edge at zoom %s", async (zoom) => {
      const { store, widen } = await renderFlow(zoom, editable);
      await widen();
      expectOnRightEdge(store, "a");
    });
  });

  it.each(ZOOMS)("should take the pointer on its ring at zoom %s", async (zoom) => {
    await renderFlow(zoom, true);
    const { left, right, top, bottom } = queryRect(RIGHT_HANDLE);
    const x = (left + right) / 2;
    const y = (top + bottom) / 2;
    const handle = queryElement(RIGHT_HANDLE);
    expect(document.elementFromPoint(x, y - RING_RADIUS + 1)).toBe(handle);
    expect(document.elementFromPoint(x + RING_RADIUS - 1, y)).toBe(handle);
  });

  it("should keep a level pipe straight when the value widens at a new zoom", async () => {
    const { store, widen, rezoom } = await renderFlow(1, false);
    await rezoom(0.5);
    await widen();
    expect(anchor(store, "a", RIGHT).y).toBeCloseTo(anchor(store, "b", LEFT).y, 1);
  });

  it("should not move a connection point when the zoom changes", async () => {
    const { store, rezoom } = await renderFlow(0.5, false);
    const expected = anchor(store, "a", RIGHT);
    await rezoom(1.5);
    expect(anchor(store, "a", RIGHT)).toEqual(expected);
    expectOnRightEdge(store, "a");
  });
});
