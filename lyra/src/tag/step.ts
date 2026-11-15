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
// React owns class names and rewrites them on render, so the step marks its state with
// data attributes instead.
const LAST = "data-last";
const DOT = "data-dot";
const EMPTY = "data-empty";
const ENTERING = "data-entering";
const MEASURING = "data-measuring";
const LIFTED = "data-lifted";
const HIDDEN = "data-hidden";
const STACKED = "--lyra-tags-stacked";
// The room the count takes with its dots, in rem. Fixed, so the step never chases the
// count's width.
const COUNT = 11;
// The dots of the first hidden tags the count shows.
const DOTS = 3;

/** A group's tags at full width, in CSS pixels. */
interface Fit {
  tags: string;
  measures: Measure[];
  // Padding, border, and the parts that are not tags, such as a caret.
  frame: number;
  fixed: number;
  gap: number;
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
  el.removeAttribute(EMPTY);
  el.toggleAttribute(MEASURING, true);
  el.style.setProperty(STACKED, "false");
  tagsOf(el).forEach((t) => {
    t.removeAttribute(HIDDEN);
    t.removeAttribute(DOT);
  });
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
  };
};

/**
 * What laid-out groups teach, so a later group whose tags were all seen before fits
 * before the browser styles it.
 */
interface Template {
  // Each tag's widths, by its name.
  tags: Map<string, Measure>;
  frame: number;
  fixed: number;
  gap: number;
  rem: number;
}

const learn = (t: Template, el: HTMLElement, f: Fit): void =>
  tagsOf(el).forEach((tag, i) => t.tags.set(tag.textContent ?? "", f.measures[i]));

// The group's fit from the tags the template has seen, or null if one is new.
const estimate = (el: HTMLElement, t: Template): Fit | null => {
  const measures: Measure[] = [];
  for (const tag of tagsOf(el)) {
    const m = t.tags.get(tag.textContent ?? "");
    if (m == null) return null;
    measures.push(m);
  }
  return {
    tags: signature(el),
    measures,
    frame: t.frame,
    fixed: t.fixed,
    gap: t.gap,
  };
};

// The width a group takes with its first k tags at full width.
const full = (f: Fit, k: number, count: number): number => {
  const n = f.measures.length;
  const tags = sum(f.measures.slice(0, k).map((m) => m.natural));
  const gaps = f.gap * Math.max(0, k + f.fixed - 1);
  return f.frame + tags + gaps + (k < n ? f.gap + count : 0);
};

// Whether a group's first k tags fit in room once stacked. Flex shrinks every tag but
// the last in proportion to its width, and none may pass its floor.
const stacks = (f: Fit, k: number, count: number, room: number): boolean => {
  if (k === 0) return true;
  const under = f.measures.slice(0, k - 1);
  const total = sum(under.map((m) => m.natural));
  const over = full(f, k, count) - room;
  if (over <= 0) return true;
  if (over >= total) return false;
  return under.every((m) => m.natural - (over * m.natural) / total >= m.floor);
};

/**
 * Fits the groups of tags of one list together, so they all show the same step: the
 * same number of tags, then a count. A group stacks its tags only when they do not fit
 * at full width, and folds one more tag into the count before any stacked tag passes
 * its floor. A hovered stacked tag lifts over its neighbors at full width.
 */
export class Steps {
  private readonly groups = new Set<HTMLElement>();
  private readonly fits = new Map<HTMLElement, Fit>();
  private readonly widths = new Map<HTMLElement, number>();
  private root: HTMLElement | null = null;
  private template: Template | null = null;
  private early = false;
  private frame = 0;
  // A resize refits before the frame paints, where reading widths costs no layout.
  private readonly resize = new ResizeObserver((entries) => {
    const changed = entries.filter((e) => {
      const el = e.target as HTMLElement;
      const width = cssWidth(el);
      if (this.widths.get(el) === width) return false;
      this.widths.set(el, width);
      return true;
    });
    if (changed.length > 0) this.flush();
  });
  // Groups mount and their tags load as the list scrolls.
  private readonly mutations = new MutationObserver(() => this.request());
  private readonly onFonts = (): void => {
    this.fits.clear();
    this.template = null;
    this.request();
  };

  // A hovered stacked tag lifts over its neighbors at full width. Its margin gives back
  // the width it gains, so they do not move.
  private lifted: HTMLElement | null = null;
  private readonly onOver = (e: PointerEvent): void => {
    const tag = (e.target as Element).closest<HTMLElement>(`.${TAG}`);
    if (tag === this.lifted) return;
    this.drop();
    const el = tag?.parentElement;
    if (tag == null || el == null || !this.groups.has(el)) return;
    if (el.style.getPropertyValue(STACKED) !== "true") return;
    if (tag.hasAttribute(LAST) || tag.hasAttribute(DOT)) return;
    const full = this.fits.get(el)?.measures[tagsOf(el).indexOf(tag)];
    if (full == null) return;
    tag.style.setProperty("--lyra-tag-full", `${full.natural}px`);
    tag.style.setProperty("--lyra-tag-shrunk", `${cssWidth(tag)}px`);
    tag.toggleAttribute(LIFTED, true);
    this.lifted = tag;
  };
  private readonly onOut = (e: PointerEvent): void => {
    const to = e.relatedTarget as Node | null;
    if (this.lifted != null && (to == null || !this.lifted.contains(to))) this.drop();
  };

  private drop(): void {
    this.lifted?.removeAttribute(LIFTED);
    this.lifted = null;
  }

  /** Watches the element that holds the groups for changes to their tags. */
  attach(root: HTMLElement): void {
    this.root = root;
    root.addEventListener("pointerover", this.onOver);
    root.addEventListener("pointerout", this.onOut);
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
    this.root?.removeEventListener("pointerover", this.onOver);
    this.root?.removeEventListener("pointerout", this.onOut);
    this.drop();
    document.fonts.removeEventListener("loadingdone", this.onFonts);
  }

  /**
   * Fits a group before the browser styles it, from what an earlier group taught, or
   * else once it lays out, before it paints.
   */
  add(el: HTMLElement): void {
    this.groups.add(el);
    this.resize.observe(el);
    if (this.early) return;
    this.early = true;
    queueMicrotask(() => {
      this.early = false;
      this.fitEarly();
    });
  }

  // New groups share the width of the groups already laid out.
  private fitEarly(): void {
    const template = this.template;
    if (template == null) return;
    const laid = [...this.widths.values()].filter((w) => w > 0);
    if (laid.length === 0) return;
    const width = Math.min(...laid);
    const fresh = [...this.groups].filter((el) => !this.widths.has(el));
    const known = fresh.filter((el) => {
      const fit = estimate(el, template);
      if (fit == null) return false;
      this.fits.set(el, fit);
      return true;
    });
    if (known.length === 0) return;
    known.forEach((el) => this.widths.set(el, width));
    this.flush();
  }

  remove(el: HTMLElement): void {
    this.groups.delete(el);
    this.fits.delete(el);
    this.widths.delete(el);
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
    // A group fits once its resize reports its width.
    const els = [...this.groups].filter((el) => this.widths.has(el));
    const stale = els.filter((el) => this.fits.get(el)?.tags !== signature(el));
    // Tags that load after their group painted fade in, rather than pop.
    stale.forEach((el) => {
      const before = this.fits.get(el)?.tags.split("\n");
      if (before == null) return;
      tagsOf(el).forEach((t) => {
        if (!before.includes(t.textContent ?? "")) t.toggleAttribute(ENTERING, true);
      });
    });
    const template = this.template;
    // Groups whose tags were all seen before fit from the template. The rest measure.
    const unseen = stale.filter((el) => {
      const fit = template == null ? null : estimate(el, template);
      if (fit != null) this.fits.set(el, fit);
      return fit == null;
    });
    // A new group lays out at full width, so its resize can measure it with no extra
    // layout. A group already fitted shows all its tags again, then measures.
    const changed = unseen.filter((el) => el.style.getPropertyValue(STACKED) !== "");
    changed.forEach(expand);
    unseen.forEach((el) => {
      // A hidden group measures once it shows, and its resize refits it.
      if (cssWidth(el) > 0) this.fits.set(el, fit(el));
    });
    changed.forEach((el) => el.removeAttribute(MEASURING));
    unseen.forEach((el) => {
      const f = this.fits.get(el);
      if (f == null || f.measures.length === 0) return;
      this.template ??= {
        tags: new Map(),
        frame: f.frame,
        fixed: f.fixed,
        gap: f.gap,
        rem: px(getComputedStyle(document.documentElement).fontSize),
      };
      learn(this.template, el, f);
    });
    const fitted = els.filter(
      (el) =>
        (this.widths.get(el) ?? 0) > 0 && (this.fits.get(el)?.measures.length ?? 0) > 0,
    );
    if (fitted.length === 0) return;
    const rem =
      this.template?.rem ?? px(getComputedStyle(document.documentElement).fontSize);
    const count = COUNT * rem;
    const fits = fitted.map((el) => this.fits.get(el) as Fit);
    const widths = fitted.map((el) => (this.widths.get(el) ?? 0) + 0.5);
    // The narrowest group sets the room every group shares.
    const room = Math.min(...widths);
    const most = Math.max(...fits.map((f) => f.measures.length));
    let shown = most;
    while (
      shown > 0 &&
      !fits.every((f) => stacks(f, Math.min(shown, f.measures.length), count, room))
    )
      shown--;
    // A lifted tag gives back the width it had, which a refit may change.
    this.drop();
    fitted.forEach((el, i) => write(el, fits[i], shown, widths[i], count));
  }
}

// Sets a property only when it changes, so an unchanged group is not styled again.
const set = (el: HTMLElement, name: string, value: string): void => {
  if (el.style.getPropertyValue(name) !== value) el.style.setProperty(name, value);
};

const write = (
  el: HTMLElement,
  f: Fit,
  step: number,
  room: number,
  count: number,
): void => {
  const tags = tagsOf(el);
  const shown = Math.min(step, tags.length);
  const overlap = shown > 0 && full(f, shown, count) > room;
  set(el, STACKED, String(overlap));
  el.toggleAttribute(EMPTY, shown === 0 && f.frame + count > room);
  const gaps = `${f.gap}px`;
  if (el.style.getPropertyValue("--lyra-tags-gap") !== gaps)
    el.style.setProperty("--lyra-tags-gap", gaps);
  const dots = String(Math.min(DOTS, tags.length - shown));
  if (el.style.getPropertyValue("--lyra-tags-dots") !== dots)
    el.style.setProperty("--lyra-tags-dots", dots);
  tags.forEach((t, i) => {
    const dot = i >= shown && i < shown + DOTS;
    t.toggleAttribute(HIDDEN, i >= shown);
    t.toggleAttribute(LAST, overlap && i === shown - 1);
    t.toggleAttribute(DOT, dot);
  });
  const more = moreOf(el);
  if (more == null) return;
  const hidden = shown < tags.length ? String(tags.length - shown) : undefined;
  if (more.dataset.count === hidden) return;
  if (hidden == null) delete more.dataset.count;
  else more.dataset.count = hidden;
};
