// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ControlHandoffGraphic } from "@/components/automate/ControlHandoffGraphic";
import { recordCanvas } from "@/testutil";

const LIT = "var(--pluto-gray-l9)";
const PHASE = 4500;

const lit = (container: HTMLElement): string[] =>
  [...container.querySelectorAll<HTMLElement>(".handoff-bar-label")]
    .filter((label) => label.style.color === LIT)
    .map((label) => label.textContent ?? "");

const highlight = (container: HTMLElement): number =>
  parseFloat(
    container.querySelector<HTMLElement>(".handoff-bar-highlight")!.style.left,
  );

// Stubs the observers and layout that jsdom lacks, records the canvas, and renders the
// graphic. Each legend label takes a third of a 300 pixel bar. The animation runs only
// when a test calls frame, so a test can jump to any time in one frame.
const setup = () => {
  const intersection = {
    callback: (() => {}) as IntersectionObserverCallback,
    disconnect: vi.fn(),
  };
  const resize = { disconnect: vi.fn() };
  const animation = { next: null as FrameRequestCallback | null };
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      constructor(callback: IntersectionObserverCallback) {
        intersection.callback = callback;
      }
      observe = vi.fn();
      disconnect = intersection.disconnect;
    },
  );
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe = vi.fn();
      disconnect = resize.disconnect;
    },
  );
  vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
    animation.next = callback;
    return 1;
  });
  vi.stubGlobal("cancelAnimationFrame", () => {
    animation.next = null;
  });
  vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(300);
  const record = recordCanvas();
  const { container, unmount } = render(<ControlHandoffGraphic />);
  const scroll = (isIntersecting: boolean): void =>
    act(() =>
      intersection.callback(
        [{ isIntersecting } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      ),
    );
  // Runs the requested frame, if any, at the given time in milliseconds.
  const frame = (at: number): void =>
    act(() => {
      const next = animation.next;
      animation.next = null;
      next?.(at);
    });
  // Averages the red and blue of the nodes drawn in a frame at the given time.
  const tint = (at: number): { r: number; b: number } => {
    record.fills.length = 0;
    frame(at);
    const rgb = record.fills.map((f) => f.style.match(/\d+/g)!.map(Number));
    const mean = (i: number): number => rgb.reduce((s, c) => s + c[i], 0) / rgb.length;
    return { r: mean(0), b: mean(2) };
  };
  return { container, unmount, record, scroll, frame, tint, intersection, resize };
};

describe("ControlHandoffGraphic", () => {
  describe("visibility", () => {
    it("should not draw before it scrolls into view", () => {
      const { record, frame } = setup();
      frame(0);
      frame(1000);
      expect(record.fills).toEqual([]);
    });

    it("should draw the network once in view", () => {
      const { record, scroll, frame } = setup();
      scroll(true);
      frame(0);
      expect(record.fills.length).toBeGreaterThan(0);
      expect(record.strokes.length).toBeGreaterThan(0);
    });

    it("should stop drawing when it scrolls out of view", () => {
      const { record, scroll, frame } = setup();
      scroll(true);
      frame(0);
      scroll(false);
      const drawn = record.fills.length;
      frame(16);
      frame(1000);
      expect(record.fills).toHaveLength(drawn);
    });

    it("should resume drawing when it scrolls back into view", () => {
      const { record, scroll, frame } = setup();
      scroll(true);
      frame(0);
      scroll(false);
      frame(16);
      const drawn = record.fills.length;
      scroll(true);
      frame(100);
      expect(record.fills.length).toBeGreaterThan(drawn);
    });

    it("should stop drawing and observing once removed", () => {
      const { unmount, record, scroll, frame, intersection, resize } = setup();
      scroll(true);
      frame(0);
      unmount();
      const drawn = record.fills.length;
      frame(1000);
      expect(record.fills).toHaveLength(drawn);
      expect(intersection.disconnect).toHaveBeenCalledOnce();
      expect(resize.disconnect).toHaveBeenCalledOnce();
    });
  });

  describe("legend", () => {
    it("should light automated at the start of the cycle", () => {
      const { container, scroll, frame } = setup();
      scroll(true);
      frame(0);
      expect(lit(container)).toEqual(["automated"]);
      expect(highlight(container)).toBeCloseTo(50, 0);
    });

    it("should hand the light to abort halfway through the first phase", () => {
      const { container, scroll, frame } = setup();
      scroll(true);
      frame(0);
      frame(PHASE / 2 + 100);
      expect(lit(container)).toEqual(["abort"]);
      expect(highlight(container)).toBeGreaterThan(100);
      expect(highlight(container)).toBeLessThan(150);
    });

    it("should hand the light from manual back to automated in the last phase", () => {
      const { container, scroll, frame } = setup();
      scroll(true);
      frame(0);
      frame(2 * PHASE + 1000);
      expect(lit(container)).toEqual(["manual"]);
      frame(2 * PHASE + 1000 + PHASE / 2);
      expect(lit(container)).toEqual(["automated"]);
    });

    it("should start the cycle again after the last phase", () => {
      const { container, scroll, frame } = setup();
      scroll(true);
      frame(0);
      frame(3 * PHASE + 100);
      expect(lit(container)).toEqual(["automated"]);
    });
  });

  describe("nodes", () => {
    it("should sweep the nodes from blue to red over the first phase", () => {
      const { scroll, tint } = setup();
      scroll(true);
      const start = tint(0);
      expect(start.b).toBeGreaterThan(start.r);
      const end = tint(PHASE - 100);
      expect(end.r).toBeGreaterThan(end.b);
    });
  });
});
