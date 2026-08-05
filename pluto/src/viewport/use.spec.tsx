// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, location, xy } from "@synnaxlabs/x";
import { fireEvent, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement, type RefObject } from "react";
import { afterEach, assert, describe, expect, it, type Mock, vi } from "vitest";

import { mockBoundingClientRect } from "@/testutil/dom";
import { Triggers } from "@/triggers";
import {
  PAN_DEFAULT_TRIGGERS,
  SELECT_DEFAULT_TRIGGERS,
  use,
  type UseEvent,
  type UseHandler,
  type UseProps,
  type UseRefValue,
  type UseReturn,
} from "@/viewport/use";

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>{children}</Triggers.Provider>
);

const CANVAS = box.construct(0, 0, 100, 100);
// Mirrors the hook's default `initial`. Note that it is top-left rooted, unlike the
// bottom-left rooted box.DECIMAL that a zoom reset installs.
const DEFAULT_VIEWPORT = box.construct(0, 0, 1, 1, location.TOP_LEFT);

const SHIFT = "ShiftLeft";
const CONTROL = "ControlLeft";
const ALT = "AltLeft";

const MIDDLE = { button: Triggers.MOUSE_MIDDLE_NUMBER };
const RIGHT = { button: Triggers.MOUSE_RIGHT_NUMBER };

interface Handle {
  canvas: HTMLDivElement;
  measure: Mock<typeof HTMLElement.prototype.getBoundingClientRect>;
  onChange: Mock<UseHandler>;
  result: { current: UseReturn };
  rerender: (props: UseProps) => void;
  ref: RefObject<UseRefValue | undefined>;
}

const canvases: HTMLElement[] = [];

afterEach(() => {
  canvases.forEach((c) => c.remove());
  canvases.length = 0;
});

const setup = (props: UseProps = {}, canvasBox: box.Box = CANVAS): Handle => {
  const onChange = vi.fn<UseHandler>();
  const ref: RefObject<UseRefValue | undefined> = { current: undefined };
  const canvas = document.createElement("div");
  const measure = mockBoundingClientRect(
    box.top(canvasBox),
    box.left(canvasBox),
    box.width(canvasBox),
    box.height(canvasBox),
  );
  canvas.getBoundingClientRect = measure;
  document.body.appendChild(canvas);
  canvases.push(canvas);
  const { result, rerender } = renderHook(
    (p: UseProps) => use({ onChange, ref, ...p }),
    { wrapper: Wrapper, initialProps: props },
  );
  result.current.ref.current = canvas;
  return { canvas, measure, onChange, result, rerender, ref };
};

const lastEvent = (onChange: Mock<UseHandler>): UseEvent => {
  const call = onChange.mock.lastCall;
  assert(call != null, "expected onChange to have been called");
  return call[0];
};

const client = ({ x, y }: xy.XY) => ({ clientX: x, clientY: y });

const press = (canvas: HTMLElement, at: xy.XY, button = Triggers.MOUSE_LEFT_NUMBER) => {
  fireEvent.mouseMove(document.body, client(at));
  fireEvent.mouseDown(canvas, { ...client(at), button });
};

const move = (to: xy.XY) => fireEvent.mouseMove(document.body, client(to));

const release = (canvas: HTMLElement, at: xy.XY, button = Triggers.MOUSE_LEFT_NUMBER) =>
  fireEvent.mouseUp(canvas, { ...client(at), button });

interface DragOptions {
  button?: number;
  keys?: string[];
}

const drag = (
  canvas: HTMLElement,
  from: xy.XY,
  to: xy.XY,
  { button, keys = [] }: DragOptions = {},
) => {
  keys.forEach((code) => fireEvent.keyDown(document.body, { code }));
  press(canvas, from, button);
  move(to);
  release(canvas, to, button);
  keys.toReversed().forEach((code) => fireEvent.keyUp(document.body, { code }));
};

// The hook never returns its viewport. A cancel trigger echoes the current viewport
// back through onChange untouched, which is the only way to read it.
const readViewport = ({ canvas, onChange }: Handle): box.Box => {
  move(box.center(box.construct(canvas)));
  fireEvent.keyDown(document.body, { code: "Escape" });
  fireEvent.keyUp(document.body, { code: "Escape" });
  return lastEvent(onChange).box;
};

describe("viewport/use", () => {
  describe("mode resolution", () => {
    it("should treat a bare left drag as a zoom", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 });
      expect(lastEvent(h.onChange).mode).toEqual("zoom");
    });

    it("should treat a left drag with Shift as a pan", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 }, { keys: [SHIFT] });
      expect(lastEvent(h.onChange).mode).toEqual("pan");
    });

    it("should treat a middle mouse drag as a pan", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 }, MIDDLE);
      expect(lastEvent(h.onChange).mode).toEqual("pan");
    });

    it("should treat a left drag with Alt as a select", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 }, { keys: [ALT] });
      expect(lastEvent(h.onChange).mode).toEqual("select");
    });

    it("should treat a left drag with Control as a zoom reset", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 }, { keys: [CONTROL] });
      expect(lastEvent(h.onChange).mode).toEqual("zoomReset");
    });

    it("should treat Escape as a cancel", () => {
      const h = setup();
      move({ x: 50, y: 50 });
      fireEvent.keyDown(document.body, { code: "Escape" });
      expect(lastEvent(h.onChange).mode).toEqual("cancel");
    });

    it("should commit the mode held at release, not at press", () => {
      const h = setup();
      press(h.canvas, { x: 20, y: 20 });
      move({ x: 80, y: 80 });
      // The in-flight drag keeps previewing the zoom it started as.
      expect(box.areaIsZero(h.result.current.maskBox)).toBe(false);
      fireEvent.keyDown(document.body, { code: SHIFT });
      release(h.canvas, { x: 80, y: 80 });
      fireEvent.keyUp(document.body, { code: SHIFT });
      expect(lastEvent(h.onChange).mode).toEqual("pan");
    });

    it("should use the default mode when no trigger matches", () => {
      const h = setup({ triggers: PAN_DEFAULT_TRIGGERS });
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 });
      expect(lastEvent(h.onChange).mode).toEqual("pan");
    });

    it("should honor a select default trigger set", () => {
      const h = setup({ triggers: SELECT_DEFAULT_TRIGGERS });
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 });
      expect(lastEvent(h.onChange).mode).toEqual("select");
    });
  });

  describe("zoom", () => {
    it("should map the dragged rectangle into the viewport", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 });
      const { box: b, stage } = lastEvent(h.onChange);
      expect(stage).toEqual("end");
      expect(box.left(b)).toBeCloseTo(0.2);
      expect(box.right(b)).toBeCloseTo(0.8);
      expect(box.top(b)).toBeCloseTo(0.2);
      expect(box.bottom(b)).toBeCloseTo(0.8);
      expect(b.root).toEqual(location.TOP_LEFT);
    });

    it("should compose successive zooms onto the previous viewport", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 });
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 });
      const b = lastEvent(h.onChange).box;
      expect(box.left(b)).toBeCloseTo(0.32);
      expect(box.right(b)).toBeCloseTo(0.68);
      expect(box.top(b)).toBeCloseTo(0.32);
      expect(box.bottom(b)).toBeCloseTo(0.68);
    });

    it("should follow the root of the current viewport", () => {
      const h = setup();
      h.ref.current?.reset();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 });
      const b = lastEvent(h.onChange).box;
      expect(b.root).toEqual(location.BOTTOM_LEFT);
      expect(box.top(b)).toBeCloseTo(0.8);
      expect(box.bottom(b)).toBeCloseTo(0.2);
    });

    it("should clear the mask once the zoom commits", () => {
      const h = setup();
      press(h.canvas, { x: 20, y: 20 });
      move({ x: 80, y: 80 });
      expect(box.areaIsZero(h.result.current.maskBox)).toBe(false);
      release(h.canvas, { x: 80, y: 80 });
      expect(box.areaIsZero(h.result.current.maskBox)).toBe(true);
    });
  });

  describe("pan", () => {
    it("should translate the viewport opposite the drag", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 60, y: 50 }, { keys: [SHIFT] });
      const b = lastEvent(h.onChange).box;
      expect(box.left(b)).toBeCloseTo(-0.4);
      expect(box.right(b)).toBeCloseTo(0.6);
      expect(box.top(b)).toBeCloseTo(-0.3);
      expect(box.bottom(b)).toBeCloseTo(0.7);
    });

    it("should never paint a mask", () => {
      const h = setup();
      fireEvent.keyDown(document.body, { code: SHIFT });
      press(h.canvas, { x: 20, y: 20 });
      move({ x: 80, y: 80 });
      expect(box.areaIsZero(h.result.current.maskBox)).toBe(true);
      release(h.canvas, { x: 80, y: 80 });
      fireEvent.keyUp(document.body, { code: SHIFT });
    });

    it("should ignore cursor movement below the translation threshold", () => {
      const h = setup();
      fireEvent.keyDown(document.body, { code: SHIFT });
      press(h.canvas, { x: 50, y: 50 });
      expect(h.onChange).toHaveBeenCalledTimes(1);
      move({ x: 51, y: 50 });
      expect(h.onChange).toHaveBeenCalledTimes(1);
      move({ x: 60, y: 50 });
      expect(h.onChange).toHaveBeenCalledTimes(2);
      expect(lastEvent(h.onChange).stage).toEqual("during");
    });
  });

  describe("select", () => {
    it("should emit the selected region without moving the viewport", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 }, { keys: [ALT] });
      const b = lastEvent(h.onChange).box;
      expect(box.left(b)).toBeCloseTo(0.2);
      expect(box.right(b)).toBeCloseTo(0.8);
      expect(readViewport(h)).toEqual(DEFAULT_VIEWPORT);
    });

    it("should leave the mask up after the selection commits", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 }, { keys: [ALT] });
      expect(box.areaIsZero(h.result.current.maskBox)).toBe(false);
    });

    it("should clear the mask on cancel", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 }, { keys: [ALT] });
      fireEvent.keyDown(document.body, { code: "Escape" });
      expect(box.areaIsZero(h.result.current.maskBox)).toBe(true);
      expect(lastEvent(h.onChange).mode).toEqual("cancel");
    });

    it("should span the full viewport height for a right click", () => {
      const h = setup();
      drag(h.canvas, { x: 50, y: 50 }, { x: 50, y: 50 }, RIGHT);
      const { box: b, mode } = lastEvent(h.onChange);
      expect(mode).toEqual("select");
      expect(box.left(b)).toBeCloseTo(0.5);
      expect(box.right(b)).toBeCloseTo(0.5);
      expect(box.top(b)).toBeCloseTo(0);
      expect(box.bottom(b)).toBeCloseTo(1);
    });

    it("should not commit a right drag that painted a mask", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 }, RIGHT);
      expect(h.onChange).not.toHaveBeenCalled();
      expect(box.areaIsZero(h.result.current.maskBox)).toBe(false);
    });
  });

  describe("click", () => {
    it("should report a sub-threshold left drag as a click", () => {
      const h = setup();
      drag(h.canvas, { x: 50, y: 50 }, { x: 52, y: 52 });
      expect(h.onChange).toHaveBeenCalledTimes(1);
      const e = lastEvent(h.onChange);
      expect(e.mode).toEqual("click");
      expect(e.stage).toEqual("end");
      expect(e.cursor).toEqual({ x: 52, y: 52 });
      expect(e.box).toEqual(DEFAULT_VIEWPORT);
    });

    it("should not paint a mask for a sub-threshold drag", () => {
      const h = setup();
      press(h.canvas, { x: 50, y: 50 });
      move({ x: 52, y: 52 });
      expect(box.areaIsZero(h.result.current.maskBox)).toBe(true);
      release(h.canvas, { x: 52, y: 52 });
    });
  });

  describe("zoom reset", () => {
    it("should restore the decimal viewport", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 });
      drag(h.canvas, { x: 50, y: 50 }, { x: 50, y: 50 }, { keys: [CONTROL] });
      expect(lastEvent(h.onChange).box).toEqual(box.DECIMAL);
      expect(readViewport(h)).toEqual(box.DECIMAL);
    });

    it("should reset through the imperative handle", () => {
      const h = setup();
      drag(h.canvas, { x: 20, y: 20 }, { x: 80, y: 80 });
      h.ref.current?.reset();
      expect(lastEvent(h.onChange)).toEqual({
        box: box.DECIMAL,
        mode: "zoomReset",
        stage: "start",
        cursor: xy.ZERO,
      });
      expect(readViewport(h)).toEqual(box.DECIMAL);
    });
  });

  describe("mask", () => {
    const OFFSET = box.construct(50, 100, 200, 400);

    it("should be relative to the canvas origin", () => {
      const h = setup({}, OFFSET);
      press(h.canvas, { x: 100, y: 200 });
      move({ x: 150, y: 300 });
      const m = h.result.current.maskBox;
      expect(box.left(m)).toEqual(50);
      expect(box.top(m)).toEqual(100);
      expect(box.width(m)).toEqual(50);
      expect(box.height(m)).toEqual(100);
    });

    it("should clamp to the canvas", () => {
      const h = setup({}, OFFSET);
      press(h.canvas, { x: 100, y: 200 });
      move({ x: 400, y: 900 });
      const m = h.result.current.maskBox;
      expect(box.right(m)).toEqual(200);
      expect(box.bottom(m)).toEqual(400);
    });

    it("should snap a short drag to the full canvas height", () => {
      const h = setup();
      press(h.canvas, { x: 20, y: 40 });
      move({ x: 80, y: 60 });
      const m = h.result.current.maskBox;
      expect(box.top(m)).toEqual(0);
      expect(box.bottom(m)).toEqual(100);
      expect(box.left(m)).toEqual(20);
      expect(box.right(m)).toEqual(80);
    });

    it("should snap a narrow drag to the full canvas width", () => {
      const h = setup();
      press(h.canvas, { x: 40, y: 20 });
      move({ x: 60, y: 80 });
      const m = h.result.current.maskBox;
      expect(box.left(m)).toEqual(0);
      expect(box.right(m)).toEqual(100);
      expect(box.top(m)).toEqual(20);
      expect(box.bottom(m)).toEqual(80);
    });

    it("should not snap when the threshold is zero", () => {
      const h = setup({ threshold: { width: 0, height: 0 } });
      press(h.canvas, { x: 20, y: 40 });
      move({ x: 80, y: 60 });
      const m = h.result.current.maskBox;
      expect(box.top(m)).toEqual(40);
      expect(box.bottom(m)).toEqual(60);
    });
  });

  describe("wheel zoom", () => {
    // Regression: the window-level wheel handler measured the canvas before checking
    // that the event targeted it, forcing a layout read per mounted viewport on every
    // wheel event anywhere in the app.
    it("should not measure the canvas for wheel events outside it", () => {
      const h = setup();
      const outside = document.createElement("div");
      document.body.appendChild(outside);
      canvases.push(outside);
      h.measure.mockClear();
      fireEvent.wheel(outside, { deltaY: 1 });
      expect(h.measure).not.toHaveBeenCalled();
      fireEvent.wheel(h.canvas, { deltaY: 1 });
      expect(h.measure).toHaveBeenCalled();
    });

    it("should ignore wheel events outside the canvas bounds", () => {
      const h = setup();
      fireEvent.wheel(h.canvas, { deltaY: 1, clientX: 500, clientY: 500 });
      expect(h.onChange).not.toHaveBeenCalled();
    });

    it("should zoom out around the cursor on a downward wheel", () => {
      const h = setup();
      fireEvent.wheel(h.canvas, { deltaY: 1, clientX: 50, clientY: 50 });
      const e = lastEvent(h.onChange);
      expect(e.mode).toEqual("zoom");
      expect(e.stage).toEqual("end");
      expect(e.cursor).toEqual({ x: 50, y: 50 });
      expect(box.left(e.box)).toBeCloseTo(-0.0175);
      expect(box.right(e.box)).toBeCloseTo(1.0175);
      expect(box.top(e.box)).toBeCloseTo(-0.0175);
      expect(box.bottom(e.box)).toBeCloseTo(1.0175);
    });

    it("should zoom in around the cursor on an upward wheel", () => {
      const h = setup();
      fireEvent.wheel(h.canvas, { deltaY: -1, clientX: 50, clientY: 50 });
      const b = lastEvent(h.onChange).box;
      expect(box.left(b)).toBeCloseTo(0.0175);
      expect(box.right(b)).toBeCloseTo(0.9825);
    });

    it("should anchor the zoom on an off-center cursor", () => {
      const h = setup();
      fireEvent.wheel(h.canvas, { deltaY: 1, clientX: 0, clientY: 0 });
      const b = lastEvent(h.onChange).box;
      expect(box.left(b)).toBeCloseTo(0);
      expect(box.right(b)).toBeCloseTo(1.035);
    });

    it("should hold the x axis fixed while Control is held", () => {
      const h = setup();
      fireEvent.keyDown(document.body, { code: CONTROL });
      fireEvent.wheel(h.canvas, { deltaY: 1, clientX: 50, clientY: 50 });
      fireEvent.keyUp(document.body, { code: CONTROL });
      const b = lastEvent(h.onChange).box;
      expect(box.left(b)).toEqual(0);
      expect(box.right(b)).toEqual(1);
      expect(box.bottom(b)).toBeCloseTo(1.0175);
    });

    it("should hold the y axis fixed while Alt is held", () => {
      const h = setup();
      fireEvent.keyDown(document.body, { code: ALT });
      fireEvent.wheel(h.canvas, { deltaY: 1, clientX: 50, clientY: 50 });
      fireEvent.keyUp(document.body, { code: ALT });
      const b = lastEvent(h.onChange).box;
      expect(box.top(b)).toEqual(0);
      expect(box.bottom(b)).toEqual(1);
      expect(box.right(b)).toBeCloseTo(1.0175);
    });

    it("should accumulate onto the current viewport", () => {
      const h = setup();
      fireEvent.wheel(h.canvas, { deltaY: 1, clientX: 50, clientY: 50 });
      fireEvent.wheel(h.canvas, { deltaY: 1, clientX: 50, clientY: 50 });
      expect(box.width(lastEvent(h.onChange).box)).toBeCloseTo(1.035 * 1.035);
    });
  });

  describe("mode preview", () => {
    it("should report the default mode at rest", () => {
      expect(setup().result.current.mode).toEqual("zoom");
      expect(setup({ triggers: PAN_DEFAULT_TRIGGERS }).result.current.mode).toEqual(
        "pan",
      );
    });

    it.each([
      [SHIFT, "pan"],
      [ALT, "select"],
      [CONTROL, "zoomReset"],
    ])("should preview %s as %s while held", (code, mode) => {
      const h = setup();
      move({ x: 50, y: 50 });
      fireEvent.keyDown(document.body, { code });
      expect(h.result.current.mode).toEqual(mode);
      fireEvent.keyUp(document.body, { code });
      expect(h.result.current.mode).toEqual("zoom");
    });

    it("should not preview a mode while the cursor is outside the canvas", () => {
      const h = setup();
      move({ x: 500, y: 500 });
      fireEvent.keyDown(document.body, { code: SHIFT });
      expect(h.result.current.mode).toEqual("zoom");
      fireEvent.keyUp(document.body, { code: SHIFT });
    });
  });

  describe("initial viewport", () => {
    const INITIAL = box.construct(
      { x: 0.25, y: 0.25 },
      { x: 0.75, y: 0.75 },
      undefined,
      undefined,
      location.BOTTOM_LEFT,
    );

    it("should start from the initial box", () => {
      expect(readViewport(setup({ initial: INITIAL }))).toEqual(INITIAL);
    });

    it("should re-seed the viewport when the initial box changes", () => {
      const h = setup();
      expect(readViewport(h)).toEqual(DEFAULT_VIEWPORT);
      h.rerender({ initial: INITIAL });
      expect(readViewport(h)).toEqual(INITIAL);
    });
  });

  describe("truncation", () => {
    it("should truncate a normal viewport to six places", () => {
      const h = setup({}, box.construct(0, 0, 300, 300));
      drag(h.canvas, { x: 10, y: 10 }, { x: 110, y: 10 }, { keys: [SHIFT] });
      const b = lastEvent(h.onChange).box;
      expect(box.left(b)).toEqual(-0.333333);
      expect(box.right(b)).toEqual(0.666667);
    });

    it("should retain precision for a viewport smaller than a thousandth", () => {
      const initial = box.construct(
        { x: 0, y: 0 },
        { x: 1e-5, y: 1e-5 },
        undefined,
        undefined,
        location.BOTTOM_LEFT,
      );
      const h = setup({ initial });
      drag(h.canvas, { x: 50, y: 50 }, { x: 83, y: 50 }, { keys: [SHIFT] });
      expect(box.left(lastEvent(h.onChange).box)).toBeCloseTo(-3.3e-6, 9);
    });
  });

  it("should not emit before the canvas element is attached", () => {
    const onChange = vi.fn<UseHandler>();
    const detached = document.createElement("div");
    detached.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    document.body.appendChild(detached);
    canvases.push(detached);
    renderHook(() => use({ onChange }), { wrapper: Wrapper });
    drag(detached, { x: 20, y: 20 }, { x: 80, y: 80 });
    expect(onChange).not.toHaveBeenCalled();
  });
});
