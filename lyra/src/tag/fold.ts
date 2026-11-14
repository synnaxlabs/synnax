// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@/css";

/** A tag's full width, and the narrowest it gets in a stack before it folds. */
interface Measure {
  natural: number;
  floor: number;
}

/** What a group shows once folded. */
export interface Hidden {
  name: string;
  color?: string;
}

const TAG = CSS.B("tag");
const MORE = CSS.BE("tags", "more");
const STACKED = CSS.BM("tags", "stacked");
const LAST = CSS.BE("tags", "last");
const MEASURING = CSS.BM("tags", "measuring");
const STACKED_VAR = "--lyra-tags-stacked";
const STACK_GAP = "--lyra-tags-stack-gap";
// The room the count takes, in rem: with the word "labels" when tags show beside it,
// without it when alone. Fixed, so the fold never chases the count's width.
const MORE_WIDE = 18.5;
const MORE_NARROW = 12;
// The characters a stacked tag keeps before it folds.
const FLOOR_CHARS = 2;

/** A tag's name, and the color of its dot if it has one. Reads style only. */
export const describe = (tag: HTMLElement): Hidden => {
  const icon = tag.querySelector("svg");
  return {
    name: tag.textContent ?? "",
    color: icon == null ? undefined : getComputedStyle(icon).color,
  };
};

const sum = (values: number[]): number => values.reduce((a, b) => a + b, 0);

const px = (value: string): number => parseFloat(value) || 0;

// An element's border-box width in its own CSS pixels. Unlike a bounding box, it does
// not change with the page's zoom.
const cssWidth = (el: Element): number => {
  const cs = getComputedStyle(el);
  const w = px(cs.width);
  if (cs.boxSizing === "border-box") return w;
  return (
    w +
    px(cs.paddingLeft) +
    px(cs.paddingRight) +
    px(cs.borderLeftWidth) +
    px(cs.borderRightWidth)
  );
};

// The box of a tag's text at full length, even where the tag clips it.
const textBox = (
  tag: HTMLElement,
): { left: number; width: number; chars: number } | null => {
  const walker = document.createTreeWalker(tag, NodeFilter.SHOW_TEXT);
  let first: Node | null = null;
  let last: Node | null = null;
  for (let n = walker.nextNode(); n != null; n = walker.nextNode())
    if ((n.textContent ?? "").trim() !== "") {
      first ??= n;
      last = n;
    }
  if (first == null || last == null) return null;
  const range = document.createRange();
  range.setStartBefore(first);
  range.setEndAfter(last);
  const r = range.getBoundingClientRect();
  const chars = range.toString().trim().length;
  return { left: r.left - tag.getBoundingClientRect().left, width: r.width, chars };
};

// Reads a tag at full width, in CSS pixels: the group must carry the measuring class.
const measure = (tag: HTMLElement): Measure => {
  const natural = cssWidth(tag);
  const text = textBox(tag);
  const scale = tag.getBoundingClientRect().width / natural;
  if (text == null || !(scale > 0)) return { natural, floor: natural };
  const cs = getComputedStyle(tag);
  const right = px(cs.paddingRight) + px(cs.borderRightWidth);
  const kept = (FLOOR_CHARS * text.width) / Math.max(1, text.chars);
  const floor = (text.left + Math.min(text.width, kept)) / scale + right;
  return { natural, floor: Math.min(natural, floor) };
};

/**
 * The number of leading tags that fit in room, each shrunk by flex no further than its
 * floor. The last shown tag keeps its full width. Every tag past it folds into a count
 * of more pixels.
 */
const countShown = (
  room: number,
  tags: Measure[],
  gap: number,
  more: number,
): number => {
  const fits = (k: number): boolean => {
    if (k === 0) return true;
    const space = room - gap * (k - 1) - (k < tags.length ? more + gap : 0);
    const under = tags.slice(0, k - 1);
    const total = sum(under.map((t) => t.natural));
    const over = total + tags[k - 1].natural - space;
    if (over <= 0) return true;
    if (over >= total) return false;
    return under.every((t) => t.natural - (over * t.natural) / total >= t.floor);
  };
  let k = tags.length;
  while (k > 0 && !fits(k)) k--;
  return k;
};

/** How the count shows: with the word "labels", without it, or not at all. */
export type Count = "wide" | "narrow" | "none";

interface Plan {
  width: number;
  stacked: boolean;
  shown: number;
  count: Count;
}

/**
 * One group of tags. Its element asks for the width of every tag at full length, so its
 * box, and with it the fold, depends only on the room its container gives.
 */
export class Group {
  readonly el: HTMLElement;
  private readonly key: string;
  private readonly onHidden: (hidden: Hidden[], count: Count) => void;
  private readonly cache = new Map<string, Measure>();
  // The group's width with every tag at full length, for the tags it holds.
  private full: { tags: string; width: number; height: number; gap: number } | null =
    null;
  private sized: Group["full"] = null;
  private saved: Partial<CSSStyleDeclaration> | null = null;

  constructor(
    el: HTMLElement,
    key: string,
    onHidden: (hidden: Hidden[], count: Count) => void,
  ) {
    this.el = el;
    this.key = key;
    this.onHidden = onHidden;
  }

  private parts(): {
    tags: HTMLElement[];
    more: HTMLElement | undefined;
    fixed: number;
  } {
    const children = Array.from(this.el.children) as HTMLElement[];
    const tags = children.filter((c) => c.classList.contains(TAG));
    const more = children.find((c) => c.classList.contains(MORE));
    return {
      tags,
      more,
      fixed: children.length - tags.length - (more == null ? 0 : 1),
    };
  }

  private keyOf(tag: HTMLElement): string {
    return `${this.key}|${tag.textContent ?? ""}`;
  }

  private signature(): string {
    const { tags, fixed } = this.parts();
    return `${fixed}|${tags.map((t) => this.keyOf(t)).join("\n")}`;
  }

  /** Forgets every measure, such as after a font or zoom change. Reads nothing. */
  invalidate(): void {
    this.cache.clear();
    this.full = null;
  }

  /** Whether the group holds tags it has not measured. Reads nothing. */
  unmeasured(): boolean {
    return this.full?.tags !== this.signature();
  }

  /** Lays every tag out at full length, for measure. Reads nothing. */
  expand(): void {
    const { el } = this;
    const { tags, more } = this.parts();
    const { width, height, minWidth, maxWidth, flexShrink } = el.style;
    this.saved = { width, height, minWidth, maxWidth, flexShrink };
    Object.assign(el.style, {
      width: "max-content",
      height: "",
      minWidth: "0",
      maxWidth: "none",
      flexShrink: "0",
    });
    el.classList.remove(STACKED);
    el.style.setProperty(STACKED_VAR, "false");
    el.classList.add(MEASURING);
    tags.forEach((tag) => (tag.style.display = ""));
    if (more != null) more.style.display = "none";
  }

  /** Reads the group and its tags at full length. Reads layout only. */
  measure(): void {
    if (this.el.getClientRects().length === 0) return;
    const { tags } = this.parts();
    tags.forEach((tag) => {
      const key = this.keyOf(tag);
      if (!this.cache.has(key)) this.cache.set(key, measure(tag));
    });
    const cs = getComputedStyle(this.el);
    this.full = {
      tags: this.signature(),
      width: cssWidth(this.el),
      height: px(cs.height),
      gap: px(cs.columnGap),
    };
  }

  /** Puts back what expand changed, except the fold, which the next write sets. */
  collapse(): void {
    const { el } = this;
    const { more } = this.parts();
    if (this.saved != null) Object.assign(el.style, this.saved);
    this.saved = null;
    el.classList.remove(MEASURING);
    if (more != null) more.style.display = "";
  }

  /**
   * Reads layout only. Returns null when the group keeps its current fold or has not
   * measured its tags.
   */
  read(rem: number): Plan | null {
    const { el, full } = this;
    // A group that does not render, such as one in a hidden panel, measures as zero.
    if (full == null || el.getClientRects().length === 0) return null;
    if (full.tags !== this.signature()) return null;
    const measures = this.parts().tags.map(
      (t) => this.cache.get(this.keyOf(t)) as Measure,
    );
    const cs = getComputedStyle(el);
    const stackGap = px(cs.getPropertyValue(STACK_GAP));
    const available = cssWidth(el);
    const loose = full.width <= available + 0.5;
    // Without its tags and gaps, the group keeps its padding, border, and fixed parts.
    const { tags, fixed } = this.parts();
    const gaps = Math.max(0, fixed + tags.length - 1);
    const frame = full.width - sum(measures.map((m) => m.natural)) - full.gap * gaps;
    const room = available - frame - stackGap * fixed;
    const shown = loose
      ? tags.length
      : countShown(room, measures, stackGap, MORE_WIDE * rem);
    let count: Count = "none";
    if (shown < tags.length)
      if (shown > 0) count = "wide";
      else if (room >= MORE_NARROW * rem) count = "narrow";
    return {
      width: full.width,
      stacked: !loose || shown < tags.length,
      shown,
      count,
    };
  }

  /** Whether the group asks for a new width. Reads nothing. */
  resizes(): boolean {
    return this.full != null && this.sized !== this.full;
  }

  /** Sets the width the group asks for. Reads nothing. */
  size(): void {
    if (this.full == null) return;
    const { el } = this;
    this.sized = this.full;
    el.style.width = `${this.full.width}px`;
    // The tags shown change the height of the group, and only the room may change its
    // size.
    el.style.height = `${this.full.height}px`;
    el.style.maxWidth = "100%";
    el.style.minWidth = el.classList.contains(CSS.M("full-x")) ? "100%" : "0";
  }

  /** Writes the fold. Reads nothing. */
  write(plan: Plan): void {
    const { el } = this;
    const { tags, more } = this.parts();
    el.classList.toggle(STACKED, plan.stacked);
    el.style.setProperty(STACKED_VAR, String(plan.stacked));
    // The count unmounts on the next render. Until then, it must not take room.
    if (more != null) more.style.display = plan.count === "none" ? "none" : "";
    tags.forEach((tag, i) => {
      tag.style.display = i < plan.shown ? "" : "none";
      tag.classList.toggle(LAST, plan.stacked && i === plan.shown - 1);
    });
    this.onHidden(tags.slice(plan.shown).map(describe), plan.count);
  }
}

/**
 * Fits groups of tags together: all reads, then all writes, so a list of groups costs
 * a fixed number of layouts, not some per group. Refits when a group changes size or
 * contents.
 */
export class Batch {
  private readonly groups = new Map<Element, Group>();
  private readonly dirty = new Set<Group>();
  private frame = 0;
  private readonly observer = new ResizeObserver((entries) => {
    entries.forEach((e) => {
      const g = this.groups.get(e.target);
      if (g != null) this.dirty.add(g);
    });
    this.flush(true);
  });
  // Tags that change inside a group, such as labels that load late, change what fits.
  private readonly mutations = new MutationObserver((records) =>
    records.forEach((r) => {
      for (let n: Node | null = r.target; n != null; n = n.parentNode) {
        const g = this.groups.get(n as Element);
        if (g != null) return this.request(g);
      }
    }),
  );

  // A change to fonts, the root's styles, or the pixel ratio changes every tag's width.
  private readonly root = new MutationObserver(() => this.invalidate());
  private resolution: MediaQueryList | null = null;
  private readonly onFonts = (): void => this.invalidate();
  private readonly onResolution = (): void => {
    this.watchResolution();
    this.invalidate();
  };

  constructor() {
    this.root.observe(document.documentElement, { attributes: true });
    this.watchResolution();
    document.fonts.addEventListener("loadingdone", this.onFonts);
  }

  // A resolution query matches one pixel ratio, so each change needs a new one.
  private watchResolution(): void {
    this.resolution?.removeEventListener("change", this.onResolution);
    this.resolution = matchMedia(`(resolution: ${devicePixelRatio}dppx)`);
    this.resolution.addEventListener("change", this.onResolution);
  }

  /** Stops watching the page. */
  dispose(): void {
    cancelAnimationFrame(this.frame);
    this.observer.disconnect();
    this.mutations.disconnect();
    this.root.disconnect();
    this.resolution?.removeEventListener("change", this.onResolution);
    document.fonts.removeEventListener("loadingdone", this.onFonts);
  }

  private invalidate(): void {
    this.groups.forEach((g) => {
      g.invalidate();
      this.request(g);
    });
  }

  add(group: Group): void {
    this.groups.set(group.el, group);
    this.request(group);
    this.observer.observe(group.el);
    this.mutations.observe(group.el, {
      childList: true,
      subtree: true,
      characterData: true,
    });
  }

  remove(group: Group): void {
    this.groups.delete(group.el);
    this.dirty.delete(group);
    this.observer.unobserve(group.el);
  }

  /** Refits the group before the next paint. */
  request(group: Group): void {
    this.dirty.add(group);
    if (this.frame !== 0) return;
    this.frame = requestAnimationFrame(() => {
      this.frame = 0;
      this.flush(false);
    });
  }

  // A write in the observer's callback must not resize a group: the observer would
  // report it a frame late, with a loop error. Such groups wait for the next frame.
  private flush(observing: boolean): void {
    const groups = [...this.dirty];
    this.dirty.clear();
    const measuring = groups.filter((g) => g.unmeasured());
    if (observing && measuring.length > 0) {
      measuring.forEach((g) => this.request(g));
      return this.fold(
        groups.filter((g) => !measuring.includes(g)),
        observing,
      );
    }
    // All groups lay out at full length together, so measuring costs one layout.
    measuring.forEach((g) => g.expand());
    measuring.forEach((g) => g.measure());
    measuring.forEach((g) => g.collapse());
    this.fold(groups, observing);
  }

  private fold(groups: Group[], observing: boolean): void {
    const resized = groups.filter((g) => g.resizes());
    if (observing) {
      resized.forEach((g) => this.request(g));
      groups = groups.filter((g) => !resized.includes(g));
    } else resized.forEach((g) => g.size());
    const rem = px(getComputedStyle(document.documentElement).fontSize);
    const plans = groups.map((g) => g.read(rem));
    groups.forEach((g, i) => {
      const plan = plans[i];
      if (plan != null) g.write(plan);
    });
  }
}
