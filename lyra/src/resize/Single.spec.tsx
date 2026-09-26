// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";
import { act, fireEvent, render } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Resize } from "@/resize";

// Single is controlled: it renders the committed size when idle and a transient drag
// size while dragging. This harness mirrors a real consumer by committing onResizeEnd
// back into the controlled size, so the pane reflects a completed drag.
const ControlledSingle = ({
  size: initial = 200,
  onResizeEnd,
  ...props
}: Resize.SingleProps): ReactElement => {
  const [size, setSize] = useState(initial);
  return (
    <Resize.Single
      size={size}
      onResizeEnd={(s: number, extra: Resize.HandlerExtra) => {
        setSize(s);
        onResizeEnd?.(s, extra);
      }}
      {...props}
    >
      <p>Hello</p>
    </Resize.Single>
  );
};

const renderSingle = (props: Resize.SingleProps): ReturnType<typeof render> =>
  render(<ControlledSingle {...props} />);

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

const drag = async (
  c: ReturnType<typeof render>,
  from: xy.XY,
  to: xy.XY,
): Promise<void> => {
  // Cursor.useDrag captures the pointer on the handle, then tracks moves/up on window
  // (capture bubbles them up). The from -> to distance must exceed the activation
  // threshold for the drag to begin.
  fireEvent.pointerDown(handleOf(c), {
    pointerId: 1,
    button: 0,
    isPrimary: true,
    clientX: from.x,
    clientY: from.y,
  });
  fireEvent.pointerMove(window, { pointerId: 1, clientX: to.x, clientY: to.y });
  // Moves are coalesced onto the next animation frame; let one land before release.
  await act(
    async () =>
      await new Promise<void>((resolve) => requestAnimationFrame(() => resolve())),
  );
  fireEvent.pointerUp(window, { pointerId: 1, clientX: to.x, clientY: to.y });
};

const lastSize = (onResize: ReturnType<typeof vi.fn>): number =>
  onResize.mock.lastCall?.[0] as number;

describe("Resize.Single", () => {
  it("should render its children", () => {
    const c = renderSingle({ location: "left", size: 50 });
    expect(c.getByText("Hello")).toBeTruthy();
  });

  it("should render a resize handle", () => {
    const c = renderSingle({ location: "left" });
    expect(handleOf(c)).toBeTruthy();
  });

  describe("initial size", () => {
    it("should apply the initial size as width for a horizontal location", () => {
      const c = renderSingle({ location: "left", size: 240 });
      expect(paneOf(c).style.width).toEqual("240px");
    });

    it("should apply the initial size as height for a vertical location", () => {
      const c = renderSingle({ location: "top", size: 180 });
      expect(paneOf(c).style.height).toEqual("180px");
    });

    it("should clamp an initial size below the lower bound", () => {
      const c = renderSingle({ location: "left", size: 40 });
      expect(paneOf(c).style.width).toEqual("100px");
    });

    it("should clamp an initial size above the upper bound", () => {
      const c = renderSingle({
        location: "left",
        size: 800,
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
      async ({ location, from, to, dimension, expected }) => {
        const onResize = vi.fn();
        const c = renderSingle({ location, size: 200, onResize });
        await drag(c, from, to);
        expect(lastSize(onResize)).toEqual(expected);
        expect(paneOf(c).style[dimension]).toEqual(`${expected}px`);
      },
    );

    it("should pass the pane box and raw drag size as the second argument", async () => {
      const onResize = vi.fn();
      const c = renderSingle({ location: "left", size: 200, onResize });
      await drag(c, { x: 500, y: 0 }, { x: 540, y: 0 });
      const extra = onResize.mock.lastCall?.[1] as Resize.HandlerExtra;
      expect(extra.box).toBeDefined();
      expect(extra.dragSize).toBe(240);
    });

    it("should not start a drag for movement below the activation threshold", () => {
      const onResize = vi.fn();
      const onResizeEnd = vi.fn();
      const c = renderSingle({ location: "left", size: 200, onResize, onResizeEnd });
      const handle = handleOf(c);
      fireEvent.pointerDown(handle, {
        pointerId: 1,
        button: 0,
        isPrimary: true,
        clientX: 500,
        clientY: 0,
      });
      fireEvent.pointerMove(window, { pointerId: 1, clientX: 502, clientY: 0 });
      fireEvent.pointerUp(window, { pointerId: 1, clientX: 502, clientY: 0 });
      expect(onResize).not.toHaveBeenCalled();
      expect(onResizeEnd).not.toHaveBeenCalled();
    });
  });
});
