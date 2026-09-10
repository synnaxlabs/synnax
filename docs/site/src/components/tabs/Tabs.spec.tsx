// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, assert, beforeEach, describe, expect, it, vi } from "vitest";

import { Tabs } from "@/components/tabs/Tabs";

const TABS = [
  { tabKey: "python", name: "Python" },
  { tabKey: "typescript", name: "TypeScript" },
];

class MockResizeObserver {
  observe = vi.fn();
  disconnect = vi.fn();
  unobserve = vi.fn();
}

describe("Tabs", () => {
  let frames: Map<number, FrameRequestCallback>;
  let nextFrame: number;
  let top: number;
  let scrollBy: ReturnType<typeof vi.fn>;

  const flushFrame = (): void => {
    const pending = [...frames.values()];
    frames.clear();
    pending.forEach((cb) => cb(0));
  };

  const renderTabs = (): void => {
    const { container } = render(
      <Tabs tabs={TABS} python={<div>py</div>} typescript={<div>ts</div>} />,
    );
    const frame = container.querySelector(".pluto-tabs");
    assert(frame != null);
    frame.getBoundingClientRect = () => ({ top }) as DOMRect;
  };

  beforeEach(() => {
    frames = new Map();
    nextFrame = 1;
    top = 100;
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
    vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
      frames.set(nextFrame, cb);
      return nextFrame++;
    });
    vi.stubGlobal("cancelAnimationFrame", (id: number) => frames.delete(id));
    // Scrolling down by y moves the frame's viewport-relative top up by y.
    scrollBy = vi.fn((_: number, y: number) => {
      top -= y;
    });
    vi.stubGlobal("scrollBy", scrollBy);
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("switches the selected tab and visible panel on click", () => {
    renderTabs();
    expect(screen.getByText("py")).toBeDefined();
    expect(screen.queryByText("ts")).toBeNull();
    fireEvent.click(screen.getByText("TypeScript"));
    expect(screen.getByText("ts")).toBeDefined();
    expect(screen.queryByText("py")).toBeNull();
    const tab = screen.getByText("TypeScript").closest('[role="tab"]');
    expect(tab?.getAttribute("aria-selected")).toBe("true");
  });

  it("scrolls away movement that lands before the first frame", () => {
    renderTabs();
    fireEvent.click(screen.getByText("TypeScript"));
    top = 60;
    flushFrame();
    expect(scrollBy).toHaveBeenCalledExactlyOnceWith(0, -40);
    expect(top).toBe(100);
    flushFrame();
    expect(scrollBy).toHaveBeenCalledTimes(1);
  });

  it("scrolls away movement that lands between frames", () => {
    renderTabs();
    fireEvent.click(screen.getByText("TypeScript"));
    flushFrame();
    top = 130;
    flushFrame();
    expect(scrollBy).toHaveBeenCalledExactlyOnceWith(0, 30);
    expect(top).toBe(100);
  });

  it("does not scroll when nothing moves", () => {
    renderTabs();
    fireEvent.click(screen.getByText("TypeScript"));
    flushFrame();
    flushFrame();
    expect(scrollBy).not.toHaveBeenCalled();
  });
});
