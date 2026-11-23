// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, onTestFinished, vi } from "vitest";

import { traceResizeLoops } from "@/testutil";

const frames = async (count: number): Promise<void> => {
  for (let i = 0; i < count; i++)
    await new Promise((resolve) => requestAnimationFrame(resolve));
};

const box = (parent: Element, className: string, height: number): HTMLDivElement => {
  const element = document.createElement("div");
  element.className = className;
  element.style.height = `${height}px`;
  parent.append(element);
  return element;
};

const isLoopError = (value: unknown): boolean =>
  value instanceof Error && value.message.startsWith("ResizeObserver loop");

const observe = (element: Element, callback: () => void): void => {
  const observer = new ResizeObserver(callback);
  observer.observe(element);
  onTestFinished(() => observer.disconnect());
};

// Each delivery grows the element, so every frame ends in a loop error.
const grow = (element: HTMLElement): void =>
  observe(element, () => {
    element.style.height = `${element.offsetHeight + 1}px`;
  });

describe("traceResizeLoops", () => {
  let reports: string[];
  let root: HTMLDivElement;

  beforeEach(() => {
    // Vitest logs each loop error that these specs make on purpose.
    const record = console.error;
    const error = vi
      .spyOn(console, "error")
      .mockImplementation((...args: unknown[]) => {
        if (!isLoopError(args[0])) record(...args);
      });
    onTestFinished(() => error.mockRestore());
    reports = [];
    onTestFinished(traceResizeLoops((report) => reports.push(report)));
    root = box(document.body, "root", 100);
    onTestFinished(() => root.remove());
  });

  it("should name the element that a callback resized", async () => {
    const grower = box(root, "grower", 10);
    grower.dataset.testid = "grower-box";
    grow(grower);
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain('div.grower in [data-testid="grower-box"]');
    expect(reports[0]).toContain("inside the callback for div.grower");
    expect(reports[0]).toContain("resizeLoop.browser.spec");
  });

  it("should name an element that a callback resized by under a pixel", async () => {
    // Chromium writes the computed height of a tall element to six digits.
    const element = box(root, "shrinker", 12000.25);
    element.style.width = "10px";
    observe(element, () => {
      element.style.height = `${element.getBoundingClientRect().height - 0.125}px`;
    });
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain(
      "div.shrinker changed from 10x12000.25 to 10x12000.125",
    );
  });

  it("should name an element whose client size a callback changed", async () => {
    const element = box(root, "bordered", 20);
    element.style.boxSizing = "border-box";
    element.style.width = "20px";
    observe(element, () => {
      element.style.border = `${element.clientWidth === 20 ? 2 : 0}px solid`;
    });
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.bordered changed its client size from 20x20");
  });

  it("should name an element observed again after a callback", async () => {
    const above = box(root, "above", 10);
    const deep = box(box(box(root, "outer", 10), "middle", 10), "inner", 5);
    const shallow = new ResizeObserver(() => {});
    shallow.observe(above);
    onTestFinished(() => shallow.disconnect());
    let observedAgain = false;
    // A MutationObserver that observes the elements above a list again does this.
    observe(deep, () => {
      if (observedAgain) return;
      observedAgain = true;
      queueMicrotask(() => {
        shallow.disconnect();
        shallow.observe(above);
      });
    });
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.above began to be observed after a callback");
    expect(reports[0]).toContain("resizeLoop.browser.spec");
  });

  it("should report nothing for a resize outside a callback", async () => {
    const element = box(root, "still", 10);
    observe(element, () => {});
    await frames(2);
    element.style.height = "20px";
    await frames(3);
    expect(reports).toEqual([]);
  });

  it("should leave out a resize from an earlier frame", async () => {
    const outer = box(root, "outer", 50);
    const middle = box(outer, "middle", 10);
    const inner = box(middle, "inner", 5);
    observe(outer, () => {});
    observe(middle, () => {});
    let resized = false;
    // The browser delivers the middle box in its next pass, since the outer box is
    // shallower, so this resize ends in no error.
    observe(inner, () => {
      if (resized) return;
      resized = true;
      middle.style.height = "20px";
    });
    await frames(3);
    expect(reports).toEqual([]);
    grow(box(root, "grower", 10));
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    for (const report of reports) {
      expect(report).toContain("div.grower");
      expect(report).not.toContain("div.middle");
    }
  });
});
