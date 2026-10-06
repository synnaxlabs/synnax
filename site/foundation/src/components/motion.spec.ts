// Copyright 2026 Synnax Labs, Inc. Licensed under licenses/BSL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { observeMotion } from "@/components/motion";

const cleanups: (() => void)[] = [];

const setup = () => {
  document.body.innerHTML = `
    <svg data-viewport-motion data-motion="paused"></svg>
    <section data-viewport-motion data-motion="paused"></section>
  `;
  const elements = Array.from(
    document.querySelectorAll<HTMLElement | SVGElement>("[data-viewport-motion]"),
  );
  let intersect: IntersectionObserverCallback;
  const observe = vi.fn();
  const disconnect = vi.fn();
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      observe = observe;
      disconnect = disconnect;
      constructor(callback: IntersectionObserverCallback) {
        intersect = callback;
      }
    },
  );
  const hidden = vi.spyOn(document, "hidden", "get").mockReturnValue(false);
  cleanups.push(observeMotion());
  const setVisible = (element: Element, isIntersecting: boolean): void => {
    intersect(
      [{ target: element, isIntersecting } as IntersectionObserverEntry],
      {} as IntersectionObserver,
    );
  };
  const setHidden = (value: boolean): void => {
    hidden.mockReturnValue(value);
    document.dispatchEvent(new Event("visibilitychange"));
  };
  return { elements, setVisible, setHidden, observe, disconnect };
};

describe("observeMotion", () => {
  afterEach(() => {
    cleanups.splice(0).forEach((cleanup) => cleanup());
    document.body.replaceChildren();
  });

  it("should run only the visible diagram and pause it when it leaves the viewport", () => {
    const { elements, setVisible, observe } = setup();
    expect(observe).toHaveBeenCalledTimes(2);
    expect(elements.map((element) => element.dataset.motion)).toEqual([
      "paused",
      "paused",
    ]);
    setVisible(elements[0], true);
    expect(elements.map((element) => element.dataset.motion)).toEqual([
      "running",
      "paused",
    ]);
    setVisible(elements[0], false);
    expect(elements[0].dataset.motion).toBe("paused");
  });

  it("should stop in a hidden tab and resume only the visible diagram", () => {
    const { elements, setVisible, setHidden } = setup();
    setVisible(elements[0], true);
    setHidden(true);
    expect(elements.map((element) => element.dataset.motion)).toEqual([
      "paused",
      "paused",
    ]);
    setHidden(false);
    expect(elements.map((element) => element.dataset.motion)).toEqual([
      "running",
      "paused",
    ]);
  });

  it("should remove its observer and visibility listener when cleaned up", () => {
    const { elements, setVisible, setHidden, disconnect } = setup();
    setVisible(elements[0], true);
    cleanups.pop()?.();
    setHidden(true);
    expect(disconnect).toHaveBeenCalledOnce();
    expect(elements[0].dataset.motion).toBe("running");
  });

  it("should preserve animation when IntersectionObserver is unavailable", () => {
    document.body.innerHTML = "<section data-viewport-motion></section>";
    vi.stubGlobal("IntersectionObserver", undefined);
    vi.spyOn(document, "hidden", "get").mockReturnValue(false);
    cleanups.push(observeMotion());
    expect(document.querySelector("section")?.dataset.motion).toBe("running");
  });
});
