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
import { type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Cursor } from "@/cursor";

// The shared setup file (src/mock/setuptests.ts) stubs the pointer-capture APIs with
// no-ops, with hasPointerCapture hard-wired to false. That makes the capture/release
// branch of useDrag untestable. This installs a faithful per-element implementation so
// the lifecycle can be asserted, and restores the no-op stubs afterwards.
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

const Harness = (props: Cursor.UseDragProps): ReactElement => {
  const onPointerDown = Cursor.useDrag(props);
  return <div data-testid="target" onPointerDown={onPointerDown} />;
};

interface PointerInit {
  pointerId?: number;
  button?: number;
  isPrimary?: boolean;
  x: number;
  y: number;
}

const DEFAULT_POINTER = { pointerId: 1, button: 0, isPrimary: true };

const pointerInit = ({ x, y, ...rest }: PointerInit) => ({
  ...DEFAULT_POINTER,
  ...rest,
  clientX: x,
  clientY: y,
});

// Moves are coalesced and delivered on the next animation frame.
const frame = async (): Promise<void> =>
  await new Promise((resolve) => requestAnimationFrame(() => resolve()));

describe("Cursor.useDrag", () => {
  let pointerCapture: ReturnType<typeof installPointerCapture>;

  beforeEach(() => {
    pointerCapture = installPointerCapture();
  });

  afterEach(() => {
    restorePointerCapture();
  });

  const renderTarget = (props: Cursor.UseDragProps): HTMLElement => {
    const c = render(<Harness {...props} />);
    return c.getByTestId("target");
  };

  const down = (el: HTMLElement, init: PointerInit): void => {
    fireEvent.pointerDown(el, pointerInit(init));
  };
  const move = (init: PointerInit): void => {
    fireEvent.pointerMove(window, pointerInit(init));
  };
  const up = (init: PointerInit): void => {
    fireEvent.pointerUp(window, pointerInit(init));
  };
  const cancel = (init: PointerInit): void => {
    fireEvent.pointerCancel(window, pointerInit(init));
  };

  describe("activation guards", () => {
    it("should ignore a press from a non-primary button", () => {
      const onStart = vi.fn();
      const el = renderTarget({ onStart });
      down(el, { button: 2, x: 0, y: 0 });
      move({ x: 100, y: 100 });
      expect(onStart).not.toHaveBeenCalled();
    });

    it("should ignore a non-primary pointer", () => {
      const onStart = vi.fn();
      const el = renderTarget({ onStart });
      down(el, { isPrimary: false, x: 0, y: 0 });
      move({ x: 100, y: 100 });
      expect(onStart).not.toHaveBeenCalled();
    });
  });

  describe("activation threshold", () => {
    it("should not start a drag before the pointer moves past the threshold", () => {
      const onStart = vi.fn();
      const onMove = vi.fn();
      const el = renderTarget({ onStart, onMove });
      down(el, { x: 0, y: 0 });
      move({ x: 3, y: 0 });
      move({ x: 0, y: 3 });
      expect(onStart).not.toHaveBeenCalled();
      expect(onMove).not.toHaveBeenCalled();
    });

    it("should start a drag once the pointer moves past the threshold", () => {
      const onStart = vi.fn();
      const el = renderTarget({ onStart });
      down(el, { x: 0, y: 0 });
      move({ x: 5, y: 0 });
      expect(onStart).toHaveBeenCalledTimes(1);
    });

    it("should fire onStart exactly once across subsequent moves", () => {
      const onStart = vi.fn();
      const el = renderTarget({ onStart });
      down(el, { x: 0, y: 0 });
      move({ x: 10, y: 0 });
      move({ x: 20, y: 0 });
      move({ x: 30, y: 0 });
      expect(onStart).toHaveBeenCalledTimes(1);
    });

    it("should report the press location, mouse key, and element to onStart", () => {
      const onStart = vi.fn();
      const el = renderTarget({ onStart });
      down(el, { x: 40, y: 60 });
      move({ x: 60, y: 60 });
      expect(onStart).toHaveBeenCalledWith({ x: 40, y: 60 }, "MouseLeft", el);
    });
  });

  describe("movement", () => {
    it("should report a box spanning the press to the current location", async () => {
      const onMove = vi.fn();
      const el = renderTarget({ onMove });
      down(el, { x: 100, y: 100 });
      move({ x: 150, y: 130 });
      await frame();
      const b = onMove.mock.lastCall?.[0] as box.Box;
      expect(box.left(b)).toEqual(100);
      expect(box.top(b)).toEqual(100);
      expect(box.width(b)).toEqual(50);
      expect(box.height(b)).toEqual(30);
    });

    it("should coalesce moves in the same frame to the latest position", async () => {
      const onMove = vi.fn();
      const el = renderTarget({ onMove });
      down(el, { x: 0, y: 0 });
      move({ x: 10, y: 0 });
      move({ x: 20, y: 0 });
      move({ x: 30, y: 0 });
      await frame();
      expect(onMove).toHaveBeenCalledTimes(1);
      const b = onMove.mock.lastCall?.[0] as box.Box;
      expect(box.width(b)).toEqual(30);
    });

    it("should report the latest move once per frame", async () => {
      const onMove = vi.fn();
      const el = renderTarget({ onMove });
      down(el, { x: 0, y: 0 });
      move({ x: 10, y: 0 });
      await frame();
      move({ x: 20, y: 0 });
      await frame();
      expect(onMove).toHaveBeenCalledTimes(2);
    });

    it("should drop a pending move when the drag ends first", async () => {
      const onMove = vi.fn();
      const el = renderTarget({ onMove });
      down(el, { x: 0, y: 0 });
      move({ x: 50, y: 0 });
      up({ x: 50, y: 0 });
      await frame();
      expect(onMove).not.toHaveBeenCalled();
    });
  });

  describe("pointer identity", () => {
    it("should ignore moves from a different pointer", () => {
      const onStart = vi.fn();
      const onMove = vi.fn();
      const el = renderTarget({ onStart, onMove });
      down(el, { pointerId: 1, x: 0, y: 0 });
      move({ pointerId: 2, x: 100, y: 100 });
      expect(onStart).not.toHaveBeenCalled();
      expect(onMove).not.toHaveBeenCalled();
    });

    it("should ignore an up from a different pointer", () => {
      const onEnd = vi.fn();
      const el = renderTarget({ onEnd });
      down(el, { pointerId: 1, x: 0, y: 0 });
      move({ pointerId: 1, x: 50, y: 0 });
      up({ pointerId: 2, x: 50, y: 0 });
      expect(onEnd).not.toHaveBeenCalled();
    });
  });

  describe("pointer capture", () => {
    it("should capture the pointer to the element when the drag starts", () => {
      const el = renderTarget({});
      down(el, { pointerId: 7, x: 0, y: 0 });
      move({ pointerId: 7, x: 50, y: 0 });
      expect(pointerCapture.setPointerCapture).toHaveBeenCalledWith(7);
      expect(el.hasPointerCapture(7)).toBe(true);
    });

    it("should release the captured pointer when the drag ends", () => {
      const el = renderTarget({});
      down(el, { pointerId: 7, x: 0, y: 0 });
      move({ pointerId: 7, x: 50, y: 0 });
      up({ pointerId: 7, x: 50, y: 0 });
      expect(pointerCapture.releasePointerCapture).toHaveBeenCalledWith(7);
      expect(el.hasPointerCapture(7)).toBe(false);
    });

    it("should not capture the pointer for a press below the threshold", () => {
      const el = renderTarget({});
      down(el, { x: 0, y: 0 });
      move({ x: 2, y: 0 });
      up({ x: 2, y: 0 });
      expect(pointerCapture.setPointerCapture).not.toHaveBeenCalled();
    });
  });

  describe("termination", () => {
    it("should report the final box to onEnd on pointer up", () => {
      const onEnd = vi.fn();
      const el = renderTarget({ onEnd });
      down(el, { x: 100, y: 100 });
      move({ x: 200, y: 100 });
      up({ x: 220, y: 140 });
      expect(onEnd).toHaveBeenCalledTimes(1);
      const b = onEnd.mock.lastCall?.[0] as box.Box;
      expect(box.width(b)).toEqual(120);
      expect(box.height(b)).toEqual(40);
    });

    it("should end the drag on pointer cancel", () => {
      const onEnd = vi.fn();
      const el = renderTarget({ onEnd });
      down(el, { x: 0, y: 0 });
      move({ x: 50, y: 0 });
      cancel({ x: 50, y: 0 });
      expect(onEnd).toHaveBeenCalledTimes(1);
      expect(pointerCapture.releasePointerCapture).toHaveBeenCalledWith(1);
    });

    it("should not fire onEnd for a stationary click", () => {
      const onStart = vi.fn();
      const onEnd = vi.fn();
      const el = renderTarget({ onStart, onEnd });
      down(el, { x: 0, y: 0 });
      up({ x: 0, y: 0 });
      expect(onStart).not.toHaveBeenCalled();
      expect(onEnd).not.toHaveBeenCalled();
    });

    it("should not fire onEnd when the press never passes the threshold", () => {
      const onEnd = vi.fn();
      const el = renderTarget({ onEnd });
      down(el, { x: 0, y: 0 });
      move({ x: 2, y: 0 });
      up({ x: 2, y: 0 });
      expect(onEnd).not.toHaveBeenCalled();
    });
  });

  describe("listener teardown", () => {
    it("should stop reporting moves after the drag ends", () => {
      const onMove = vi.fn();
      const el = renderTarget({ onMove });
      down(el, { x: 0, y: 0 });
      move({ x: 50, y: 0 });
      up({ x: 50, y: 0 });
      onMove.mockClear();
      move({ x: 100, y: 0 });
      expect(onMove).not.toHaveBeenCalled();
    });

    it("should detach window listeners even when the press stays a click", () => {
      const onMove = vi.fn();
      const el = renderTarget({ onMove });
      down(el, { x: 0, y: 0 });
      up({ x: 0, y: 0 });
      move({ x: 100, y: 0 });
      expect(onMove).not.toHaveBeenCalled();
    });
  });
});
