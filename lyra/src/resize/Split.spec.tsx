// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Resize } from "@/resize";
import { mockBoundingClientRect } from "@/testutil/dom";

const PARENT_SIZE = 1000;

const renderSplit = (
  props: Omit<Resize.SplitProps, "children">,
): ReturnType<typeof render> =>
  render(
    <Resize.Split {...props}>
      <p>First</p>
      <p>Last</p>
    </Resize.Split>,
  );

const handleOf = (c: ReturnType<typeof render>): HTMLElement => {
  const handle = c.container.querySelector<HTMLElement>(".pluto-resize__handle");
  if (handle == null) throw new Error("resize handle not found");
  return handle;
};

const panesOf = (c: ReturnType<typeof render>): HTMLElement[] =>
  Array.from(c.container.querySelectorAll<HTMLElement>(".pluto-resize"));

const drag = async (
  c: ReturnType<typeof render>,
  { from, to }: { from: number; to: number },
): Promise<void> => {
  // Cursor.useDrag captures the pointer on the handle, then tracks moves/up on window
  // (capture bubbles them up). The from -> to distance must exceed the activation
  // threshold for the drag to begin.
  fireEvent.pointerDown(handleOf(c), {
    pointerId: 1,
    button: 0,
    isPrimary: true,
    clientX: from,
    clientY: 0,
  });
  fireEvent.pointerMove(window, { pointerId: 1, clientX: to, clientY: 0 });
  // Moves are coalesced onto the next animation frame; let one land before release.
  await act(
    async () =>
      await new Promise<void>((resolve) => requestAnimationFrame(() => resolve())),
  );
  fireEvent.pointerUp(window, { pointerId: 1, clientX: to, clientY: 0 });
};

const lastSize = (onResize: ReturnType<typeof vi.fn>): number =>
  onResize.mock.lastCall?.[0] as number;

describe("Resize.Split", () => {
  beforeEach(() => {
    vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(
      mockBoundingClientRect(0, 0, PARENT_SIZE, PARENT_SIZE),
    );
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should render both panes", () => {
    const c = renderSplit({});
    expect(c.getByText("First")).toBeTruthy();
    expect(c.getByText("Last")).toBeTruthy();
  });

  it("should render a handle only between the panes", () => {
    const c = renderSplit({});
    expect(c.container.querySelectorAll(".pluto-resize__handle")).toHaveLength(1);
  });

  it("should grow the first pane when the handle is dragged forward", async () => {
    const onResize = vi.fn();
    const c = renderSplit({ size: 0.5, onResize });
    await drag(c, { from: 500, to: 600 });
    expect(lastSize(onResize)).toBeCloseTo(0.6);
  });

  it("should shrink the first pane when the handle is dragged backward", async () => {
    const onResize = vi.fn();
    const c = renderSplit({ size: 0.5, onResize });
    await drag(c, { from: 500, to: 350 });
    expect(lastSize(onResize)).toBeCloseTo(0.35);
  });

  it("should clamp the first pane to the minimum size", async () => {
    const onResize = vi.fn();
    const c = renderSplit({ size: 0.5, minSize: 100, onResize });
    await drag(c, { from: 500, to: -10000 });
    expect(lastSize(onResize)).toBeCloseTo(100 / PARENT_SIZE);
  });

  it("should clamp so the second pane keeps the minimum size", async () => {
    const onResize = vi.fn();
    const c = renderSplit({ size: 0.5, minSize: 100, onResize });
    await drag(c, { from: 500, to: 10000 });
    expect(lastSize(onResize)).toBeCloseTo(1 - 100 / PARENT_SIZE);
  });

  it("should commit the final ratio once on drag end", async () => {
    const onResizeEnd = vi.fn();
    const c = renderSplit({ size: 0.5, onResizeEnd });
    await drag(c, { from: 500, to: 600 });
    expect(onResizeEnd).toHaveBeenCalledTimes(1);
    expect(lastSize(onResizeEnd)).toBeCloseTo(0.6);
  });

  it("should not report a resize on mount", () => {
    const onResize = vi.fn();
    const onResizeEnd = vi.fn();
    renderSplit({ size: 0.5, onResize, onResizeEnd });
    expect(onResize).not.toHaveBeenCalled();
    expect(onResizeEnd).not.toHaveBeenCalled();
  });

  it("should offset an even split so re-render observers fire", () => {
    const c = renderSplit({ size: 0.5 });
    const [first, last] = panesOf(c);
    expect(first.style.width).not.toEqual(last.style.width);
  });
});
