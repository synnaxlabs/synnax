// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook, waitFor } from "@testing-library/react";
import {
  type Node,
  ReactFlow,
  type ReactFlowProps,
  ReactFlowProvider,
  useNodesInitialized,
  useReactFlow,
  useUpdateNodeInternals,
} from "@xyflow/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { useFitView, useInitialFitView } from "@/vis/diagram/useFitView";

const CONTAINER = { width: 1000, height: 500 };

const Custom = (): ReactElement => (
  <div>
    <span data-testid="label">label</span>
  </div>
);
const NODE_TYPES = { custom: Custom };
const NODES: Node[] = [
  { id: "a", type: "custom", position: { x: 100, y: 100 }, data: {} },
];

const domRect = (x: number, y: number, width: number, height: number): DOMRect =>
  ({
    x,
    y,
    width,
    height,
    left: x,
    top: y,
    right: x + width,
    bottom: y + height,
  }) as DOMRect;

const rect = (
  el: Element,
  x: number,
  y: number,
  width: number,
  height: number,
): void => {
  el.getBoundingClientRect = () => domRect(x, y, width, height);
};

const createWrapper = (props: ReactFlowProps = {}) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <ReactFlowProvider>
      <ReactFlow defaultNodes={NODES} nodeTypes={NODE_TYPES} maxZoom={1} {...props} />
      {children}
    </ReactFlowProvider>
  );
  return Wrapper;
};

const renderFitView = (props?: ReactFlowProps) =>
  renderHook(() => ({ fitView: useFitView(), flow: useReactFlow() }), {
    wrapper: createWrapper(props),
  });

// The node renders at flow (100, 100) as a 50x50 box and its label hangs 80px to the
// left of it, so the union spans (20, 100) to (150, 150) and centers at (85, 125).
const layoutNode = (): void => {
  const node = document.querySelector('[data-id="a"]');
  if (node == null) throw new Error("node not rendered");
  rect(node, 100, 100, 50, 50);
  rect(node.querySelector('[data-testid="label"]')!, 20, 110, 70, 20);
};

describe("Diagram.useFitView", () => {
  // jsdom lays nothing out; React Flow sizes its container from these.
  beforeAll(() => {
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(
      CONTAINER.width,
    );
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(
      CONTAINER.height,
    );
  });
  afterAll(() => vi.restoreAllMocks());

  it("centers the union of a node and its label in the container", () => {
    const { result } = renderFitView();
    layoutNode();
    act(() => result.current.fitView({ padding: 0 }));
    expect(result.current.flow.getViewport()).toEqual({ x: 415, y: 125, zoom: 1 });
  });

  it("measures the nodes through the current viewport", () => {
    const { result } = renderFitView();
    act(() => void result.current.flow.setViewport({ x: 40, y: 10, zoom: 2 }));
    // The same flow-space layout as above, seen at double zoom and panned.
    const node = document.querySelector('[data-id="a"]')!;
    rect(node, 240, 210, 100, 100);
    rect(node.querySelector('[data-testid="label"]')!, 80, 230, 140, 40);
    act(() => result.current.fitView({ padding: 0 }));
    expect(result.current.flow.getViewport()).toEqual({ x: 415, y: 125, zoom: 1 });
  });

  it("does not snap the bounds to the grid", () => {
    const { result } = renderFitView({ snapToGrid: true, snapGrid: [100, 100] });
    layoutNode();
    act(() => result.current.fitView({ padding: 0 }));
    expect(result.current.flow.getViewport()).toEqual({ x: 415, y: 125, zoom: 1 });
  });

  it("leaves the viewport alone when no node is rendered", () => {
    const { result } = renderFitView({ defaultNodes: [] });
    act(() => result.current.fitView({ padding: 0 }));
    expect(result.current.flow.getViewport()).toEqual({ x: 0, y: 0, zoom: 1 });
  });
});

const FITTED = { x: 415, y: 125, zoom: 1 };
const PADDING_NONE = { padding: 0 };

describe("Diagram.useInitialFitView", () => {
  // The fit fires on measurement, so rects must resolve before rendering. React
  // Flow's measurement also needs DOMMatrixReadOnly, which jsdom lacks.
  beforeAll(() => {
    vi.stubGlobal(
      "DOMMatrixReadOnly",
      class {
        m22: number;
        constructor(transform?: string) {
          this.m22 = Number(transform?.match(/scale\(([\d.]+)\)/)?.[1] ?? 1);
        }
      },
    );
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(
      CONTAINER.width,
    );
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(
      CONTAINER.height,
    );
    vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (
      this: Element,
    ) {
      if (this.matches('[data-id="a"]')) return domRect(100, 100, 50, 50);
      if (this.matches('[data-testid="label"]')) return domRect(20, 110, 70, 20);
      return domRect(0, 0, 0, 0);
    });
  });
  afterAll(() => vi.restoreAllMocks());

  // jsdom fires no ResizeObserver, so each test measures the node through React
  // Flow's public useUpdateNodeInternals to make the nodes initialize.
  const renderInitialFit = (enabled: boolean) =>
    renderHook(
      (props: { enabled: boolean }) => {
        useInitialFitView(props.enabled, PADDING_NONE);
        return {
          flow: useReactFlow(),
          measure: useUpdateNodeInternals(),
          initialized: useNodesInitialized(),
        };
      },
      { wrapper: createWrapper(), initialProps: { enabled } },
    );

  it("fits once the nodes initialize", async () => {
    const { result } = renderInitialFit(true);
    act(() => result.current.measure("a"));
    await waitFor(() => expect(result.current.initialized).toBe(true));
    await waitFor(() => expect(result.current.flow.getViewport()).toEqual(FITTED));
  });

  it("does not fit again while it stays enabled", async () => {
    const { result, rerender } = renderInitialFit(true);
    act(() => result.current.measure("a"));
    await waitFor(() => expect(result.current.flow.getViewport()).toEqual(FITTED));
    act(() => void result.current.flow.setViewport({ x: 0, y: 0, zoom: 1 }));
    rerender({ enabled: true });
    expect(result.current.flow.getViewport()).toEqual({ x: 0, y: 0, zoom: 1 });
  });

  it("fits again after a disable and re-enable cycle", async () => {
    const { result, rerender } = renderInitialFit(true);
    act(() => result.current.measure("a"));
    await waitFor(() => expect(result.current.flow.getViewport()).toEqual(FITTED));
    act(() => void result.current.flow.setViewport({ x: 0, y: 0, zoom: 1 }));
    rerender({ enabled: false });
    rerender({ enabled: true });
    await waitFor(() => expect(result.current.flow.getViewport()).toEqual(FITTED));
  });

  it("does not fit while disabled", async () => {
    const { result } = renderInitialFit(false);
    act(() => result.current.measure("a"));
    await new Promise((resolve) => setTimeout(resolve, 100));
    expect(result.current.flow.getViewport()).toEqual({ x: 0, y: 0, zoom: 1 });
  });
});
