// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location, type sticky, type xy } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CSS } from "@/css";
import { Legend } from "@/vis/legend";

const setRect = (el: Element, left: number, top: number, w: number, h: number) =>
  Object.defineProperty(el, "getBoundingClientRect", {
    value: () => ({
      left,
      top,
      width: w,
      height: h,
      x: left,
      y: top,
      right: left + w,
      bottom: top + h,
      toJSON: () => ({}),
    }),
  });

const renderContainer = (props: Legend.ContainerProps): HTMLElement => {
  const { container } = render(
    <div>
      <Legend.Container {...props} />
    </div>,
  );
  const legend = container.querySelector(`.${CSS.B("legend")}`);
  if (!(legend instanceof HTMLElement) || legend.parentElement == null)
    throw new Error("legend not found");
  setRect(legend.parentElement, 0, 0, 400, 400);
  setRect(legend, 40, 40, 100, 50);
  return legend;
};

const drag = (el: HTMLElement, from: xy.XY, to: xy.XY): void => {
  fireEvent.pointerDown(el, {
    pointerId: 1,
    button: 0,
    isPrimary: true,
    clientX: from.x,
    clientY: from.y,
  });
  fireEvent.pointerMove(window, { pointerId: 1, clientX: to.x, clientY: to.y });
  fireEvent.pointerUp(window, { pointerId: 1, clientX: to.x, clientY: to.y });
};

describe("Legend.Container", () => {
  it("should move the legend by the drag distance", () => {
    const onChange = vi.fn<(v: sticky.XY) => void>();
    const legend = renderContainer({ onChange });
    drag(legend, { x: 50, y: 50 }, { x: 150, y: 150 });
    expect(onChange).toHaveBeenLastCalledWith({
      x: 0.35,
      y: 0.35,
      root: location.TOP_LEFT,
      units: { x: "decimal", y: "decimal" },
    });
  });

  it("should keep the legend inside its parent", () => {
    const onChange = vi.fn<(v: sticky.XY) => void>();
    const legend = renderContainer({ onChange });
    drag(legend, { x: 50, y: 50 }, { x: -150, y: 50 });
    expect(onChange).toHaveBeenLastCalledWith({
      x: 0,
      y: 40,
      root: location.TOP_LEFT,
      units: { x: "px", y: "px" },
    });
  });

  it("should not move the legend when it is not draggable", () => {
    const onChange = vi.fn<(v: sticky.XY) => void>();
    const legend = renderContainer({ onChange, draggable: false });
    drag(legend, { x: 50, y: 50 }, { x: 150, y: 150 });
    expect(onChange).not.toHaveBeenCalled();
  });
});
