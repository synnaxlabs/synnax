// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { start, STORAGE_KEY } from "@/components/nav/collapse";

// Mirrors the markup of Page.astro.
const PAGE = `
  <main>
    <nav class="page-nav">
      <div class="page-nav__panel">
        <button data-page-nav-toggle aria-label="Collapse sidebar"></button>
        <div class="reference-tree">
          <div class="page-nav-section-header" role="treeitem" aria-level="1">
            <p class="lyra-text">Channels</p>
          </div>
          <a role="treeitem" aria-level="2" href="#channels">Overview</a>
          <div class="page-nav-section-header" role="treeitem" aria-level="1">
            <p class="lyra-text">Ranges</p>
          </div>
          <a role="treeitem" aria-level="2" href="#ranges">Overview</a>
        </div>
      </div>
      <div class="page-nav__tooltip" id="page-nav-tooltip" hidden>
        <span data-tooltip-label></span><span data-tooltip-keys></span>
      </div>
    </nav>
    <article><p>Text.</p></article>
  </main>
`;

const nav = (): HTMLElement => document.querySelector(".page-nav") as HTMLElement;
const isCollapsed = (doc: Document = document): boolean =>
  doc.documentElement.dataset.pageNavCollapsed != null;
const toggle = (): HTMLElement =>
  document.querySelector("[data-page-nav-toggle]") as HTMLElement;
const sections = (): HTMLElement[] => [
  ...document.querySelectorAll<HTMLElement>(".page-nav-section-header"),
];
const links = (): HTMLAnchorElement[] => [
  ...document.querySelectorAll<HTMLAnchorElement>("a[role='treeitem']"),
];
const article = (): HTMLElement => document.querySelector("article") as HTMLElement;
const tooltip = (): HTMLElement =>
  document.querySelector(".page-nav__tooltip") as HTMLElement;
const tooltipLabel = (): string | null =>
  tooltip().querySelector("[data-tooltip-label]")?.textContent ?? null;
const tooltipKeys = (): HTMLElement =>
  tooltip().querySelector("[data-tooltip-keys]") as HTMLElement;

// A mouse crossing from one element to another.
const cross = (from: Element, to: Element): void => {
  from.dispatchEvent(
    new MouseEvent("pointerout", { bubbles: true, relatedTarget: to }),
  );
  to.dispatchEvent(
    new MouseEvent("pointerover", { bubbles: true, relatedTarget: from }),
  );
};

const press = (init: KeyboardEventInit, target: Element = document.body): boolean =>
  target.dispatchEvent(
    new KeyboardEvent("keydown", { bubbles: true, cancelable: true, ...init }),
  );

const SHORTCUT: KeyboardEventInit = { key: "b", code: "KeyB", metaKey: true };

const collapse = (): void => toggle().click();

describe("collapse", () => {
  beforeAll(() => start());

  beforeEach(() => {
    vi.useFakeTimers();
    localStorage.clear();
    delete document.documentElement.dataset.pageNavCollapsed;
    document.body.innerHTML = PAGE;
    document.dispatchEvent(new Event("astro:after-swap"));
    // jsdom lays nothing out, so every element reads as not shown.
    vi.spyOn(nav(), "getClientRects").mockReturnValue([
      new DOMRect(),
    ] as unknown as DOMRectList);
  });

  afterEach(() => {
    vi.advanceTimersByTime(1000);
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  describe("toggle", () => {
    it("should collapse the sidebar and remember it", () => {
      collapse();
      expect(isCollapsed()).toBe(true);
      expect(toggle().getAttribute("aria-label")).toBe("Expand sidebar");
      expect(toggle().getAttribute("aria-expanded")).toBe("false");
      expect(localStorage.getItem(STORAGE_KEY)).toBe("true");
    });

    it("should expand a collapsed sidebar and remember it", () => {
      collapse();
      toggle().click();
      expect(isCollapsed()).toBe(false);
      expect(toggle().getAttribute("aria-label")).toBe("Collapse sidebar");
      expect(toggle().getAttribute("aria-expanded")).toBe("true");
      expect(localStorage.getItem(STORAGE_KEY)).toBe("false");
    });
  });

  describe("page swap", () => {
    const swapTo = (collapsedBefore: boolean, page = PAGE): Document => {
      if (collapsedBefore) collapse();
      const next = document.implementation.createHTMLDocument();
      next.body.innerHTML = page;
      document.dispatchEvent(
        Object.assign(new Event("astro:before-swap"), { newDocument: next }),
      );
      return next;
    };

    it("should collapse an incoming sidebar before it paints", () => {
      expect(isCollapsed(swapTo(true))).toBe(true);
    });

    it("should leave an incoming sidebar open when not collapsed", () => {
      expect(isCollapsed(swapTo(false))).toBe(false);
    });

    it("should leave a page without the sidebar unmarked", () => {
      expect(isCollapsed(swapTo(true, "<main><article></article></main>"))).toBe(false);
    });

    it("should carry the collapse when the browser blocks storage", () => {
      const blocked = (): never => {
        throw new DOMException("blocked", "SecurityError");
      };
      vi.spyOn(Storage.prototype, "getItem").mockImplementation(blocked);
      vi.spyOn(Storage.prototype, "setItem").mockImplementation(blocked);
      expect(isCollapsed(swapTo(true))).toBe(true);
    });
  });

  describe("shortcut", () => {
    it("should toggle the sidebar on Cmd or Ctrl+B", () => {
      expect(press(SHORTCUT)).toBe(false);
      expect(isCollapsed()).toBe(true);
      press({ ...SHORTCUT, metaKey: false, ctrlKey: true });
      expect(isCollapsed()).toBe(false);
    });

    it("should leave the key alone in a text field", () => {
      const input = document.createElement("input");
      article().append(input);
      expect(press(SHORTCUT, input)).toBe(true);
      expect(isCollapsed()).toBe(false);
    });

    it("should leave the key alone when the sidebar is not shown", () => {
      vi.spyOn(nav(), "getClientRects").mockReturnValue([] as unknown as DOMRectList);
      expect(press(SHORTCUT)).toBe(true);
      expect(isCollapsed()).toBe(false);
    });

    it("should ignore B with another modifier", () => {
      press({ ...SHORTCUT, shiftKey: true });
      expect(isCollapsed()).toBe(false);
    });
  });

  describe("section icons", () => {
    it("should take focus only while collapsed", () => {
      expect(sections().map((s) => s.tabIndex)).toEqual([-1, -1]);
      collapse();
      expect(sections().map((s) => s.tabIndex)).toEqual([0, 0]);
    });

    it("should open the first page of the section when collapsed", () => {
      const open = vi.fn((e: Event) => e.preventDefault());
      links()[1].addEventListener("click", open);
      collapse();
      sections()[1].click();
      expect(open).toHaveBeenCalledOnce();
    });

    it("should open the first page on Enter", () => {
      const open = vi.fn((e: Event) => e.preventDefault());
      links()[0].addEventListener("click", open);
      collapse();
      press({ key: "Enter" }, sections()[0]);
      expect(open).toHaveBeenCalledOnce();
    });

    it("should not open a page when expanded", () => {
      const open = vi.fn((e: Event) => e.preventDefault());
      links()[0].addEventListener("click", open);
      sections()[0].click();
      expect(open).not.toHaveBeenCalled();
    });
  });

  describe("tooltip", () => {
    it("should show the toggle action and shortcut after the hover delay", () => {
      cross(article(), toggle());
      vi.advanceTimersByTime(699);
      expect(tooltip().hidden).toBe(true);
      vi.advanceTimersByTime(1);
      expect(tooltip().hidden).toBe(false);
      expect(tooltipLabel()).toBe("Collapse sidebar");
      expect(tooltipKeys().hidden).toBe(false);
      expect(toggle().getAttribute("aria-describedby")).toBe("page-nav-tooltip");
    });

    it("should name a collapsed section without the shortcut", () => {
      collapse();
      cross(article(), sections()[1]);
      vi.advanceTimersByTime(700);
      expect(tooltipLabel()).toBe("Ranges");
      expect(tooltipKeys().hidden).toBe(true);
    });

    it("should not show for a section of the expanded sidebar", () => {
      cross(article(), sections()[0]);
      vi.advanceTimersByTime(700);
      expect(tooltip().hidden).toBe(true);
    });

    it("should close after the pointer leaves", () => {
      cross(article(), toggle());
      vi.advanceTimersByTime(700);
      cross(toggle(), article());
      vi.advanceTimersByTime(150);
      expect(tooltip().hidden).toBe(true);
      expect(toggle().hasAttribute("aria-describedby")).toBe(false);
    });

    it("should not show when the pointer passes over", () => {
      cross(article(), toggle());
      cross(toggle(), article());
      vi.advanceTimersByTime(700);
      expect(tooltip().hidden).toBe(true);
    });

    it("should move between icons without a delay", () => {
      collapse();
      cross(article(), sections()[0]);
      vi.advanceTimersByTime(700);
      cross(sections()[0], sections()[1]);
      expect(tooltip().hidden).toBe(false);
      expect(tooltipLabel()).toBe("Ranges");
    });

    it("should close on Escape", () => {
      cross(article(), toggle());
      vi.advanceTimersByTime(700);
      press({ key: "Escape" });
      expect(tooltip().hidden).toBe(true);
    });

    it("should ignore touch", () => {
      // jsdom has no PointerEvent, so a mouse event carries the pointer type.
      const tap = new MouseEvent("pointerover", { bubbles: true });
      Object.defineProperty(tap, "pointerType", { value: "touch" });
      toggle().dispatchEvent(tap);
      vi.advanceTimersByTime(700);
      expect(tooltip().hidden).toBe(true);
    });
  });
});
