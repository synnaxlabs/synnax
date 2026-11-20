// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { assert, describe, expect, it, vi } from "vitest";

import { Portal } from "@/portal";
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
  document.querySelector<HTMLElement>(".lyra-tooltip");

// The left edge that centers the tooltip's real width on x.
const centeredLeft = (tip: HTMLElement, x: number): string =>
  `${Math.round(x - tip.offsetWidth / 2)}px`;

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
    assert(frame != null);
    expect(frame.getAttribute("role")).toBe("tooltip");
    expect(frame.textContent).toBe("Tip");
    expect(frame.style.left).toBe(centeredLeft(frame, 150));
    expect(frame.style.top).toBe(`${94 - frame.offsetHeight}px`);
  });

  it("should honor the preferred location", () => {
    const anchor = createAnchor();
    render(
      <Tooltip.Frame anchor={anchor} location={{ x: "center", y: "bottom" }}>
        Tip
      </Tooltip.Frame>,
    );
    const frame = getFrame();
    assert(frame != null);
    expect(frame.style.left).toBe(centeredLeft(frame, 150));
    expect(frame.style.top).toBe("146px");
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
    const frame = getFrame();
    assert(frame != null);
    expect(frame.style.left).toBe(centeredLeft(frame, 350));
  });

  it("should keep the tooltip inside the window's right edge", () => {
    const anchor = createAnchor({ ...ANCHOR_RECT, left: 980, right: 1000, x: 980 });
    render(<Tooltip.Frame anchor={anchor}>Tip</Tooltip.Frame>);
    const frame = getFrame() as HTMLElement;
    Object.defineProperty(frame, "offsetWidth", { get: () => 200 });
    window.dispatchEvent(new Event("resize"));
    expect(frame.style.left).toBe(`${window.innerWidth - 200}px`);
  });

  it("should measure the tooltip at the origin", () => {
    const anchor = createAnchor({ ...ANCHOR_RECT, left: 980, right: 1000, x: 980 });
    render(<Tooltip.Frame anchor={anchor}>Tip</Tooltip.Frame>);
    const frame = getFrame() as HTMLElement;
    // The box reads narrower once placed, as a wrapped one near the edge does.
    Object.defineProperty(frame, "offsetWidth", {
      get: () => (frame.style.left === "0px" ? 200 : 50),
    });
    window.dispatchEvent(new Event("resize"));
    expect(frame.style.left).toBe(`${window.innerWidth - 200}px`);
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

  it("should portal into the root element by default", () => {
    const root = document.createElement("div");
    root.id = "root";
    document.body.appendChild(root);
    render(<Tooltip.Frame anchor={createAnchor()}>Tip</Tooltip.Frame>);
    expect(getFrame()?.parentElement).toBe(root);
    root.remove();
  });

  it("should portal into a Portal.Target", () => {
    const target = document.createElement("div");
    document.body.appendChild(target);
    render(
      <Portal.Target element={target}>
        <Tooltip.Frame anchor={createAnchor()}>Tip</Tooltip.Frame>
      </Portal.Target>,
    );
    expect(getFrame()?.parentElement).toBe(target);
    target.remove();
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
