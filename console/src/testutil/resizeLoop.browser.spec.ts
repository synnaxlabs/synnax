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

const box = (parent: ParentNode, className: string, height: number): HTMLDivElement => {
  const element = document.createElement("div");
  element.className = className;
  element.style.height = `${height}px`;
  parent.append(element);
  return element;
};

// Nests levels boxes under the parent and returns the deepest.
const nest = (parent: ParentNode, levels: number): HTMLDivElement => {
  let element = box(parent, "level", 10);
  for (let i = 1; i < levels; i++) element = box(element, "level", 10);
  return element;
};

const elementDepth = (element: Element): number => {
  let d = 0;
  for (let e: Element | null = element; e != null; e = e.parentElement) d++;
  return d;
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
    // Each size is exact in six digits and in 1/64 of a pixel.
    const element = box(root, "shrinker", 100.25);
    element.style.width = "10px";
    observe(element, () => {
      element.style.height = `${element.getBoundingClientRect().height - 0.125}px`;
    });
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.shrinker changed from 10x100.25 to 10x100.125");
  });

  it("should name an element whose content size a callback changed", async () => {
    const element = box(root, "bordered", 20);
    element.style.boxSizing = "border-box";
    element.style.width = "20px";
    observe(element, () => {
      element.style.border = `${element.clientWidth === 20 ? 2 : 0}px solid`;
    });
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.bordered changed its content size from 20x20");
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

  it("should name an element whose padding a callback changed", async () => {
    const element = box(root, "padded", 20);
    element.style.boxSizing = "border-box";
    element.style.width = "20px";
    observe(element, () => {
      const padded = element.style.padding === "2px";
      element.style.padding = padded ? "0px" : "2px";
    });
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toMatch(
      /div\.padded changed its content size from (20x20|16x16)/,
    );
  });

  it("should leave out a deep element observed after a callback", async () => {
    grow(box(root, "grower", 10));
    const deep = box(box(root, "parent", 20), "deep", 10);
    const deeper = box(box(deep, "below", 10), "deeper", 5);
    const late = new ResizeObserver(() => {});
    onTestFinished(() => late.disconnect());
    let armed = false;
    observe(deep, () => {
      if (!armed) return;
      armed = false;
      queueMicrotask(() => late.observe(deeper));
    });
    await frames(2);
    armed = true;
    deep.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    for (const report of reports) expect(report).not.toContain("div.deeper");
  });

  it("should name an observed element that a callback took off the page", async () => {
    const leaver = box(root, "leaver", 10);
    observe(leaver, () => {});
    const deep = box(box(root, "parent", 20), "deep", 10);
    let armed = false;
    observe(deep, () => {
      if (!armed) return;
      armed = false;
      leaver.remove();
    });
    await frames(2);
    armed = true;
    deep.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.leaver left the page inside the callback for");
  });

  it("should name an element that a callback put back on the page", async () => {
    const returner = box(root, "returner", 10);
    observe(returner, () => {});
    const deep = box(nest(root, 3), "deep", 10);
    let armed = false;
    observe(deep, () => {
      if (!armed) return;
      armed = false;
      root.append(returner);
    });
    await frames(2);
    // The browser reports a zero size for the element off the page.
    returner.remove();
    await frames(2);
    armed = true;
    deep.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain(
      "div.returner came back on the page inside the callback for",
    );
  });

  it("should leave out an element that leaves the page with its observer", async () => {
    const subtree = box(root, "subtree", 10);
    const child = box(subtree, "child", 10);
    const watcher = new ResizeObserver(() => {});
    watcher.observe(child);
    onTestFinished(() => watcher.disconnect());
    const deep = box(box(root, "parent", 20), "deep", 10);
    let armed = false;
    observe(deep, () => {
      if (!armed) return;
      armed = false;
      subtree.remove();
      watcher.disconnect();
    });
    grow(box(root, "grower", 10));
    await frames(2);
    armed = true;
    deep.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    for (const report of reports) expect(report).not.toContain("left the page");
  });

  it("should blame only the callback that resized an element", async () => {
    const shallow = box(root, "shallow", 10);
    observe(shallow, () => {});
    const first = box(box(root, "parent", 20), "first", 10);
    const second = box(box(root, "parent", 20), "second", 10);
    let armed = false;
    observe(first, () => {});
    observe(second, () => {
      if (!armed) return;
      armed = false;
      shallow.style.height = "20px";
    });
    await frames(2);
    armed = true;
    first.style.height = "15px";
    second.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.shallow changed");
    expect(reports[0]).not.toContain("callback for div.first");
  });

  it("should name an element resized long after a callback", async () => {
    const late = box(root, "late", 10);
    observe(late, () => {});
    const deep = box(box(root, "parent", 20), "deep", 10);
    let armed = false;
    observe(deep, () => {
      if (!armed) return;
      armed = false;
      let hops = 10;
      const hop = (): void => {
        hops -= 1;
        if (hops > 0) queueMicrotask(hop);
        else late.style.height = "20px";
      };
      queueMicrotask(hop);
    });
    await frames(2);
    armed = true;
    deep.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toMatch(
      /div\.late changed from \d+x10 to \d+x20 later in the frame of the callback/,
    );
  });

  it("should leave out a new target below the shallowest of its pass", async () => {
    const shallow = box(nest(root, 1), "shallow", 10);
    observe(shallow, () => {});
    const first = box(nest(root, 1), "first", 10);
    const second = box(nest(root, 6), "second", 10);
    const target = box(nest(root, 3), "target", 10);
    const late = new ResizeObserver(() => {});
    onTestFinished(() => late.disconnect());
    let armed = false;
    observe(first, () => {});
    observe(second, () => {
      if (!armed) return;
      armed = false;
      shallow.style.height = "20px";
      queueMicrotask(() => late.observe(target));
    });
    await frames(2);
    armed = true;
    first.style.height = "15px";
    second.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.shallow changed");
    expect(reports[0]).not.toContain("div.target");
  });

  it("should leave out a resize that the next pass delivers", async () => {
    const shallow = box(nest(root, 1), "shallow", 10);
    observe(shallow, () => {});
    const middle = box(nest(root, 3), "middle", 10);
    observe(middle, () => {});
    const first = box(nest(root, 6), "first", 10);
    const second = box(nest(root, 1), "second", 10);
    let armed = false;
    observe(first, () => {
      if (armed) middle.style.height = "20px";
    });
    observe(second, () => {
      if (!armed) return;
      armed = false;
      shallow.style.height = "20px";
    });
    await frames(2);
    armed = true;
    first.style.height = "15px";
    second.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.shallow changed");
    expect(reports[0]).not.toContain("div.middle");
  });

  it("should name a target observed again only with a new box", async () => {
    const same = box(root, "same", 10);
    const other = box(root, "other", 10);
    const watcher = new ResizeObserver(() => {});
    watcher.observe(same);
    watcher.observe(other);
    onTestFinished(() => watcher.disconnect());
    const deep = box(nest(root, 3), "deep", 10);
    let armed = false;
    observe(deep, () => {
      if (!armed) return;
      armed = false;
      watcher.observe(same);
      watcher.observe(other, { box: "border-box" });
    });
    await frames(2);
    armed = true;
    deep.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.other began to be observed after a callback");
    expect(reports[0]).not.toContain("div.same");
  });

  it("should leave out padding under an observed border box", async () => {
    grow(box(root, "grower", 10));
    const element = box(root, "boxed", 20);
    element.style.boxSizing = "border-box";
    const watcher = new ResizeObserver(() => {});
    watcher.observe(element, { box: "border-box" });
    onTestFinished(() => watcher.disconnect());
    const deep = box(nest(root, 3), "deep", 10);
    let armed = false;
    observe(deep, () => {
      if (!armed) return;
      armed = false;
      element.style.padding = "2px";
    });
    await frames(2);
    armed = true;
    deep.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    for (const report of reports) expect(report).not.toContain("div.boxed");
  });

  it("should count an element off the page from its own depth", async () => {
    const shallow = box(root, "shallow", 10);
    observe(shallow, () => {});
    const deep = box(nest(root, 3), "deep", 10);
    const subtree = box(root, "subtree", 10);
    // Off the page, the leaf sits one below the target.
    const leaf = box(nest(subtree, elementDepth(deep) - 1), "leaf", 10);
    const watcher = new ResizeObserver(() => {});
    watcher.observe(leaf);
    onTestFinished(() => watcher.disconnect());
    let armed = false;
    observe(deep, () => {
      if (!armed) return;
      armed = false;
      shallow.style.height = "20px";
      subtree.remove();
    });
    await frames(2);
    armed = true;
    deep.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.shallow changed");
    expect(reports[0]).not.toContain("div.leaf");
  });

  it("should keep a cause that many deep changes follow", async () => {
    const shallow = box(nest(root, 1), "shallow", 10);
    observe(shallow, () => {});
    const deep = Array.from({ length: 25 }, () => box(nest(root, 6), "deep", 10));
    const watcher = new ResizeObserver(() => {});
    for (const element of deep) watcher.observe(element);
    onTestFinished(() => watcher.disconnect());
    const trigger = box(nest(root, 3), "trigger", 10);
    let armed = false;
    observe(trigger, () => {
      if (!armed) return;
      armed = false;
      shallow.style.height = "20px";
      for (const element of deep) element.style.height = "20px";
    });
    await frames(2);
    armed = true;
    trigger.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.shallow changed");
    expect(reports[0]).not.toContain("div.deep");
  });

  it("should start a pass at an observer created after the last one", async () => {
    const cause = box(nest(root, 3), "cause", 10);
    observe(cause, () => {});
    const first = box(nest(root, 1), "first", 10);
    const second = box(nest(root, 6), "second", 10);
    let armed = false;
    // The browser delivers the second box in the next pass, as the first is shallower.
    observe(first, () => {
      if (armed) second.style.height = "20px";
    });
    observe(second, () => {
      if (!armed) return;
      armed = false;
      cause.style.height = "20px";
    });
    await frames(2);
    armed = true;
    first.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.cause changed");
  });

  // Only WebKit joins the second call of the target observer to the pass of the first.
  const placeByReported = async (parent: HTMLElement): Promise<void> => {
    const shallow = box(parent, "shallow", 10);
    observe(shallow, () => {});
    const first = box(nest(parent, 1), "first", 10);
    const target = box(nest(parent, 6), "target", 10);
    const middle = box(nest(parent, 3), "middle", 10);
    let armed = false;
    let firstCalls = 0;
    observe(first, () => {
      if (!armed) return;
      firstCalls += 1;
      target.style.height = `${20 + 10 * firstCalls}px`;
      if (firstCalls === 1) first.style.height = "25px";
    });
    let targetCalls = 0;
    let skippedAt: CSSNumberish | null = null;
    observe(target, () => {
      if (!armed) return;
      targetCalls += 1;
      if (targetCalls !== 2) return;
      skippedAt = document.timeline.currentTime;
      shallow.style.height = "20px";
      middle.style.height = "20px";
    });
    let middleAt: CSSNumberish | null = null;
    observe(middle, () => {
      if (armed) middleAt = document.timeline.currentTime;
    });
    await frames(2);
    armed = true;
    shallow.style.height = "15px";
    first.style.height = "15px";
    target.style.height = "20px";
    await frames(3);
    armed = false;
    // WebKit reports the size from the gathering of a pass, so the second call of the
    // target observer joins the pass of the first and the browser delivers the middle
    // box. Chromium reports the size at the call, so the second call starts a pass that
    // skips it.
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.shallow changed");
    const skipped = middleAt !== skippedAt;
    expect(reports[0].includes("div.middle changed")).toBe(skipped);
  };

  it("should place an observer by the size the browser reported to it", async () => {
    await placeByReported(root);
  });

  it("should place an observer under a scaled ancestor", async () => {
    const scaled = box(root, "scaled", 100);
    scaled.style.transform = "scale(2)";
    await placeByReported(scaled);
  });

  it("should leave out an element whose ancestor a callback scaled", async () => {
    const shallow = box(root, "shallow", 10);
    observe(shallow, () => {});
    const scaled = box(root, "scaled", 10);
    observe(box(scaled, "inner", 10), () => {});
    const deep = box(nest(root, 3), "deep", 10);
    let armed = false;
    observe(deep, () => {
      if (!armed) return;
      armed = false;
      scaled.style.transform = "scale(2)";
      shallow.style.height = "20px";
    });
    await frames(2);
    armed = true;
    deep.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.shallow changed");
    expect(reports[0]).not.toContain("div.inner");
  });

  it("should count depth through shadow roots and slots", async () => {
    const shallow = box(root, "shallow", 10);
    observe(shallow, () => {});
    const shadow = box(root, "host", 10).attachShadow({ mode: "open" });
    const hidden = box(nest(shadow, 4), "hidden", 10);
    const slotHost = box(root, "slot-host", 10);
    nest(slotHost.attachShadow({ mode: "open" }), 4).append(
      document.createElement("slot"),
    );
    const slotted = box(slotHost, "slotted", 10);
    const watcher = new ResizeObserver(() => {});
    watcher.observe(hidden);
    watcher.observe(slotted);
    onTestFinished(() => watcher.disconnect());
    const trigger = box(nest(root, 3), "trigger", 10);
    let armed = false;
    // In the flat tree, both elements sit below the trigger.
    observe(trigger, () => {
      if (!armed) return;
      armed = false;
      shallow.style.height = "20px";
      hidden.style.height = "20px";
      slotted.style.height = "20px";
    });
    await frames(2);
    armed = true;
    trigger.style.height = "15px";
    await frames(3);
    expect(reports.length).toBeGreaterThan(0);
    expect(reports[0]).toContain("div.shallow changed");
    expect(reports[0]).not.toContain("div.hidden");
    expect(reports[0]).not.toContain("div.slotted");
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
