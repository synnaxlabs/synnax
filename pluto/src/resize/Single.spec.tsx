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
import { describe, expect, it, vi } from "vitest";

import { Resize } from "@/resize";

const renderSingle = (props: Resize.SingleProps): ReturnType<typeof render> =>
  render(
    <Resize.Single {...props}>
      <p>Hello</p>
    </Resize.Single>,
  );

const paneOf = (c: ReturnType<typeof render>): HTMLElement => {
  const pane = c.container.querySelector<HTMLElement>(".pluto-resize");
  if (pane == null) throw new Error("resize pane not found");
  return pane;
};

const handleOf = (c: ReturnType<typeof render>): HTMLElement => {
  const handle = c.container.querySelector<HTMLElement>(".pluto-resize__handle");
  if (handle == null) throw new Error("resize handle not found");
  return handle;
};

const drag = (c: ReturnType<typeof render>, from: xy.XY, to: xy.XY): void => {
  // jsdom's synthetic dragStart drops clientX/Y, so dispatch a MouseEvent (which carries
  // them) under the "dragstart" type to drive useCursorDrag.
  fireEvent(
    handleOf(c),
    new MouseEvent("dragstart", { clientX: from.x, clientY: from.y, bubbles: true }),
  );
  fireEvent.mouseMove(window, { clientX: to.x, clientY: to.y, buttons: 1 });
  fireEvent.mouseUp(window, { clientX: to.x, clientY: to.y });
};

const lastSize = (onResize: ReturnType<typeof vi.fn>): number =>
  onResize.mock.lastCall?.[0] as number;

describe("Resize.Single", () => {
  it("should render its children", () => {
    const c = renderSingle({ location: "left", initialSize: 50 });
    expect(c.getByText("Hello")).toBeTruthy();
  });

  it("should render a draggable handle", () => {
    const c = renderSingle({ location: "left" });
    expect(handleOf(c).draggable).toBe(true);
  });

  describe("initial size", () => {
    it("should apply the initial size as width for a horizontal location", () => {
      const c = renderSingle({ location: "left", initialSize: 240 });
      expect(paneOf(c).style.width).toEqual("240px");
    });

    it("should apply the initial size as height for a vertical location", () => {
      const c = renderSingle({ location: "top", initialSize: 180 });
      expect(paneOf(c).style.height).toEqual("180px");
    });

    it("should clamp an initial size below the lower bound", () => {
      const c = renderSingle({ location: "left", initialSize: 40 });
      expect(paneOf(c).style.width).toEqual("100px");
    });

    it("should clamp an initial size above the upper bound", () => {
      const c = renderSingle({
        location: "left",
        initialSize: 800,
        sizeBounds: { lower: 100, upper: 300 },
      });
      expect(paneOf(c).style.width).toEqual("300px");
    });
  });

  describe("drag", () => {
    interface DragCase {
      location: Resize.SingleProps["location"];
      from: xy.XY;
      to: xy.XY;
      dimension: "width" | "height";
      expected: number;
    }
    it.each<DragCase>([
      {
        location: "left",
        from: { x: 500, y: 0 },
        to: { x: 560, y: 0 },
        dimension: "width",
        expected: 260,
      },
      {
        location: "right",
        from: { x: 500, y: 0 },
        to: { x: 560, y: 0 },
        dimension: "width",
        expected: 140,
      },
      {
        location: "top",
        from: { x: 0, y: 500 },
        to: { x: 0, y: 560 },
        dimension: "height",
        expected: 260,
      },
      {
        location: "bottom",
        from: { x: 0, y: 500 },
        to: { x: 0, y: 560 },
        dimension: "height",
        expected: 140,
      },
    ])(
      "should resize to $expected when dragged from a $location location",
      ({ location, from, to, dimension, expected }) => {
        const onResize = vi.fn();
        const c = renderSingle({ location, initialSize: 200, onResize });
        drag(c, from, to);
        expect(lastSize(onResize)).toEqual(expected);
        expect(paneOf(c).style[dimension]).toEqual(`${expected}px`);
      },
    );

    it("should pass the panel box as the second argument to onResize", () => {
      const onResize = vi.fn();
      const c = renderSingle({ location: "left", initialSize: 200, onResize });
      drag(c, { x: 500, y: 0 }, { x: 540, y: 0 });
      expect(onResize).toHaveBeenCalledWith(expect.any(Number), expect.anything());
    });

    it("should clamp to the lower bound without collapsing past it", () => {
      const onResize = vi.fn();
      const onCollapse = vi.fn();
      const c = renderSingle({
        location: "left",
        initialSize: 200,
        sizeBounds: { lower: 100 },
        onResize,
        onCollapse,
      });
      drag(c, { x: 500, y: 0 }, { x: 300, y: 0 });
      expect(lastSize(onResize)).toEqual(100);
      expect(paneOf(c).style.width).toEqual("100px");
      expect(onCollapse).not.toHaveBeenCalled();
    });
  });

  describe("collapse", () => {
    it("should collapse and fire onCollapse when dragged past the threshold", () => {
      const onCollapse = vi.fn();
      const c = renderSingle({
        location: "left",
        initialSize: 200,
        sizeBounds: { lower: 100 },
        collapseThreshold: 0.5,
        onCollapse,
      });
      drag(c, { x: 500, y: 0 }, { x: 300, y: 0 });
      expect(paneOf(c).style.width).toEqual("2px");
      expect(paneOf(c).className).toContain("pluto--collapsed");
      expect(onCollapse).toHaveBeenCalledTimes(1);
    });

    it("should mark the pane expanded while above the collapsed size", () => {
      const c = renderSingle({ location: "left", initialSize: 200 });
      expect(paneOf(c).className).toContain("pluto--expanded");
    });
  });
});
