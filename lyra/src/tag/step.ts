// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@/css";
import { cssWidth, type Measure, measure, px, sum } from "@/tag/fold";

const TAG = CSS.B("tag");
const MORE = CSS.BE("tags", "more");
const LAST = CSS.BE("tags", "last");
const DOTS = CSS.BM("tags", "dots");
const EMPTY = CSS.BM("tags", "empty");
const ENTERING = CSS.BM("tags", "entering");
const MEASURING = CSS.BM("tags", "measuring");
const HIDDEN = "data-hidden";
const STACKED = "--lyra-tags-stacked";
const STACK_GAP = "--lyra-tags-stack-gap";
// The room, in rem, the count takes beside tags, and with its dots alone.
const COUNT = 13;
const DOT_COUNT = 10;

/** A group's tags at full width, in CSS pixels. */
interface Fit {
  tags: string;
  measures: Measure[];
  // Padding, border, and the parts that are not tags, such as a caret.
  frame: number;
  fixed: number;
  gap: number;
  stackGap: number;
}

const tagsOf = (el: HTMLElement): HTMLElement[] =>
  Array.from(el.children).filter((c) => c.classList.contains(TAG)) as HTMLElement[];

const signature = (el: HTMLElement): string =>
  tagsOf(el)
    .map((t) => t.textContent ?? "")
    .join("\n");

// The parts of a group that are neither tags nor the count.
const fixedOf = (el: HTMLElement): Element[] =>
  Array.from(el.children).filter(
    (c) =>
      !c.classList.contains(TAG) &&
      !c.classList.contains(MORE) &&
      getComputedStyle(c).display !== "none",
  );

const moreOf = (el: HTMLElement): HTMLElement | undefined =>
  Array.from(el.children).find(
    (c): c is HTMLElement => c instanceof HTMLElement && c.classList.contains(MORE),
  );

const expand = (el: HTMLElement): void => {
  const more = moreOf(el);
  if (more != null) delete more.dataset.count;
  el.classList.remove(DOTS, EMPTY);
  el.classList.add(MEASURING);
  el.style.setProperty(STACKED, "false");
  tagsOf(el).forEach((t) => t.removeAttribute(HIDDEN));
};

const fit = (el: HTMLElement): Fit => {
  const tags = tagsOf(el);
  const measures = tags.map(measure);
  const cs = getComputedStyle(el);
  const gap = px(cs.columnGap);
  const fixed = fixedOf(el);
  return {
    tags: signature(el),
    measures,
    frame:
      px(cs.paddingLeft) +
      px(cs.paddingRight) +
      px(cs.borderLeftWidth) +
      px(cs.borderRightWidth) +
      sum(fixed.map(cssWidth)),
    fixed: fixed.length,
    gap,
    stackGap: px(cs.getPropertyValue(STACK_GAP)),
  };
};

// The width a group takes with its first k tags at full width.
const full = (f: Fit, k: number, count: number): number => {
  const n = f.measures.length;
  const tags = sum(f.measures.slice(0, k).map((m) => m.natural));
  const gaps = f.gap * Math.max(0, k + f.fixed - 1);
  return f.frame + tags + gaps + (k < n ? f.gap + count : 0);
};

// The narrowest a group gets with its first k tags stacked: each at its floor, the last
// at full width.
const stacked = (f: Fit, k: number, count: number): number => {
  const n = f.measures.length;
  const under = sum(f.measures.slice(0, k - 1).map((m) => m.floor));
  const last = f.measures[k - 1].natural;
  const gaps = f.stackGap * Math.max(0, k + f.fixed - 1);
  return f.frame + under + last + gaps + (k < n ? f.gap + count : 0);
};

/**
 * Fits the groups of tags of one list together, so they all show the same step: the
 * same number of tags, then a count. A group stacks its tags only when they do not fit
 * at full width. It sets `--lyra-tags-need` on its root to the full width of the
 * widest group, and `--lyra-tags-fit` on each group to the width its step takes
 * without stacking.
 */
export class Steps {
  private readonly groups = new Set<HTMLElement>();
  private readonly fits = new Map<HTMLElement, Fit>();
  private root: HTMLElement | null = null;
  private frame = 0;
  // A resize refits before the frame paints. A hovered group widens alone, and may stop
  // stacking.
  private readonly resize = new ResizeObserver(() => this.flush());
  // Groups mount and their tags load as the list scrolls.
  private readonly mutations = new MutationObserver(() => this.request());
  private readonly onFonts = (): void => {
    this.fits.clear();
    this.request();
  };

  /** Watches the element that holds the groups, and sets `--lyra-tags-need` on it. */
  attach(root: HTMLElement): void {
    this.root = root;
    document.fonts.addEventListener("loadingdone", this.onFonts);
    this.mutations.observe(root, {
      childList: true,
      subtree: true,
      characterData: true,
    });
    this.request();
  }

  /** Stops watching the page, until the next {@link attach}. */
  detach(): void {
    cancelAnimationFrame(this.frame);
    this.frame = 0;
    this.resize.disconnect();
    this.mutations.disconnect();
    document.fonts.removeEventListener("loadingdone", this.onFonts);
  }

  add(el: HTMLElement): void {
    this.groups.add(el);
    this.resize.observe(el);
    this.request();
  }

  remove(el: HTMLElement): void {
    this.groups.delete(el);
    this.fits.delete(el);
    this.resize.unobserve(el);
  }

  private request(): void {
    if (this.frame !== 0) return;
    this.frame = requestAnimationFrame(() => {
      this.frame = 0;
      this.flush();
    });
  }

  private flush(): void {
    cancelAnimationFrame(this.frame);
    this.frame = 0;
    const els = [...this.groups].filter((el) => el.getClientRects().length > 0);
    const stale = els.filter((el) => this.fits.get(el)?.tags !== signature(el));
    // Tags that load after their group painted fade in, rather than pop.
    stale.forEach((el) => {
      const before = this.fits.get(el)?.tags.split("\n");
      if (before == null) return;
      tagsOf(el).forEach((t) => {
        if (!before.includes(t.textContent ?? "")) t.classList.add(ENTERING);
      });
    });
    // All new groups lay out at full width together, so measuring costs one layout.
    stale.forEach(expand);
    stale.forEach((el) => this.fits.set(el, fit(el)));
    stale.forEach((el) => el.classList.remove(MEASURING));
    const fitted = els.filter((el) => this.fits.get(el)?.measures.length !== 0);
    if (fitted.length === 0) return;
    const rem = px(getComputedStyle(document.documentElement).fontSize);
    const count = COUNT * rem;
    const fits = fitted.map((el) => this.fits.get(el) as Fit);
    const need = Math.max(...fits.map((f) => full(f, f.measures.length, count)));
    // Set before reading widths, since the container sizes from it.
    this.root?.style.setProperty("--lyra-tags-need", `${need}px`);
    // Hovering widens one group's container, so the narrowest is the shared room.
    const room = Math.min(...fitted.map(cssWidth)) + 0.5;
    const most = Math.max(...fits.map((f) => f.measures.length));
    let shown = most;
    while (
      shown > 0 &&
      !fits.every((f) => stacked(f, Math.min(shown, f.measures.length), count) <= room)
    )
      shown--;
    const dots = DOT_COUNT * rem;
    fitted.forEach((el, i) =>
      write(el, fits[i], shown, cssWidth(el) + 0.5, count, dots),
    );
  }
}

const write = (
  el: HTMLElement,
  f: Fit,
  step: number,
  room: number,
  count: number,
  dots: number,
): void => {
  const tags = tagsOf(el);
  const shown = Math.min(step, tags.length);
  const overlap = shown > 0 && full(f, shown, count) > room;
  el.style.setProperty(STACKED, String(overlap));
  el.style.setProperty("--lyra-tags-fit", `${full(f, shown, count)}px`);
  el.classList.toggle(DOTS, shown === 0);
  el.classList.toggle(EMPTY, shown === 0 && f.frame + dots > room);
  tags.forEach((t, i) => {
    t.toggleAttribute(HIDDEN, i >= shown);
    t.classList.toggle(LAST, overlap && i === shown - 1);
  });
  const more = moreOf(el);
  if (more == null) return;
  // A stacked group's count keeps the full gap, clear of the last tag's face.
  more.style.marginLeft = overlap ? `${f.gap - f.stackGap}px` : "";
  if (shown < tags.length) more.dataset.count = String(tags.length - shown);
  else delete more.dataset.count;
};
