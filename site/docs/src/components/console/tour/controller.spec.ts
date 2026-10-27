// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { start } from "@/components/console/tour/controller";
import { GROUPS, STEPS } from "@/components/console/tour/steps";
import { BASE, hotspots, STILLS } from "@/components/console/tour/stills";

class Observer {
  observe = vi.fn();
  disconnect = vi.fn();
}

// Mirrors the markup of Tour.astro and the Step sections of the page.
const markup = (): string => `
  <figure data-tour-root>
    <div class="docs-tour__stage" tabindex="0">
      ${STILLS.map((still) => `<div data-still="${still}"></div>`).join("")}
      <div data-spotlight></div>
      ${STILLS.map(
        (still) => `
          <div data-hotspots="${still}" ${still === BASE ? "" : "hidden"}>
            ${hotspots(still)
              .map(({ step }) =>
                step == null
                  ? `<div class="docs-tour__blocker"></div>`
                  : `<button class="docs-tour__hotspot" data-step-target="${step.key}">
                    </button>`,
              )
              .join("")}
          </div>`,
      ).join("")}
    </div>
    <div data-card hidden>
      <div data-card-arrow></div>
      <div data-card-meta></div>
      ${STEPS.map(
        ({ key, anchor }) =>
          `<section id="${anchor}" data-step="${key}" hidden></section>`,
      ).join("")}
      <button data-tour="back">Back</button>
      <button data-tour="next">Next</button>
    </div>
    <div>
      <button data-tour="start">Start the tour</button>
      <button data-tour="end" hidden>End the tour</button>
      ${GROUPS.map(
        (group) => `
          <div data-progress-group="${group.key}">
            ${STEPS.filter((s) => s.group === group.key)
              .map(
                (s) =>
                  `<button class="docs-tour__progress-step" data-step-target="${s.key}">
                  </button>`,
              )
              .join("")}
          </div>`,
      ).join("")}
    </div>
  </figure>
`;

const get = <E extends Element = HTMLElement>(selector: string): E => {
  const el = document.querySelector<E>(selector);
  if (el == null) throw new Error(`no ${selector}`);
  return el;
};

const control = (action: string): HTMLButtonElement => get(`[data-tour="${action}"]`);

const hotspot = (still: string, step: string): HTMLElement =>
  get(`[data-hotspots="${still}"] [data-step-target="${step}"]`);

const progress = (step: string): HTMLElement =>
  get(`.docs-tour__progress-step[data-step-target="${step}"]`);

const card = (): HTMLElement => get("[data-card]");

const shownStep = (): string | null =>
  card().hidden
    ? null
    : (document.querySelector<HTMLElement>("[data-step]:not([hidden])")?.dataset.step ??
      null);

const shownStill = (): string | undefined =>
  document.querySelector<HTMLElement>("[data-still].docs-tour--active")?.dataset.still;

const press = (init: KeyboardEventInit): boolean =>
  get(".docs-tour__stage").dispatchEvent(
    new KeyboardEvent("keydown", { bubbles: true, cancelable: true, ...init }),
  );

const scrollIntoView = vi.fn();

const mount = (): void => {
  document.body.innerHTML = markup();
  document.dispatchEvent(new Event("astro:after-swap"));
};

describe("tour", () => {
  beforeAll(() => {
    vi.stubGlobal("ResizeObserver", Observer);
    Element.prototype.scrollIntoView = scrollIntoView;
    // Browsers drop the focus of an element as it hides, and jsdom does not. jsdom
    // also ignores a blur of a hidden element, so the blur comes first.
    const hidden = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "hidden");
    Object.defineProperty(HTMLElement.prototype, "hidden", {
      ...hidden,
      set(this: HTMLElement, value: boolean) {
        const focused = document.activeElement;
        if (value && focused instanceof HTMLElement && this.contains(focused))
          focused.blur();
        hidden?.set?.call(this, value);
      },
    });
    start();
  });

  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["requestAnimationFrame"] });
    scrollIntoView.mockClear();
    mount();
  });

  afterEach(() => {
    vi.useRealTimers();
    window.history.replaceState({}, "", window.location.pathname);
  });

  it("should open the first step when the tour starts", () => {
    control("start").click();
    expect(shownStep()).toBe(STEPS[0].key);
    expect(get("[data-card-meta]").textContent).toBe(`Mosaic · 1 of ${STEPS.length}`);
    expect(control("back").disabled).toBe(true);
    expect(control("start").hidden).toBe(true);
    expect(control("end").hidden).toBe(false);
  });

  it("should step forward and back through the steps", () => {
    control("start").click();
    control("next").click();
    expect(shownStep()).toBe(STEPS[1].key);
    expect(control("back").disabled).toBe(false);
    control("back").click();
    expect(shownStep()).toBe(STEPS[0].key);
  });

  it("should end the tour when Finish is selected on the last step", () => {
    progress(STEPS[STEPS.length - 1].key).click();
    expect(control("next").textContent).toBe("Finish");
    control("next").click();
    expect(shownStep()).toBeNull();
    expect(control("start").hidden).toBe(false);
  });

  it("should end the tour when End the tour is selected", () => {
    control("start").click();
    control("end").click();
    expect(shownStep()).toBeNull();
  });

  it("should mark the steps before a selected progress step as done", () => {
    progress("search").click();
    const index = STEPS.findIndex((s) => s.key === "search");
    expect(progress(STEPS[index - 1].key).classList).toContain("docs-tour--done");
    expect(progress("search").classList).toContain("docs-tour--active");
    expect(progress("search").classList).not.toContain("docs-tour--done");
    expect(get('[data-progress-group="left"]').classList).toContain(
      "docs-tour--active",
    );
  });

  it("should move focus to the stills when the focused control hides", () => {
    control("start").focus();
    control("start").click();
    expect(document.activeElement).toBe(get(".docs-tour__stage"));
    control("end").focus();
    control("end").click();
    expect(document.activeElement).toBe(get(".docs-tour__stage"));
  });

  it("should move focus to the stills when the hotspots of a focused part hide", () => {
    hotspot(BASE, "channel").focus();
    hotspot(BASE, "channel").click();
    expect(document.activeElement).toBe(get(".docs-tour__stage"));
  });

  it("should keep focus on a control that stays shown", () => {
    control("start").click();
    control("next").focus();
    control("next").click();
    expect(document.activeElement).toBe(control("next"));
  });

  it("should show the still of a step and its hotspots", () => {
    hotspot(BASE, "channel").click();
    expect(shownStep()).toBe("channel");
    expect(shownStill()).toBe("channel");
    expect(get('[data-hotspots="channel"]').hidden).toBe(false);
    expect(get(`[data-hotspots="${BASE}"]`).hidden).toBe(true);
  });

  it("should close a toolbar step when its toolbar is selected again", () => {
    hotspot(BASE, "channel").click();
    hotspot("channel", "channel").click();
    expect(shownStep()).toBeNull();
    expect(shownStill()).toBe(BASE);
  });

  it("should keep a base still step open when its part is selected again", () => {
    hotspot(BASE, "schematic").click();
    hotspot(BASE, "schematic").click();
    expect(shownStep()).toBe("schematic");
  });

  it("should open a toolbar step when its key is pressed", () => {
    expect(press({ key: "r" })).toBe(false);
    expect(shownStep()).toBe("range");
  });

  it("should ignore a toolbar key pressed with a modifier", () => {
    expect(press({ key: "r", altKey: true })).toBe(true);
    expect(shownStep()).toBeNull();
  });

  it("should open the palette step on Cmd or Ctrl+P", () => {
    expect(press({ key: "p", metaKey: true })).toBe(false);
    expect(shownStep()).toBe("search");
    expect(shownStill()).toBe("search");
  });

  it("should step with the arrow keys", () => {
    press({ key: "ArrowRight" });
    press({ key: "ArrowRight" });
    expect(shownStep()).toBe(STEPS[1].key);
    press({ key: "ArrowLeft" });
    expect(shownStep()).toBe(STEPS[0].key);
  });

  it("should end the tour on Escape", () => {
    control("start").click();
    press({ key: "Escape" });
    expect(shownStep()).toBeNull();
  });

  it("should close the palette step on a selection outside of the palette", () => {
    press({ key: "p", ctrlKey: true });
    get('[data-hotspots="search"] .docs-tour__blocker').click();
    expect(shownStep()).toBe("search");
    get("[data-still=search]").click();
    expect(shownStep()).toBeNull();
  });

  it("should spotlight the regions of the active step", () => {
    const spotlight = get("[data-spotlight]");
    hotspot(BASE, "lineplot").click();
    vi.advanceTimersToNextFrame();
    expect(spotlight.classList).toContain("docs-tour--active");
    expect(spotlight.style.left).toMatch(/^calc\(.+% - 4px\)$/);
    control("end").click();
    vi.advanceTimersToNextFrame();
    expect(spotlight.classList).not.toContain("docs-tour--active");
  });

  it("should open the step that the URL hash links to", () => {
    const ranges = STEPS.find((s) => s.key === "range");
    window.location.hash = `#${ranges?.anchor}`;
    mount();
    expect(shownStep()).toBe("range");
    expect(scrollIntoView).toHaveBeenCalledOnce();
  });

  it("should open a step when the URL hash changes to it", () => {
    window.location.hash = "#getting-help";
    window.dispatchEvent(new HashChangeEvent("hashchange"));
    expect(shownStep()).toBe("help");
  });

  it("should stop responding to a tour that a page swap removed", () => {
    const old = get("[data-tour-root]");
    document.body.replaceChildren();
    document.dispatchEvent(new Event("astro:after-swap"));
    document.body.append(old);
    control("start").click();
    expect(shownStep()).toBeNull();
  });
});
