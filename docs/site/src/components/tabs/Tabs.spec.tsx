// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, assert, beforeEach, describe, expect, it, vi } from "vitest";

import { Tabs } from "@/components/tabs/Tabs";

const TABS = [
  { tabKey: "python", name: "Python" },
  { tabKey: "typescript", name: "TypeScript" },
];

class MockResizeObserver {
  static body: MockResizeObserver | undefined;
  callback: ResizeObserverCallback;
  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
  }
  observe = vi.fn((target: Element) => {
    if (target === document.body) MockResizeObserver.body = this;
  });
  disconnect = vi.fn();
  unobserve = vi.fn();
  resize(): void {
    this.callback([], this);
  }
}

describe("Tabs", () => {
  let top: number;
  let scrollBy: ReturnType<typeof vi.fn>;

  const renderTabs = (queryParamKey?: string): void => {
    const { container } = render(
      <Tabs
        tabs={TABS}
        queryParamKey={queryParamKey}
        python={<div>py</div>}
        typescript={<div>ts</div>}
      />,
    );
    const frame = container.querySelector(".pluto-tabs");
    assert(frame != null);
    frame.getBoundingClientRect = () => ({ top }) as DOMRect;
  };

  const hidden = (text: string): boolean =>
    screen.getByText(text).closest("[hidden]") != null;

  const clickTypeScript = (): MockResizeObserver => {
    fireEvent.click(screen.getByText("TypeScript"));
    assert(MockResizeObserver.body != null);
    return MockResizeObserver.body;
  };

  beforeEach(() => {
    top = 100;
    MockResizeObserver.body = undefined;
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
    vi.stubGlobal("ResizeObserver", MockResizeObserver);
    // Scrolling down by y moves the frame's viewport-relative top up by y.
    scrollBy = vi.fn((_: number, y: number) => {
      top -= y;
    });
    vi.stubGlobal("scrollBy", scrollBy);
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    vi.useRealTimers();
    window.history.replaceState({}, "", window.location.pathname);
  });

  it("switches the selected tab and visible panel on click", () => {
    renderTabs();
    expect(hidden("py")).toBe(false);
    expect(hidden("ts")).toBe(true);
    fireEvent.click(screen.getByText("TypeScript"));
    expect(hidden("ts")).toBe(false);
    expect(hidden("py")).toBe(true);
    const tab = screen.getByText("TypeScript").closest('[role="tab"]');
    expect(tab?.getAttribute("aria-selected")).toBe("true");
  });

  it("keeps its tab when the url names one it lacks", () => {
    renderTabs("client");
    fireEvent.click(screen.getByText("TypeScript"));
    const url = new URL(window.location.href);
    url.searchParams.set("client", "console");
    window.history.replaceState({}, "", url.toString());
    act(() => {
      window.dispatchEvent(new CustomEvent("urlchange"));
    });
    expect(hidden("ts")).toBe(false);
    expect(hidden("py")).toBe(true);
  });

  it("scrolls away drift while the page settles", () => {
    renderTabs();
    const observer = clickTypeScript();
    top = 60;
    observer.resize();
    expect(scrollBy).toHaveBeenCalledExactlyOnceWith(0, -40);
    expect(top).toBe(100);
  });

  it("does not scroll when nothing moves", () => {
    renderTabs();
    clickTypeScript().resize();
    expect(scrollBy).not.toHaveBeenCalled();
  });

  it("stops when the reader scrolls", () => {
    renderTabs();
    const observer = clickTypeScript();
    fireEvent.wheel(window);
    expect(observer.disconnect).toHaveBeenCalledOnce();
  });

  it("stops once the page has settled", () => {
    renderTabs();
    const observer = clickTypeScript();
    vi.advanceTimersByTime(1000);
    expect(observer.disconnect).toHaveBeenCalledOnce();
  });

  it("stops on unmount", () => {
    renderTabs();
    const observer = clickTypeScript();
    cleanup();
    expect(observer.disconnect).toHaveBeenCalledOnce();
  });
});
