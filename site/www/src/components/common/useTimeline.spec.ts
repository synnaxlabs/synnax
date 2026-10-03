// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useTimeline } from "@/components/common/useTimeline";

const EXAMPLES = [
  { steps: [{ duration: 100 }, { duration: 200 }] },
  { steps: [{ duration: 300 }, { duration: 400 }, { duration: 500 }] },
];

const advance = (ms: number): void =>
  act(() => {
    vi.advanceTimersByTime(ms);
  });

describe("useTimeline", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe("playback", () => {
    it("should start on the first step of the first example", () => {
      const { result } = renderHook(() => useTimeline(EXAMPLES));
      expect(result.current.activeTab).toBe(0);
      expect(result.current.stepIndex).toBe(0);
      expect(result.current.step).toBe(EXAMPLES[0].steps[0]);
    });

    it("should hold each step for its duration", () => {
      const { result } = renderHook(() => useTimeline(EXAMPLES));
      advance(99);
      expect(result.current.stepIndex).toBe(0);
      advance(1);
      expect(result.current.stepIndex).toBe(1);
      advance(199);
      expect(result.current.stepIndex).toBe(1);
    });

    it("should loop to the first step after the last one", () => {
      const { result } = renderHook(() => useTimeline(EXAMPLES));
      advance(100);
      advance(200);
      expect(result.current.stepIndex).toBe(0);
    });
  });

  describe("tab selection", () => {
    it("should start the selected example from its first step", () => {
      const { result } = renderHook(() => useTimeline(EXAMPLES));
      advance(100);
      act(() => result.current.selectTab(1));
      expect(result.current.activeTab).toBe(1);
      expect(result.current.stepIndex).toBe(0);
      expect(result.current.step).toBe(EXAMPLES[1].steps[0]);
      advance(300);
      expect(result.current.stepIndex).toBe(1);
    });

    it("should keep the current step when the active tab is selected again", () => {
      const { result } = renderHook(() => useTimeline(EXAMPLES));
      advance(100);
      act(() => result.current.selectTab(0));
      expect(result.current.stepIndex).toBe(1);
    });
  });

  describe("hover", () => {
    it("should pause while the pointer is over the container", () => {
      const { result } = renderHook(() => useTimeline(EXAMPLES));
      act(() => result.current.containerProps.onMouseEnter());
      advance(1000);
      expect(result.current.stepIndex).toBe(0);
      expect(result.current.containerProps.style).toEqual({ "--play-state": "paused" });
    });

    it("should restart the current step when the pointer leaves", () => {
      const { result } = renderHook(() => useTimeline(EXAMPLES));
      advance(50);
      act(() => result.current.containerProps.onMouseEnter());
      const { playKey } = result.current;
      act(() => result.current.containerProps.onMouseLeave());
      expect(result.current.playKey).not.toBe(playKey);
      expect(result.current.containerProps.style).toEqual({
        "--play-state": "running",
      });
      advance(99);
      expect(result.current.stepIndex).toBe(0);
      advance(1);
      expect(result.current.stepIndex).toBe(1);
    });

    it("should stay paused when a tab is selected under the pointer", () => {
      const { result } = renderHook(() => useTimeline(EXAMPLES));
      act(() => result.current.containerProps.onMouseEnter());
      act(() => result.current.selectTab(1));
      advance(1000);
      expect(result.current.activeTab).toBe(1);
      expect(result.current.stepIndex).toBe(0);
    });
  });
});
