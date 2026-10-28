// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { find, GROUPS, STEPS } from "@/components/console/tour/steps";
import { BASE, box } from "@/components/console/tour/stills";

const ROOT = "[data-tour-root]";
const ACTIVE = "docs-tour--active";
const DONE = "docs-tour--done";
const WARM = "docs-tour--warm";
// Pixels between the card and the box it points at, and between the card and the edge
// of the stage.
const GAP = 14;
const INSET = 12;
// Pixels the spotlight leaves around its box.
const PAD = 4;

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

const clamp = (v: number, lower: number, upper: number): number =>
  Math.min(Math.max(v, lower), Math.max(lower, upper));

interface Rect {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

type Side = "left" | "right" | "top" | "bottom";

interface Placement {
  x: number;
  y: number;
  side: Side | null;
}

// Puts the card beside the box where it fits, and inside the box when nothing does.
const placeBeside = (
  rect: Rect,
  width: number,
  height: number,
  frameWidth: number,
  frameHeight: number,
): Placement => {
  const midX = (rect.left + rect.right) / 2;
  const midY = (rect.top + rect.bottom) / 2;
  const x = clamp(midX - width / 2, INSET, frameWidth - width - INSET);
  const y = clamp(midY - height / 2, INSET, frameHeight - height - INSET);
  if (rect.right + GAP + width <= frameWidth - INSET)
    return { x: rect.right + GAP, y, side: "left" };
  if (rect.left - GAP - width >= INSET)
    return { x: rect.left - GAP - width, y, side: "right" };
  if (rect.bottom + GAP + height <= frameHeight - INSET)
    return { x, y: rect.bottom + GAP, side: "top" };
  if (rect.top - GAP - height >= INSET)
    return { x, y: rect.top - GAP - height, side: "bottom" };
  return {
    x: clamp(rect.right - width - INSET, INSET, frameWidth - width - INSET),
    y: clamp(rect.bottom - height - INSET, INSET, frameHeight - height - INSET),
    side: null,
  };
};

interface Tour {
  root: HTMLElement;
  /** The index of the active step, or null before the tour starts. */
  step: number | null;
}

const place = ({ root, step: index }: Tour): void => {
  const spotlight = one(root, "[data-spotlight]");
  if (index == null) {
    spotlight.classList.remove(ACTIVE);
    return;
  }
  const { still, regions } = STEPS[index];
  const { left, top, width, height } = box(still, regions);
  Object.assign(spotlight.style, {
    left: `calc(${left}% - ${PAD}px)`,
    top: `calc(${top}% - ${PAD}px)`,
    width: `calc(${width}% + ${PAD * 2}px)`,
    height: `calc(${height}% + ${PAD * 2}px)`,
  });
  spotlight.classList.add(ACTIVE);

  const card = one(root, "[data-card]");
  const arrow = one(root, "[data-card-arrow]");
  if (getComputedStyle(card).position !== "absolute") {
    arrow.dataset.side = "";
    return;
  }
  // The stage's border moves its content off the frame's corner.
  const stage = one(root, ".docs-tour__stage");
  const { clientLeft, clientTop, clientWidth, clientHeight } = stage;
  const rect: Rect = {
    left: clientLeft + (left / 100) * clientWidth,
    top: clientTop + (top / 100) * clientHeight,
    right: clientLeft + ((left + width) / 100) * clientWidth,
    bottom: clientTop + ((top + height) / 100) * clientHeight,
  };
  const { offsetWidth: cardWidth, offsetHeight: cardHeight } = card;
  const { x, y, side } = placeBeside(
    rect,
    cardWidth,
    cardHeight,
    stage.offsetWidth,
    stage.offsetHeight,
  );
  card.style.left = `${x}px`;
  card.style.top = `${y}px`;
  arrow.dataset.side = side ?? "";
  const midX = (rect.left + rect.right) / 2;
  const midY = (rect.top + rect.bottom) / 2;
  const vertical = side === "left" || side === "right";
  Object.assign(arrow.style, {
    top: vertical ? `${clamp(midY - y, 16, cardHeight - 16)}px` : "",
    left: vertical ? "" : `${clamp(midX - x, 16, cardWidth - 16)}px`,
  });
};

const show = (root: HTMLElement, still: string): void => {
  for (const el of all(root, "[data-still]")) {
    const active = el.dataset.still === still;
    el.classList.toggle(ACTIVE, active);
    if (active) el.removeAttribute("aria-hidden");
    else el.setAttribute("aria-hidden", "true");
  }
  for (const el of all(root, "[data-hotspots]"))
    el.hidden = el.dataset.hotspots !== still;
};

const go = (tour: Tour, index: number | null): void => {
  const { root } = tour;
  // The tour's keys listen on its root, so focus must not leave with a hidden control.
  // Browsers drop the focus of a control as it hides, so record it first.
  const focused = root.contains(document.activeElement) ? document.activeElement : null;
  root.classList.add(WARM);
  tour.step = index;
  const active = index == null ? null : STEPS[index];
  show(root, active?.still ?? BASE);
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
  one<HTMLButtonElement>(root, '[data-tour="back"]').disabled = index === 0;
  one(root, '[data-tour="next"]').textContent =
    index === STEPS.length - 1 ? "Finish" : "Next";
  for (const el of all(root, ".docs-tour__progress-step")) {
    const i = STEPS.findIndex((s) => s.key === el.dataset.stepTarget);
    el.classList.toggle(ACTIVE, i === index);
    el.classList.toggle(DONE, index != null && i < index);
  }
  for (const el of all(root, "[data-progress-group]"))
    el.classList.toggle(ACTIVE, el.dataset.progressGroup === active?.group);
  if (focused?.closest("[hidden]") != null) one(root, ".docs-tour__stage").focus();
  requestAnimationFrame(() => place(tour));
};

const next = (tour: Tour): void => {
  if (tour.step == null) go(tour, 0);
  else go(tour, tour.step === STEPS.length - 1 ? null : tour.step + 1);
};

const back = (tour: Tour): void => {
  if (tour.step != null && tour.step > 0) go(tour, tour.step - 1);
};

const indexOf = (key: string): number => STEPS.indexOf(find(key));

// Selecting the part of the active step closes what the step opened, as in the
// Console: a toolbar, the component toolbar, or the palette.
const select = (tour: Tour, key: string): void => {
  const index = indexOf(key);
  if (index === tour.step && STEPS[index].still !== BASE) go(tour, null);
  else go(tour, index);
};

// A link to a step's anchor opens the step, since the browser cannot scroll to the
// hidden step itself.
const openLinked = (tour: Tour): void => {
  const index = STEPS.findIndex((s) => `#${s.anchor}` === location.hash);
  if (index === -1) return;
  go(tour, index);
  tour.root.scrollIntoView({ block: "start" });
};

const mount = (root: HTMLElement): (() => void) => {
  for (const { key } of STEPS) one(root, `[data-card] [data-step="${key}"]`);
  const tour: Tour = { root, step: null };
  const stage = one(root, ".docs-tour__stage");

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
    const progress = e.target.closest<HTMLElement>(".docs-tour__progress-step");
    if (progress != null) {
      go(tour, indexOf(progress.dataset.stepTarget ?? ""));
      return;
    }
    const hotspot = e.target.closest<HTMLElement>(".docs-tour__hotspot");
    if (hotspot != null) {
      select(tour, hotspot.dataset.stepTarget ?? "");
      return;
    }
    if (
      tour.step != null &&
      STEPS[tour.step].modal === true &&
      e.target.closest(".docs-tour__blocker") == null &&
      stage.contains(e.target)
    )
      go(tour, null);
  };

  const handleKey = (e: KeyboardEvent): void => {
    if (e.key === "Escape") {
      go(tour, null);
      return;
    }
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "p") {
      e.preventDefault();
      select(tour, "search");
      return;
    }
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    if (e.key === "ArrowRight" || e.key === "ArrowLeft") {
      e.preventDefault();
      if (e.key === "ArrowRight") next(tour);
      else back(tour);
      return;
    }
    const step = STEPS.find((s) => s.trigger?.toLowerCase() === e.key.toLowerCase());
    if (step == null) return;
    e.preventDefault();
    select(tour, step.key);
  };

  // The other stills load once the reader reaches for the tour.
  const warm = (): void => root.classList.add(WARM);
  const resize = new ResizeObserver(() => place(tour));
  resize.observe(stage);
  resize.observe(one(root, "[data-card]"));
  root.addEventListener("click", handleClick);
  root.addEventListener("pointerenter", warm, { once: true });
  root.addEventListener("focusin", warm, { once: true });
  root.addEventListener("keydown", handleKey);
  const handleHash = (): void => openLinked(tour);
  window.addEventListener("hashchange", handleHash);
  openLinked(tour);

  return () => {
    resize.disconnect();
    root.removeEventListener("click", handleClick);
    root.removeEventListener("pointerenter", warm);
    root.removeEventListener("focusin", warm);
    root.removeEventListener("keydown", handleKey);
    window.removeEventListener("hashchange", handleHash);
  };
};

/**
 * Starts every Console tour on the page: its steps highlight each part of the Console
 * on stills of it, and its parts, toolbar keys, and command palette keys open their
 * steps. Call it once; it follows page swaps by itself.
 */
export const start = (): void => {
  let unmounts: Array<() => void> = [];
  const load = (): void => {
    for (const unmount of unmounts) unmount();
    unmounts = all(document, ROOT).map(mount);
  };
  load();
  document.addEventListener("astro:after-swap", load);
};
