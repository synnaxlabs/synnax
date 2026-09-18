// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { startPinningHashTarget } from "@/util/hash";

class MockResizeObserver {
  static last: MockResizeObserver | undefined;
  callback: ResizeObserverCallback;
  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
    MockResizeObserver.last = this;
  }
  observe = vi.fn();
  disconnect = vi.fn();
  unobserve = vi.fn();
  resize(): void {
    this.callback([], this);
  }
}

const navigate = (navigationType: string): void => {
  const swap = Object.assign(new Event("astro:before-swap"), { navigationType });
  document.dispatchEvent(swap);
  document.dispatchEvent(new Event("astro:page-load"));
};

describe("startPinningHashTarget", () => {
  let scrollIntoView: ReturnType<typeof vi.fn>;
  let stop: () => void;

  beforeEach(() => {
    MockResizeObserver.last = undefined;
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
    const heading = document.createElement("h2");
    heading.id = "read-from-a-range";
    scrollIntoView = vi.fn();
    heading.scrollIntoView = scrollIntoView;
    document.body.appendChild(heading);
    window.location.hash = "#read-from-a-range";
    stop = startPinningHashTarget();
  });

  afterEach(() => {
    stop();
    document.body.replaceChildren();
    vi.unstubAllGlobals();
    vi.useRealTimers();
    window.history.replaceState({}, "", window.location.pathname);
  });

  it("should scroll the hash target back into view when the page resizes", () => {
    navigate("push");
    MockResizeObserver.last?.resize();
    expect(scrollIntoView).toHaveBeenCalledOnce();
  });

  it("should not pin when the url has no hash", () => {
    window.history.replaceState({}, "", window.location.pathname);
    navigate("push");
    expect(MockResizeObserver.last).toBeUndefined();
  });

  it("should not pin when no element matches the hash", () => {
    window.location.hash = "#missing";
    navigate("push");
    expect(MockResizeObserver.last).toBeUndefined();
  });

  it("should leave history traversals alone", () => {
    navigate("traverse");
    expect(MockResizeObserver.last).toBeUndefined();
  });

  it("should stop when the reader scrolls", () => {
    navigate("push");
    window.dispatchEvent(new Event("wheel"));
    expect(MockResizeObserver.last?.disconnect).toHaveBeenCalledOnce();
  });

  it("should stop once the page has settled", () => {
    navigate("push");
    vi.advanceTimersByTime(5000);
    expect(MockResizeObserver.last?.disconnect).toHaveBeenCalledOnce();
  });

  it("should stop when the next navigation starts", () => {
    navigate("push");
    const first = MockResizeObserver.last;
    navigate("push");
    expect(first?.disconnect).toHaveBeenCalledOnce();
    expect(MockResizeObserver.last).not.toBe(first);
  });

  it("should pin on a fresh page load but not on a reload", () => {
    stop();
    const entries = vi.spyOn(performance, "getEntriesByType");
    entries.mockReturnValue([{ type: "reload" } as PerformanceNavigationTiming]);
    stop = startPinningHashTarget();
    expect(MockResizeObserver.last).toBeUndefined();
    stop();
    entries.mockReturnValue([{ type: "navigate" } as PerformanceNavigationTiming]);
    stop = startPinningHashTarget();
    expect(MockResizeObserver.last).toBeDefined();
    entries.mockRestore();
  });

  it("should stop listening for navigations after it is stopped", () => {
    stop();
    navigate("push");
    expect(MockResizeObserver.last).toBeUndefined();
  });
});
