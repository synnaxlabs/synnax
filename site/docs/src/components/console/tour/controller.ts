// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { advance, current, plot, run, type State } from "@/components/console/tour/sim";
import { GROUPS, type Scene, STEPS } from "@/components/console/tour/steps";

const ROOT = "[data-console-tour]";
const SELECTED = "pluto--selected";
const FOCUSED = "pluto--alt-color";
const ENABLED = "docs-console--enabled";
const ACTIVE = "docs-console--active";
const DONE = "docs-console--done";
const TANK_HEIGHT = 198;
const TANK_BOTTOM = 200;
// Pixels between the card and the box it points at, and between the card and the edge
// of the Console.
const GAP = 14;
const INSET = 12;
// Pixels the spotlight leaves around its box.
const PAD = 4;
// Drawers slide in, so the card moves again once they finish.
const SETTLE_MS = 180;

const TRIGGERS: Record<string, string> = {
  c: "channel",
  r: "range",
  w: "workspace",
  d: "device",
  t: "task",
  a: "arc",
  s: "status",
  v: "component",
};

const all = <E extends Element = HTMLElement>(
  root: ParentNode,
  selector: string,
): E[] => [...root.querySelectorAll<E>(selector)];

const one = <E extends Element = HTMLElement>(
  root: ParentNode,
  selector: string,
): E => {
  const el = root.querySelector<E>(selector);
  if (el == null) throw new Error(`the Console tour has no ${selector}`);
  return el;
};

const setToolbar = (root: HTMLElement, key: string | null): void => {
  for (const el of all(root, ".docs-console__drawer"))
    el.hidden = el.dataset.drawer !== key;
  for (const el of all(root, '[data-toolbar]:not([data-toolbar="component"])'))
    el.classList.toggle(SELECTED, el.dataset.toolbar === key);
};

const setComponent = (root: HTMLElement, open: boolean): void => {
  one(root, '[data-drawer="component"]').hidden = !open;
  one(root, '[data-toolbar="component"]').classList.toggle(SELECTED, open);
};

const setPalette = (root: HTMLElement, open: boolean): void => {
  one(root, "[data-palette]").hidden = !open;
};

const focusLeaf = (root: HTMLElement, leaf: string): void => {
  for (const el of all(root, "[data-leaf]")) {
    const tab = el.querySelector(`.pluto-tabs__tab.${SELECTED}`);
    tab?.classList.toggle(FOCUSED, el.dataset.leaf === leaf);
  }
  for (const el of all(root, "[data-component-icon]"))
    el.hidden = el.dataset.componentIcon !== leaf;
  for (const el of all(root, "[data-component-panel]"))
    el.hidden = el.dataset.componentPanel !== leaf;
};

const selectLeafTab = (tab: HTMLElement): void => {
  const leaf = tab.closest<HTMLElement>("[data-leaf]");
  if (leaf == null) return;
  const key = tab.dataset.leafTab;
  for (const el of all(leaf, "[data-leaf-tab]")) {
    const selected = el === tab;
    el.classList.toggle(SELECTED, selected);
    el.setAttribute("aria-selected", String(selected));
    if (!selected) el.classList.remove(FOCUSED);
  }
  for (const el of all(leaf, "[data-leaf-panel]"))
    el.hidden = el.dataset.leafPanel !== key;
};

const applyScene = (root: HTMLElement, scene: Scene): void => {
  setToolbar(root, scene.toolbar ?? null);
  setComponent(root, scene.component === true);
  setPalette(root, scene.palette === true);
  if (scene.leaf != null) focusLeaf(root, scene.leaf);
};

interface Box {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

// Returns one box around every shown element that matches the selectors, relative to
// the frame.
const boxAround = (root: HTMLElement, frame: DOMRect, selectors: string[]): Box => {
  const rects = selectors
    .flatMap((selector) => all(root, selector))
    .filter((el) => el.closest("[hidden]") == null)
    .map((el) => el.getBoundingClientRect())
    .filter((r) => r.width > 0 && r.height > 0);
  if (rects.length === 0)
    throw new Error(`nothing to highlight for ${selectors.join(", ")}`);
  return {
    left: Math.min(...rects.map((r) => r.left)) - frame.left,
    top: Math.min(...rects.map((r) => r.top)) - frame.top,
    right: Math.max(...rects.map((r) => r.right)) - frame.left,
    bottom: Math.max(...rects.map((r) => r.bottom)) - frame.top,
  };
};

const clamp = (v: number, lower: number, upper: number): number =>
  Math.min(Math.max(v, lower), Math.max(lower, upper));

type Side = "left" | "right" | "top" | "bottom";

interface Placement {
  x: number;
  y: number;
  side: Side | null;
}

// Puts the card beside the box where it fits, and inside the box when nothing does.
const placeBeside = (
  box: Box,
  width: number,
  height: number,
  frameWidth: number,
  frameHeight: number,
): Placement => {
  const midX = (box.left + box.right) / 2;
  const midY = (box.top + box.bottom) / 2;
  const x = clamp(midX - width / 2, INSET, frameWidth - width - INSET);
  const y = clamp(midY - height / 2, INSET, frameHeight - height - INSET);
  if (box.right + GAP + width <= frameWidth - INSET)
    return { x: box.right + GAP, y, side: "left" };
  if (box.left - GAP - width >= INSET)
    return { x: box.left - GAP - width, y, side: "right" };
  if (box.bottom + GAP + height <= frameHeight - INSET)
    return { x, y: box.bottom + GAP, side: "top" };
  if (box.top - GAP - height >= INSET)
    return { x, y: box.top - GAP - height, side: "bottom" };
  return {
    x: clamp(box.right - width - INSET, INSET, frameWidth - width - INSET),
    y: clamp(box.bottom - height - INSET, INSET, frameHeight - height - INSET),
    side: null,
  };
};

interface Tour {
  root: HTMLElement;
  state: State;
  visible: boolean;
  /** The index of the active step, or null before the tour starts. */
  step: number | null;
  settle?: ReturnType<typeof setTimeout>;
}

const place = ({ root, step: index }: Tour): void => {
  const frameEl = one(root, ".docs-console__frame");
  const card = one(root, "[data-card]");
  const arrow = one(root, "[data-card-arrow]");
  const spotlight = one(root, "[data-spotlight]");
  const frame = frameEl.getBoundingClientRect();
  const floating = getComputedStyle(card).position === "absolute";
  const { offsetWidth: width, offsetHeight: height } = card;
  if (index == null) {
    spotlight.classList.remove(ACTIVE);
    return;
  }
  const box = boxAround(root, frame, STEPS[index].targets);
  // The spotlight sits inside the stage, whose border moves it off the frame's corner.
  const { clientLeft, clientTop } = one(root, ".docs-console__stage");
  Object.assign(spotlight.style, {
    left: `${box.left - clientLeft - PAD}px`,
    top: `${box.top - clientTop - PAD}px`,
    width: `${box.right - box.left + PAD * 2}px`,
    height: `${box.bottom - box.top + PAD * 2}px`,
  });
  spotlight.classList.add(ACTIVE);
  if (!floating) {
    arrow.dataset.side = "";
    return;
  }
  const { x, y, side } = placeBeside(box, width, height, frame.width, frame.height);
  card.style.left = `${x}px`;
  card.style.top = `${y}px`;
  arrow.dataset.side = side ?? "";
  if (side === "left" || side === "right")
    arrow.style.top = `${clamp((box.top + box.bottom) / 2 - y, 16, height - 16)}px`;
  else arrow.style.left = `${clamp((box.left + box.right) / 2 - x, 16, width - 16)}px`;
};

const settle = (tour: Tour): void => {
  clearTimeout(tour.settle);
  requestAnimationFrame(() => place(tour));
  tour.settle = setTimeout(() => place(tour), SETTLE_MS);
};

const go = (tour: Tour, index: number | null): void => {
  const { root } = tour;
  tour.step = index;
  const active = index == null ? null : STEPS[index];
  applyScene(root, active?.scene ?? {});
  one(root, "[data-card]").hidden = active == null;
  one(root, '[data-tour="start"]').hidden = active != null;
  one(root, '[data-tour="end"]').hidden = active == null;
  for (const el of all(root, "[data-card] [data-step]"))
    el.hidden = el.dataset.step !== active?.key;
  const group = GROUPS.find((g) => g.key === active?.group);
  one(root, "[data-card-meta]").textContent =
    index == null || group == null
      ? ""
      : `${group.name} · ${index + 1} of ${STEPS.length}`;
  const backButton = one<HTMLButtonElement>(root, '[data-tour="back"]');
  backButton.disabled = index === 0;
  one(root, '[data-tour="next"]').textContent =
    index === STEPS.length - 1 ? "Finish" : "Next";
  for (const el of all(root, "[data-step-target]")) {
    const i = STEPS.findIndex((s) => s.key === el.dataset.stepTarget);
    el.classList.toggle(ACTIVE, i === index);
    el.classList.toggle(DONE, index != null && i < index);
  }
  for (const el of all(root, "[data-progress-group]"))
    el.classList.toggle(ACTIVE, el.dataset.progressGroup === active?.group);
  settle(tour);
};

const next = (tour: Tour): void => {
  if (tour.step == null) go(tour, 0);
  else go(tour, tour.step === STEPS.length - 1 ? null : tour.step + 1);
};

const back = (tour: Tour): void => {
  if (tour.step != null && tour.step > 0) go(tour, tour.step - 1);
};

const goTo = (tour: Tour, key: string): void => {
  const index = STEPS.findIndex((s) => s.key === key);
  if (index !== -1) go(tour, index);
};

// A link to a step's anchor opens the step, since the browser cannot scroll to the
// hidden step itself.
const openLinked = (tour: Tour): void => {
  const index = STEPS.findIndex((s) => `#${s.anchor}` === location.hash);
  if (index === -1) return;
  go(tour, index);
  tour.root.scrollIntoView({ block: "start" });
};

// Selecting the toolbar of the active step closes it, as in the Console.
const toggleToolbar = (tour: Tour, key: string): void => {
  if (tour.step != null && STEPS[tour.step].key === key) go(tour, null);
  else goTo(tour, key);
};

const render = ({ root, state }: Tour): void => {
  const { pressure, press, vent } = current(state);
  const box = root.querySelector<HTMLElement>("[data-plot]");
  const svg = box?.querySelector("svg");
  if (box != null && svg != null && box.clientWidth > 0) {
    const { clientWidth: w, clientHeight: h } = box;
    svg.setAttribute("viewBox", `0 0 ${w} ${h}`);
    svg.setAttribute("width", String(w));
    svg.setAttribute("height", String(h));
    svg.innerHTML = plot(state, w, h);
  }
  const fill = root.querySelector("[data-tank-fill]");
  const height = (pressure / 500) * TANK_HEIGHT;
  fill?.setAttribute("y", (TANK_BOTTOM - height).toFixed(1));
  fill?.setAttribute("height", height.toFixed(1));
  const value = root.querySelector("[data-tank-value]");
  if (value != null) value.textContent = pressure.toFixed(1);
  root
    .querySelector('[data-valve-group="press"]')
    ?.classList.toggle(ENABLED, press === 1);
  root
    .querySelector('[data-valve-group="vent"]')
    ?.classList.toggle(ENABLED, vent === 1);
};

const fit = (root: HTMLElement): void => {
  const canvas = root.querySelector<HTMLElement>(".docs-console__canvas");
  const stage = root.querySelector<HTMLElement>(".docs-console__stage");
  if (canvas == null || stage == null) return;
  const width = parseFloat(
    getComputedStyle(root).getPropertyValue("--docs-console-width"),
  );
  canvas.style.setProperty(
    "--docs-console-zoom",
    String(Math.min(1, stage.clientWidth / width)),
  );
};

const command = (tour: Tour, name: string): void => {
  const { state } = tour;
  state.stage = name === "start" ? "pressurize" : "vent";
  state.stageStart = state.t;
};

const mount = (root: HTMLElement, motion: boolean): (() => void) => {
  for (const { key } of STEPS) one(root, `[data-card] [data-step="${key}"]`);
  const tour: Tour = {
    root,
    state: run(34),
    visible: false,
    step: null,
  };
  const stage = one(root, ".docs-console__stage");
  const palette = one(root, "[data-palette]");

  const handleClick = (e: MouseEvent): void => {
    if (!(e.target instanceof Element)) return;
    const control = e.target.closest<HTMLElement>("[data-tour]");
    if (control != null) {
      const action = control.dataset.tour;
      if (action === "next" || action === "start") next(tour);
      else if (action === "back") back(tour);
      else go(tour, null);
      return;
    }
    const target = e.target.closest<HTMLElement>("[data-step-target]");
    if (target != null) {
      goTo(tour, target.dataset.stepTarget ?? "");
      return;
    }
    if (!palette.hidden) {
      setPalette(root, false);
      if (tour.step != null && STEPS[tour.step].key === "search") go(tour, null);
      return;
    }
    const cmd = e.target.closest<SVGElement>("[data-command]");
    if (cmd != null) {
      command(tour, cmd.dataset.command ?? "");
      return;
    }
    const item = e.target.closest<HTMLElement>(
      ".docs-console__drawer .pluto-list__item",
    );
    if (item != null) {
      for (const el of all(item.closest("[data-drawer]") ?? root, ".pluto-list__item"))
        el.classList.toggle(SELECTED, el === item);
      return;
    }
    const toolbar = e.target.closest<HTMLElement>("[data-toolbar]");
    if (toolbar != null) {
      toggleToolbar(tour, toolbar.dataset.toolbar ?? "");
      return;
    }
    const leafTab = e.target.closest<HTMLElement>("[data-leaf-tab]");
    if (leafTab != null) selectLeafTab(leafTab);
    const part = e.target.closest<HTMLElement>("[data-part]");
    if (part != null) goTo(tour, part.dataset.part ?? "");
  };

  const handleKey = (e: KeyboardEvent): void => {
    if (e.key === "Escape") {
      go(tour, null);
      return;
    }
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "p") {
      e.preventDefault();
      goTo(tour, "search");
      return;
    }
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    if (e.key === "ArrowRight" || e.key === "ArrowLeft") {
      e.preventDefault();
      if (e.key === "ArrowRight") next(tour);
      else back(tour);
      return;
    }
    const key = TRIGGERS[e.key.toLowerCase()];
    if (key == null) return;
    e.preventDefault();
    toggleToolbar(tour, key);
  };

  const timer = motion
    ? setInterval(() => {
        if (!tour.visible) return;
        advance(tour.state);
        render(tour);
      }, 100)
    : null;
  const intersection = new IntersectionObserver(([entry]) => {
    tour.visible = entry.isIntersecting;
  });
  intersection.observe(root);
  const resize = new ResizeObserver(() => {
    fit(root);
    render(tour);
    place(tour);
  });
  resize.observe(stage);
  resize.observe(one(root, "[data-plot]"));
  resize.observe(one(root, "[data-card]"));
  root.addEventListener("click", handleClick);
  stage.addEventListener("keydown", handleKey);
  const handleHash = (): void => openLinked(tour);
  window.addEventListener("hashchange", handleHash);
  fit(root);
  render(tour);
  go(tour, null);
  openLinked(tour);

  return () => {
    if (timer != null) clearInterval(timer);
    clearTimeout(tour.settle);
    intersection.disconnect();
    resize.disconnect();
    root.removeEventListener("click", handleClick);
    stage.removeEventListener("keydown", handleKey);
    window.removeEventListener("hashchange", handleHash);
  };
};

/**
 * Starts every copy of the Console on the page: its tour steps through each part, its
 * toolbars, tabs, and command palette respond to clicks and keys, and its tank runs
 * while it is on screen. Call it once; it follows page swaps by itself.
 */
export const start = (): void => {
  let unmounts: Array<() => void> = [];
  const load = (): void => {
    for (const unmount of unmounts) unmount();
    const motion = !matchMedia("(prefers-reduced-motion: reduce)").matches;
    unmounts = all(document, ROOT).map((root) => mount(root, motion));
  };
  load();
  document.addEventListener("astro:after-swap", load);
};
