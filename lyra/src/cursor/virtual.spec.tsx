// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, useRef } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Cursor } from "@/cursor";

// useVirtualDrag drives the gesture through pointer capture (el.onpointermove, plus a
// once-only pointerup listener on the element). The shared setup file stubs the capture
// APIs with no-ops, so install a faithful per-element implementation and restore the
// stubs afterwards.
const installPointerCapture = () => {
  const captured = new WeakMap<HTMLElement, Set<number>>();
  const setPointerCapture = vi.fn(function (this: HTMLElement, id: number) {
    const ids = captured.get(this) ?? new Set<number>();
    ids.add(id);
    captured.set(this, ids);
  });
  const releasePointerCapture = vi.fn(function (this: HTMLElement, id: number) {
    captured.get(this)?.delete(id);
  });
  const hasPointerCapture = vi.fn(function (this: HTMLElement, id: number) {
    return captured.get(this)?.has(id) ?? false;
  });
  HTMLElement.prototype.setPointerCapture = setPointerCapture;
  HTMLElement.prototype.releasePointerCapture = releasePointerCapture;
  HTMLElement.prototype.hasPointerCapture = hasPointerCapture;
  return { setPointerCapture, releasePointerCapture, hasPointerCapture };
};

const restorePointerCapture = () => {
  HTMLElement.prototype.setPointerCapture = () => {};
  HTMLElement.prototype.releasePointerCapture = () => {};
  HTMLElement.prototype.hasPointerCapture = () => false;
};

type VirtualProps = Omit<Cursor.UseVirtualDragProps, "ref">;

const Harness = (props: VirtualProps): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);
  Cursor.useVirtualDrag({ ...props, ref });
  return <div data-testid="target" ref={ref} />;
};

const DEFAULT_POINTER = { pointerId: 1, button: 0, isPrimary: true };

const pointerInit = (x: number, y: number, overrides: object = {}) => ({
  ...DEFAULT_POINTER,
  ...overrides,
  clientX: x,
  clientY: y,
});

describe("Cursor.useVirtualDrag", () => {
  let pointerCapture: ReturnType<typeof installPointerCapture>;

  beforeEach(() => {
    pointerCapture = installPointerCapture();
  });

  afterEach(() => {
    restorePointerCapture();
  });

  const renderTarget = (
    props: VirtualProps,
  ): { el: HTMLElement; unmount: () => void } => {
    const c = render(<Harness {...props} />);
    return { el: c.getByTestId("target"), unmount: c.unmount };
  };

  const down = (el: HTMLElement, x: number, y: number, overrides?: object): void => {
    fireEvent.pointerDown(el, pointerInit(x, y, overrides));
  };
  const move = (el: HTMLElement, x: number, y: number): void => {
    fireEvent.pointerMove(el, pointerInit(x, y));
  };
  const up = (el: HTMLElement, x: number, y: number, overrides?: object): void => {
    fireEvent.pointerUp(el, pointerInit(x, y, overrides));
  };

  it("should activate immediately on press with no threshold", () => {
    const onStart = vi.fn();
    const { el } = renderTarget({ onStart });
    down(el, 10, 20);
    expect(onStart).toHaveBeenCalledWith({ x: 10, y: 20 }, "MouseLeft", el);
  });

  describe("activation guards", () => {
    it("should ignore a press from a non-primary button", () => {
      const onStart = vi.fn();
      const { el } = renderTarget({ onStart });
      down(el, 0, 0, { button: 2 });
      expect(onStart).not.toHaveBeenCalled();
      expect(pointerCapture.setPointerCapture).not.toHaveBeenCalled();
    });

    it("should ignore a non-primary pointer", () => {
      const onStart = vi.fn();
      const { el } = renderTarget({ onStart });
      down(el, 0, 0, { isPrimary: false });
      expect(onStart).not.toHaveBeenCalled();
    });
  });

  describe("pointer identity", () => {
    it("should ignore moves from a different pointer", () => {
      const onMove = vi.fn();
      const { el } = renderTarget({ onMove });
      down(el, 0, 0, { pointerId: 1 });
      move(el, 50, 0);
      onMove.mockClear();
      fireEvent.pointerMove(el, pointerInit(100, 0, { pointerId: 2 }));
      expect(onMove).not.toHaveBeenCalled();
    });

    it("should ignore an up from a different pointer and still end on the real up", () => {
      const onEnd = vi.fn();
      const { el } = renderTarget({ onEnd });
      down(el, 0, 0, { pointerId: 1 });
      up(el, 10, 0, { pointerId: 2 });
      expect(onEnd).not.toHaveBeenCalled();
      up(el, 20, 0, { pointerId: 1 });
      expect(onEnd).toHaveBeenCalledTimes(1);
    });
  });

  it("should capture the pointer to the element on press", () => {
    const { el } = renderTarget({});
    down(el, 0, 0, { pointerId: 9 });
    expect(pointerCapture.setPointerCapture).toHaveBeenCalledWith(9);
    expect(el.hasPointerCapture(9)).toBe(true);
  });

  it("should report a box spanning the press to the current location on move", () => {
    const onMove = vi.fn();
    const { el } = renderTarget({ onMove });
    down(el, 100, 100);
    move(el, 160, 140);
    const b = onMove.mock.lastCall?.[0] as box.Box;
    expect(box.left(b)).toEqual(100);
    expect(box.top(b)).toEqual(100);
    expect(box.width(b)).toEqual(60);
    expect(box.height(b)).toEqual(40);
  });

  it("should report the final box to onEnd on pointer up", () => {
    const onEnd = vi.fn();
    const { el } = renderTarget({ onEnd });
    down(el, 0, 0);
    move(el, 30, 30);
    up(el, 50, 80);
    expect(onEnd).toHaveBeenCalledTimes(1);
    const b = onEnd.mock.lastCall?.[0] as box.Box;
    expect(box.width(b)).toEqual(50);
    expect(box.height(b)).toEqual(80);
  });

  it("should release the captured pointer on pointer up", () => {
    const { el } = renderTarget({});
    down(el, 0, 0, { pointerId: 9 });
    up(el, 10, 10, { pointerId: 9 });
    expect(pointerCapture.releasePointerCapture).toHaveBeenCalledWith(9);
    expect(el.hasPointerCapture(9)).toBe(false);
  });

  it("should stop reporting moves after the gesture ends", () => {
    const onMove = vi.fn();
    const { el } = renderTarget({ onMove });
    down(el, 0, 0);
    move(el, 20, 0);
    up(el, 20, 0);
    onMove.mockClear();
    move(el, 40, 0);
    expect(onMove).not.toHaveBeenCalled();
  });

  it("should end the gesture only once", () => {
    const onEnd = vi.fn();
    const { el } = renderTarget({ onEnd });
    down(el, 0, 0);
    up(el, 10, 0);
    up(el, 20, 0);
    expect(onEnd).toHaveBeenCalledTimes(1);
  });

  it("should detach the pointerdown listener on unmount", () => {
    const onStart = vi.fn();
    const { el, unmount } = renderTarget({ onStart });
    unmount();
    down(el, 0, 0);
    expect(onStart).not.toHaveBeenCalled();
  });
});
