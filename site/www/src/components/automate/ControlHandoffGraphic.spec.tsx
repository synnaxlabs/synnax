// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ControlHandoffGraphic } from "@/components/automate/ControlHandoffGraphic";
import { type CanvasRecord, recordCanvas } from "@/testutil";

const LIT = "var(--pluto-gray-l9)";
const PHASE = 4500;

let record: CanvasRecord;
let intersect: IntersectionObserverCallback;
const disconnect = vi.fn();
const resizeDisconnect = vi.fn();

class Intersection {
  constructor(callback: IntersectionObserverCallback) {
    intersect = callback;
  }
  observe = vi.fn();
  disconnect = disconnect;
}

class Resize {
  observe = vi.fn();
  disconnect = resizeDisconnect;
}

const scroll = (isIntersecting: boolean): void =>
  act(() =>
    intersect(
      [{ isIntersecting } as IntersectionObserverEntry],
      {} as IntersectionObserver,
    ),
  );

const advance = (ms: number): void =>
  act(() => {
    vi.advanceTimersByTime(ms);
  });

const lit = (container: HTMLElement): string[] =>
  [...container.querySelectorAll<HTMLElement>(".handoff-bar-label")]
    .filter((label) => label.style.color === LIT)
    .map((label) => label.textContent ?? "");

const highlight = (container: HTMLElement): number =>
  parseFloat(
    container.querySelector<HTMLElement>(".handoff-bar-highlight")!.style.left,
  );

// Averages the red and blue of the nodes drawn in one frame.
const tint = (): { r: number; b: number } => {
  record.fills.length = 0;
  advance(16);
  const rgb = record.fills.map((f) => f.style.match(/\d+/g)!.map(Number));
  const mean = (i: number): number => rgb.reduce((s, c) => s + c[i], 0) / rgb.length;
  return { r: mean(0), b: mean(2) };
};

describe("ControlHandoffGraphic", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["requestAnimationFrame", "cancelAnimationFrame"] });
    record = recordCanvas();
    disconnect.mockClear();
    resizeDisconnect.mockClear();
    vi.stubGlobal("IntersectionObserver", Intersection);
    vi.stubGlobal("ResizeObserver", Resize);
    // jsdom has no layout. Each legend label takes a third of a 300 pixel bar.
    vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(300);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("visibility", () => {
    it("should not draw before it scrolls into view", () => {
      render(<ControlHandoffGraphic />);
      advance(1000);
      expect(record.fills).toEqual([]);
    });

    it("should draw the network once in view", () => {
      render(<ControlHandoffGraphic />);
      scroll(true);
      advance(16);
      expect(record.fills.length).toBeGreaterThan(0);
      expect(record.strokes.length).toBeGreaterThan(0);
    });

    it("should stop drawing when it scrolls out of view", () => {
      render(<ControlHandoffGraphic />);
      scroll(true);
      advance(100);
      scroll(false);
      advance(16);
      const drawn = record.fills.length;
      advance(1000);
      expect(record.fills).toHaveLength(drawn);
    });

    it("should resume drawing when it scrolls back into view", () => {
      render(<ControlHandoffGraphic />);
      scroll(true);
      advance(100);
      scroll(false);
      advance(100);
      const drawn = record.fills.length;
      scroll(true);
      advance(100);
      expect(record.fills.length).toBeGreaterThan(drawn);
    });

    it("should stop drawing and observing once removed", () => {
      const { unmount } = render(<ControlHandoffGraphic />);
      scroll(true);
      advance(100);
      unmount();
      const drawn = record.fills.length;
      advance(1000);
      expect(record.fills).toHaveLength(drawn);
      expect(disconnect).toHaveBeenCalledOnce();
      expect(resizeDisconnect).toHaveBeenCalledOnce();
    });
  });

  describe("legend", () => {
    it("should light automated at the start of the cycle", () => {
      const { container } = render(<ControlHandoffGraphic />);
      scroll(true);
      advance(32);
      expect(lit(container)).toEqual(["automated"]);
      expect(highlight(container)).toBeCloseTo(50, 0);
    });

    it("should hand the light to abort halfway through the first phase", () => {
      const { container } = render(<ControlHandoffGraphic />);
      scroll(true);
      advance(16);
      advance(PHASE / 2 + 100);
      expect(lit(container)).toEqual(["abort"]);
      expect(highlight(container)).toBeGreaterThan(100);
      expect(highlight(container)).toBeLessThan(150);
    });

    it("should hand the light from manual back to automated in the last phase", () => {
      const { container } = render(<ControlHandoffGraphic />);
      scroll(true);
      advance(16);
      advance(2 * PHASE + 1000);
      expect(lit(container)).toEqual(["manual"]);
      advance(PHASE / 2);
      expect(lit(container)).toEqual(["automated"]);
    });

    it("should start the cycle again after the last phase", () => {
      const { container } = render(<ControlHandoffGraphic />);
      scroll(true);
      advance(16);
      advance(3 * PHASE + 100);
      expect(lit(container)).toEqual(["automated"]);
    });
  });

  describe("nodes", () => {
    it("should sweep the nodes from blue to red over the first phase", () => {
      render(<ControlHandoffGraphic />);
      scroll(true);
      const start = tint();
      expect(start.b).toBeGreaterThan(start.r);
      advance(PHASE - 100);
      const end = tint();
      expect(end.r).toBeGreaterThan(end.b);
    });
  });
});
