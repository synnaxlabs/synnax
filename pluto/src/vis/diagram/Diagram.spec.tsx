// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";

import { Component } from "@/component";
import { render } from "@/testutil";
import { Triggers } from "@/triggers";
import { Diagram } from "@/vis/diagram";
import { diagram } from "@/vis/diagram/aether";

// The nodrag child stands in for a connection handle: React Flow's drag filter rejects
// it, so its mousedown reaches the window like a click on the canvas would.
const NodeView = ({ nodeKey }: Diagram.NodeProps): ReactElement => (
  <div>
    {nodeKey}
    <span className="nodrag" data-testid={`handle-${nodeKey}`} />
  </div>
);

const Base = Diagram.create({ node: Component.renderProp(NodeView) });

const NODES: Diagram.Node[] = [
  { key: "a", position: { x: 0, y: 0 } },
  { key: "b", position: { x: 200, y: 0 } },
];
const VIEWPORT: Diagram.Viewport = { position: { x: 0, y: 0 }, zoom: 1 };
const noop = () => {};

interface HarnessProps {
  onSelectionChange: (selected: string[]) => void;
  onViewportChange: (viewport: Diagram.Viewport) => void;
}

// The diagram is controlled, so the harness feeds each selection back as the next
// `selected` prop the way a caller would.
const Harness = ({
  onSelectionChange,
  onViewportChange,
}: HarnessProps): ReactElement => {
  const [selected, setSelected] = useState<string[]>([]);
  const handleSelectionChange = (next: string[]): void => {
    setSelected(next);
    onSelectionChange(next);
  };
  return (
    <Triggers.Provider>
      <Base
        nodes={NODES}
        edges={[]}
        onNodesChange={noop}
        onEdgesChange={noop}
        selected={selected}
        onSelectionChange={handleSelectionChange}
        editable
        onEditableChange={noop}
        viewport={VIEWPORT}
        onViewportChange={onViewportChange}
        fitViewOnResize={false}
        setFitViewOnResize={noop}
        viewportMode="zoom"
        onViewportModeChange={noop}
        visible
      />
    </Triggers.Provider>
  );
};

const RECT = {
  x: 0,
  y: 0,
  width: 1000,
  height: 500,
  top: 0,
  left: 0,
  right: 1000,
  bottom: 500,
  toJSON: () => ({}),
} as DOMRect;

// The @juggle polyfill reads computed styles, which jsdom leaves empty, so it never
// reports a size. useResize and the React Flow container re-measure on notify and
// ignore the entries. Nodes stay unmeasured: React Flow reads their transforms through
// DOMMatrixReadOnly, which jsdom lacks.
class ImmediateResizeObserver {
  constructor(private readonly notify: (entries: ResizeObserverEntry[]) => void) {}
  observe(el: Element): void {
    if (!el.classList.contains("react-flow__node")) this.notify([]);
  }
  unobserve(): void {}
  disconnect(): void {}
}

const renderDiagram = async () => {
  const onSelectionChange = vi.fn<(selected: string[]) => void>();
  const onViewportChange = vi.fn<(viewport: Diagram.Viewport) => void>();
  render(
    <Harness
      onSelectionChange={onSelectionChange}
      onViewportChange={onViewportChange}
    />,
    { registry: diagram.REGISTRY, render: true },
  );
  await waitFor(() => expect(node("b")).not.toBeNull());
  return { onSelectionChange, onViewportChange };
};

const node = (key: string): HTMLElement =>
  document.querySelector(`.react-flow__node[data-id="${key}"]`) as HTMLElement;

const handle = (key: string): HTMLElement =>
  document.querySelector(`[data-testid="handle-${key}"]`) as HTMLElement;

const pane = (): HTMLElement =>
  document.querySelector(".react-flow__pane") as HTMLElement;

describe("Diagram", () => {
  // Canvas.useRegion measures against the lower2d canvas and bails without it.
  const canvas = document.createElement("div");

  beforeAll(() => {
    vi.stubGlobal("ResizeObserver", ImmediateResizeObserver);
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue(RECT);
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(RECT.width);
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(RECT.height);
    canvas.className = "pluto-canvas--lower2d";
    document.body.appendChild(canvas);
  });

  afterAll(() => {
    canvas.remove();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  describe("selection", () => {
    it("replaces the selection on a plain click", async () => {
      const { onSelectionChange } = await renderDiagram();
      fireEvent.click(node("a"));
      expect(onSelectionChange).toHaveBeenLastCalledWith(["a"]);
      fireEvent.click(node("b"));
      expect(onSelectionChange).toHaveBeenLastCalledWith(["b"]);
    });

    it.each(["Meta", "Control", "Shift"])(
      "toggles the clicked node in the selection while %s is held",
      async (key) => {
        const { onSelectionChange } = await renderDiagram();
        fireEvent.click(node("a"));
        fireEvent.keyDown(window, { key, code: `${key}Left` });
        fireEvent.click(node("b"));
        expect(onSelectionChange).toHaveBeenLastCalledWith(["a", "b"]);
        fireEvent.click(node("a"));
        expect(onSelectionChange).toHaveBeenLastCalledWith(["b"]);
        fireEvent.keyUp(window, { key, code: `${key}Left` });
        fireEvent.click(node("a"));
        expect(onSelectionChange).toHaveBeenLastCalledWith(["a"]);
      },
    );
  });

  describe("zoom reset", () => {
    const holdControl = (): void => {
      fireEvent.mouseMove(window, { clientX: 10, clientY: 10 });
      fireEvent.keyDown(window, { key: "Control", code: "ControlLeft" });
    };

    it("resets the view on a Control-click of the canvas", async () => {
      const { onViewportChange } = await renderDiagram();
      holdControl();
      onViewportChange.mockClear();
      fireEvent.mouseDown(pane(), { button: 0 });
      expect(onViewportChange).toHaveBeenCalled();
    });

    it("leaves the view alone on a Control-click of a node", async () => {
      const { onViewportChange } = await renderDiagram();
      holdControl();
      onViewportChange.mockClear();
      fireEvent.mouseDown(handle("a"), { button: 0 });
      expect(onViewportChange).not.toHaveBeenCalled();
    });
  });
});
