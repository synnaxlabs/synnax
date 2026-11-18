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

import { schematic } from "@synnaxlabs/client";
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
import { userEvent } from "vitest/browser";

import { CSS } from "@/css";
import { Grid } from "@/schematic/node/common/grid";
import { Value } from "@/schematic/node/general/value";
import { Diagram } from "@/vis/diagram";

const LEFT = "1";
const RIGHT = "2";
const UNITS = "mbar";
const WIDE_UNITS = "mbar per square second";
const RIGHT_HANDLE = `.react-flow__node[data-id="a"] [data-handleid="${RIGHT}"]`;
const NODE_DRAG_HANDLE = `.react-flow__node[data-id="a"] .${Grid.DRAG_HANDLE_CLASS}`;
const RING_PROBE_RATIO = 0.8;
const DOT_SIZE = 8;
const TARGET_SIZE = 13;
const TRANSPARENT = "rgba(0, 0, 0, 0)";
const ZOOMS = [0.5, 0.75, 1, 1.5];
const COLOR = color.construct("#00ff00");

type Store = ReturnType<typeof useStoreApi>;
type ValueNode = Node<{ units: string }, "custom">;

const ValueNodeRenderer = ({ data }: NodeProps<ValueNode>): ReactElement => (
  <div className={Grid.DRAG_HANDLE_CLASS}>
    <Value.spec.Preview
      {...schematic.valueNodeConfigZ.parse({ variant: "value", strokeColor: COLOR })}
      units={data.units}
    />
  </div>
);

const NODE_TYPES = { custom: ValueNodeRenderer };

const NODES: ValueNode[] = [
  { id: "a", type: "custom", position: { x: 50, y: 50 }, data: { units: UNITS } },
  { id: "b", type: "custom", position: { x: 500, y: 50 }, data: { units: UNITS } },
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
      flow.updateNodeData("a", { units: WIDE_UNITS });
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
  const endpoint = Diagram.resolveEndpoint(internals, handle);
  if (endpoint == null) throw new Error(`handle ${handle} not found`);
  return endpoint.position;
};

const handleWidth = (store: Store, nodeKey: string, handle: string): number => {
  const { handleBounds } = internalNode(store, nodeKey).internals;
  const bounds = handleBounds?.source?.find(({ id }) => id === handle);
  if (bounds == null) throw new Error(`handle ${handle} not found`);
  return bounds.width;
};

const screenSize = (handle: HTMLElement, pseudo: string, zoom: number): number => {
  const style = getComputedStyle(handle, pseudo);
  const outer =
    parseFloat(style.width) +
    parseFloat(style.borderLeftWidth) +
    parseFloat(style.borderRightWidth);
  return outer * parseFloat(style.scale) * zoom;
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

// The symbol can resize after mount, and React Flow measures it again a frame later.
const expectOnRightEdge = async (store: Store, nodeKey: string): Promise<void> =>
  await vi.waitFor(() => {
    const edge = rightEdge(store, nodeKey);
    const point = anchor(store, nodeKey, RIGHT);
    expect(point.x - handleWidth(store, nodeKey, RIGHT) / 2).toBeCloseTo(edge.x, 0);
    expect(point.y).toBeCloseTo(edge.y, 0);
  });

describe("value symbol connection points", () => {
  describe.each([
    { editable: false, mode: "viewing" },
    { editable: true, mode: "editing" },
  ])("while $mode", ({ editable }) => {
    it.each(ZOOMS)("should sit on the outer right edge at zoom %s", async (zoom) => {
      const { store } = await renderFlow(zoom, editable);
      await expectOnRightEdge(store, "a");
    });

    it.each(ZOOMS)("should follow the widened edge at zoom %s", async (zoom) => {
      const { store, widen } = await renderFlow(zoom, editable);
      await widen();
      await expectOnRightEdge(store, "a");
    });
  });

  it.each(ZOOMS)("should take the pointer on its ring at zoom %s", async (zoom) => {
    await renderFlow(zoom, true);
    const { left, right, top, bottom } = queryRect(RIGHT_HANDLE);
    const x = (left + right) / 2;
    const y = (top + bottom) / 2;
    const handle = queryElement(RIGHT_HANDLE);
    const ringWidth = parseFloat(getComputedStyle(handle, "::before").width);
    expect(ringWidth).toBeGreaterThan(0);
    const probe = (ringWidth / 2) * RING_PROBE_RATIO;
    expect(document.elementFromPoint(x, y - probe)).toBe(handle);
    expect(document.elementFromPoint(x + probe, y)).toBe(handle);
  });

  it.each(ZOOMS)(
    "should draw a dot smaller than its target on hover at zoom %s",
    async (zoom) => {
      await renderFlow(zoom, true);
      await userEvent.hover(queryElement(NODE_DRAG_HANDLE));
      const handle = queryElement(RIGHT_HANDLE);
      const dot = getComputedStyle(handle, "::after");
      await vi.waitFor(() => expect(dot.backgroundColor).not.toBe(TRANSPARENT));
      expect(screenSize(handle, "::after", zoom)).toBeCloseTo(DOT_SIZE, 0);
      expect(screenSize(handle, "::before", zoom)).toBeCloseTo(TARGET_SIZE, 0);
    },
  );

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
    await expectOnRightEdge(store, "a");
  });
});
