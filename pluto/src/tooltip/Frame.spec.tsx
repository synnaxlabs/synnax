// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Tooltip } from "@/tooltip";

const ANCHOR_RECT = {
  left: 100,
  top: 100,
  right: 200,
  bottom: 140,
  width: 100,
  height: 40,
  x: 100,
  y: 100,
  toJSON: () => ({}),
} as DOMRect;

const MOVED_RECT = { ...ANCHOR_RECT, left: 300, right: 400, x: 300 } as DOMRect;

const ZERO_RECT = {
  left: 0,
  top: 0,
  right: 0,
  bottom: 0,
  width: 0,
  height: 0,
  x: 0,
  y: 0,
  toJSON: () => ({}),
} as DOMRect;

const getFrame = (): HTMLElement | null =>
  document.querySelector<HTMLElement>(".pluto-tooltip");

const createAnchor = (rect: DOMRect = ANCHOR_RECT): HTMLElement => {
  const el = document.createElement("div");
  el.getBoundingClientRect = () => rect;
  document.body.appendChild(el);
  return el;
};

describe("Tooltip.Frame", () => {
  it("should render a tooltip above the anchor by default", () => {
    const anchor = createAnchor();
    render(<Tooltip.Frame anchor={anchor}>Tip</Tooltip.Frame>);
    const frame = getFrame();
    expect(frame?.getAttribute("role")).toBe("tooltip");
    expect(frame?.textContent).toBe("Tip");
    expect(frame?.style.left).toBe("150px");
    expect(frame?.style.top).toBe("94px");
  });

  it("should honor the preferred location", () => {
    const anchor = createAnchor();
    render(
      <Tooltip.Frame anchor={anchor} location={{ x: "center", y: "bottom" }}>
        Tip
      </Tooltip.Frame>,
    );
    expect(getFrame()?.style.left).toBe("150px");
    expect(getFrame()?.style.top).toBe("146px");
  });

  it("should apply the id and class name", () => {
    const anchor = createAnchor();
    render(
      <Tooltip.Frame anchor={anchor} id="tip" className="custom">
        Tip
      </Tooltip.Frame>,
    );
    const frame = getFrame();
    expect(frame?.id).toBe("tip");
    expect(frame?.classList).toContain("custom");
  });

  it("should follow the anchor when it moves", () => {
    const anchor = createAnchor();
    const { rerender } = render(<Tooltip.Frame anchor={anchor}>Tip</Tooltip.Frame>);
    anchor.getBoundingClientRect = () => MOVED_RECT;
    rerender(<Tooltip.Frame anchor={anchor}>Tip</Tooltip.Frame>);
    expect(getFrame()?.style.left).toBe("350px");
  });

  it("should report a lost anchor once it leaves the document", () => {
    const anchor = createAnchor();
    const onAnchorLost = vi.fn();
    const { rerender } = render(
      <Tooltip.Frame anchor={anchor} onAnchorLost={onAnchorLost}>
        Tip
      </Tooltip.Frame>,
    );
    expect(onAnchorLost).not.toHaveBeenCalled();
    anchor.remove();
    rerender(
      <Tooltip.Frame anchor={anchor} onAnchorLost={onAnchorLost}>
        Tip
      </Tooltip.Frame>,
    );
    expect(onAnchorLost).toHaveBeenCalled();
  });

  it("should report a lost anchor when it has no area", () => {
    const anchor = createAnchor(ZERO_RECT);
    const onAnchorLost = vi.fn();
    render(
      <Tooltip.Frame anchor={anchor} onAnchorLost={onAnchorLost}>
        Tip
      </Tooltip.Frame>,
    );
    expect(onAnchorLost).toHaveBeenCalled();
  });
});
