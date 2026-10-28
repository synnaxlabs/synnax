// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import reactRenderer from "@astrojs/react/server.js";
import { experimental_AstroContainer as AstroContainer } from "astro/container";
import { JSDOM } from "jsdom";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { start } from "./lostSignal";
import LostSignal from "./LostSignal.astro";

const ORIGIN = "https://docs.synnaxlabs.com";

const render = async (path: string): Promise<string> => {
  const container = await AstroContainer.create();
  container.addServerRenderer({ name: "@astrojs/react", renderer: reactRenderer });
  return await container.renderToString(LostSignal, {
    request: new Request(new URL(path, ORIGIN)),
  });
};

class ResizeObserverStub {
  static last: ResizeObserverStub | undefined;
  disconnected = false;
  constructor(
    readonly callback: (entries: { contentRect: { width: number } }[]) => void,
  ) {
    ResizeObserverStub.last = this;
  }
  observe(): void {}
  disconnect(): void {
    this.disconnected = true;
  }
  resize(width: number): void {
    this.callback([{ contentRect: { width } }]);
  }
}

class IntersectionObserverStub {
  static last: IntersectionObserverStub | undefined;
  constructor(readonly callback: (entries: { isIntersecting: boolean }[]) => void) {
    IntersectionObserverStub.last = this;
  }
  observe(): void {}
  disconnect(): void {}
  intersect(isIntersecting: boolean): void {
    this.callback([{ isIntersecting }]);
  }
}

describe("LostSignal", () => {
  describe("server render", () => {
    const names = async (path: string): Promise<string[]> => {
      const { document } = new JSDOM(await render(path)).window;
      return [...document.querySelectorAll("[data-pathname]")].map((el) =>
        el.textContent.trim(),
      );
    };

    it("should name the series after the decoded path", async () => {
      expect(await names("/reference/chan%20nels")).toEqual([
        "/reference/chan nels",
        "/reference/chan nels",
      ]);
    });

    it("should show a path that does not decode as it is", async () => {
      expect(await names("/reference/%E0%A4%A")).toContain("/reference/%E0%A4%A");
    });
  });

  describe("browser", () => {
    let markup: string;
    let reducedMotion: boolean;
    let window: JSDOM["window"];

    const get = <E extends Element>(selector: string): E => {
      const el = window.document.querySelector<E>(selector);
      if (el == null) throw new Error(`no element matches ${selector}`);
      return el;
    };

    // Left to right, since the plot keeps a tick element while its tick stays in view.
    const tickLabels = (): string[] =>
      [...window.document.querySelectorAll<HTMLElement>(".lost-signal__tick")]
        .sort((a, b) => parseFloat(a.style.left) - parseFloat(b.style.left))
        .map((el) => el.dataset.label ?? "");

    const loss = (): number =>
      parseFloat(
        get<HTMLElement>(".lost-signal").style.getPropertyValue("--lost-signal-loss"),
      );

    const hover = (x: number): void => {
      get(".lost-signal__plot").dispatchEvent(
        new window.MouseEvent("pointermove", { clientX: x }),
      );
    };

    // Opens the plot at the path, the way a reader lands on a missing page.
    const open = (path = "/reference/channles"): void => {
      window = new JSDOM(markup, {
        url: new URL(path, ORIGIN).href,
        pretendToBeVisual: true,
      }).window;
      vi.stubGlobal("document", window.document);
      vi.stubGlobal("location", window.location);
      get(".lost-signal__plot").getBoundingClientRect = () =>
        ({ left: 0, width: 1000 }) as DOMRect;
      start();
    };

    beforeAll(async () => {
      markup = await render("/reference/channles");
    });

    beforeEach(() => {
      reducedMotion = true;
      vi.useFakeTimers({
        toFake: ["Date", "performance", "setTimeout", "clearTimeout"],
      });
      vi.setSystemTime(new Date(2026, 9, 24, 10, 0, 40));
      vi.stubGlobal("ResizeObserver", ResizeObserverStub);
      vi.stubGlobal("IntersectionObserver", IntersectionObserverStub);
      vi.stubGlobal("matchMedia", () => ({ matches: reducedMotion }));
      vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) =>
        setTimeout(() => callback(performance.now()), 16),
      );
      vi.stubGlobal("cancelAnimationFrame", (id: number) => clearTimeout(id));
    });

    afterEach(() => {
      vi.useRealTimers();
      vi.unstubAllGlobals();
    });

    it("should name the series after the path in the address bar", () => {
      open("/reference/chan%20nels");
      expect(get(".lost-signal__path").textContent).toBe("/reference/chan nels");
    });

    it("should mark time ticks on whole steps of the clock", () => {
      open();
      expect(tickLabels()).toEqual([":15", ":30"]);
    });

    it("should fit more ticks into a wider plot", () => {
      open();
      ResizeObserverStub.last?.resize(400);
      expect(tickLabels()).toEqual([":20", ":30", ":40"]);
    });

    it("should label a tick on the minute with the hour and minute", () => {
      vi.setSystemTime(new Date(2026, 9, 24, 10, 1, 10));
      open();
      expect(tickLabels()).toEqual([":45", "10:01"]);
    });

    it("should read the value and time under the pointer", () => {
      open();
      hover(500);
      expect(get<HTMLElement>(".lost-signal__cursor").hidden).toBe(false);
      expect(get("[data-time]").textContent).toBe("10:00:27.000");
      expect(get("[data-value]").textContent).toMatch(/^\d+\.\d{2}$/);
    });

    it("should show no data past the end of the signal", () => {
      open();
      hover(900);
      expect(get("[data-value]").textContent).toBe("No data");
      expect(get(".lost-signal__cursor").classList).toContain(
        "lost-signal__cursor--empty",
      );
    });

    it("should hide the cursor when the pointer leaves", () => {
      open();
      hover(500);
      get(".lost-signal__plot").dispatchEvent(new window.MouseEvent("pointerleave"));
      expect(get<HTMLElement>(".lost-signal__cursor").hidden).toBe(true);
    });

    it("should start paused when the reader prefers reduced motion", () => {
      open();
      expect(get(".lost-signal__live").getAttribute("aria-pressed")).toBe("false");
      const before = loss();
      vi.advanceTimersByTime(2000);
      expect(loss()).toBe(before);
    });

    it("should keep a paused plot still when it resizes", () => {
      open();
      const before = loss();
      vi.advanceTimersByTime(2000);
      ResizeObserverStub.last?.resize(400);
      expect(loss()).toBe(before);
    });

    it("should grow the empty part of the plot while live", () => {
      reducedMotion = false;
      open();
      expect(get(".lost-signal__live").getAttribute("aria-pressed")).toBe("true");
      vi.advanceTimersByTime(2000);
      expect(loss()).toBeCloseTo((20 / 28) * 100, 0);
    });

    it("should stop when the reader pauses it", () => {
      reducedMotion = false;
      open();
      get<HTMLButtonElement>(".lost-signal__live").click();
      expect(get(".lost-signal__live").getAttribute("aria-pressed")).toBe("false");
      const before = loss();
      vi.advanceTimersByTime(2000);
      expect(loss()).toBe(before);
    });

    it("should stop while the plot is out of view", () => {
      reducedMotion = false;
      open();
      IntersectionObserverStub.last?.intersect(false);
      const before = loss();
      vi.advanceTimersByTime(2000);
      expect(loss()).toBe(before);
    });

    it("should stop the previous plot when a page swap replaces it", () => {
      open();
      const previous = ResizeObserverStub.last;
      window.document.body.innerHTML = markup;
      window.document.dispatchEvent(new window.Event("astro:after-swap"));
      expect(previous?.disconnected).toBe(true);
      expect(ResizeObserverStub.last).not.toBe(previous);
    });
  });
});
