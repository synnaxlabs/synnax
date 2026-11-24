// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const LOOP_MESSAGE = /^ResizeObserver loop/;
const MICROTASKS = 3;
const MAX_CAUSES = 20;
const STACK_FRAMES = 4;
const COMPONENTS = 3;

interface Size {
  width: number;
  height: number;
  /** The content size, which a scrollbar, a border, or padding changes. */
  inner: string;
  /** The padding and border widths around the content box. */
  frame: string;
}

// An element without a layout box, or an inline one, has an auto size, which the
// browser reports as zero.
const px = (value: string): number => parseFloat(value) || 0;

// Measures the boxes that the browser observes, which leave out transforms. The offset
// size rounds away a change of less than a pixel. Chromium writes a computed size to
// six significant digits, so it misses a smaller change on a very large element.
const measure = (element: Element): Size => {
  const style = getComputedStyle(element);
  const padding = {
    x: px(style.paddingLeft) + px(style.paddingRight),
    y: px(style.paddingTop) + px(style.paddingBottom),
  };
  const border = {
    x: px(style.borderLeftWidth) + px(style.borderRightWidth),
    y: px(style.borderTopWidth) + px(style.borderBottomWidth),
  };
  const outer = style.boxSizing === "border-box" ? 0 : 1;
  const width = px(style.width) + outer * (padding.x + border.x);
  const height = px(style.height) + outer * (padding.y + border.y);
  // A scrollbar takes its space from the content box but not from the computed size.
  const bar = { x: 0, y: 0 };
  if (element instanceof HTMLElement) {
    bar.x = Math.max(
      0,
      Math.round(element.offsetWidth - element.clientWidth - border.x),
    );
    bar.y = Math.max(
      0,
      Math.round(element.offsetHeight - element.clientHeight - border.y),
    );
  }
  const contentWidth = width - padding.x - border.x - bar.x;
  const contentHeight = height - padding.y - border.y - bar.y;
  const inner = `${contentWidth}x${contentHeight}`;
  const frame = [
    style.paddingTop,
    style.paddingRight,
    style.paddingBottom,
    style.paddingLeft,
    style.borderTopWidth,
    style.borderRightWidth,
    style.borderBottomWidth,
    style.borderLeftWidth,
  ].join(" ");
  return { width, height, inner, frame };
};

const callerStack = (): string =>
  (new Error().stack ?? "")
    .split("\n")
    .map((line) => line.trim())
    .filter(
      (line) =>
        !["", "Error"].includes(line) && !line.includes("testutil/resizeLoop.ts"),
    )
    .slice(0, STACK_FRAMES)
    .join("\n");

interface Observation {
  box: ResizeObserverBoxOptions;
  /** The number of observations made before this one. */
  order: number;
  /** The size that the browser last reported. */
  reported: Size;
}

interface Pass {
  /** The browser gathers only targets deeper than this. */
  floor: number;
  /** The sizes of the observed elements when the browser gathered the pass. */
  sizes: Map<Element, Size>;
  /** The number of observations made before the browser gathered the pass. */
  observations: number;
  called: Set<ResizeObserver>;
  /** The depth of the shallowest target delivered in the pass. */
  shallowest: number;
}

interface Change {
  text: string;
  depth: number;
  pass: Pass;
}

interface Context {
  origin: string;
  targets: string;
  during: Pass;
}

const ZERO: Size = { width: 0, height: 0, inner: "0x0", frame: "" };

// Layout sizes are multiples of 1/64 of a pixel, and Chromium writes a computed size
// to six significant digits.
const near = (a: number, b: number): boolean =>
  Math.abs(a - b) < Math.max(1 / 128, Math.abs(b) * 1e-5);

const matches = (entry: ResizeObserverEntry, size: Size): boolean => {
  const [{ inlineSize, blockSize }] = entry.borderBoxSize;
  return near(inlineSize, size.width) && near(blockSize, size.height);
};

const formatSize = ({ width, height }: Size): string => `${width}x${height}`;

// The browser counts the element and its ancestors in the flat tree, so an element off
// the page has a depth of at least one.
const flatParent = (element: Element): Element | null => {
  if (element.assignedSlot != null) return element.assignedSlot;
  const parent = element.parentNode;
  return parent instanceof ShadowRoot ? parent.host : element.parentElement;
};

const depth = (element: Element): number => {
  let d = 0;
  for (let e: Element | null = element; e != null; e = flatParent(e)) d++;
  return d;
};

const stringProperty = (value: object, key: string): string | undefined => {
  const property: unknown = Reflect.get(value, key);
  return typeof property === "string" && property !== "" ? property : undefined;
};

const componentName = (type: unknown): string | undefined => {
  if (typeof type === "function")
    return stringProperty(type, "displayName") ?? stringProperty(type, "name");
  if (typeof type === "object" && type != null)
    return (
      stringProperty(type, "displayName") ?? componentName(Reflect.get(type, "render"))
    );
  return undefined;
};

// Reads the internal React fiber of the element to name the components around it.
const componentNames = (element: Element): string[] => {
  const key = Object.keys(element).find((k) => k.startsWith("__reactFiber$"));
  if (key == null) return [];
  const names: string[] = [];
  let fiber: unknown = Reflect.get(element, key);
  while (typeof fiber === "object" && fiber != null && names.length < COMPONENTS) {
    const name = componentName(Reflect.get(fiber, "type"));
    if (name != null && !names.includes(name)) names.push(name);
    fiber = Reflect.get(fiber, "return");
  }
  return names;
};

const label = (element: Element): string => {
  let text = element.tagName.toLowerCase();
  const className = element.getAttribute("class")?.trim();
  if (className != null && className !== "")
    text += `.${className.split(/\s+/).join(".")}`;
  const ariaLabel = element.getAttribute("aria-label");
  if (ariaLabel != null) text += `[aria-label="${ariaLabel}"]`;
  const testID = element.closest("[data-testid]")?.getAttribute("data-testid");
  if (testID != null) text += ` in [data-testid="${testID}"]`;
  const names = componentNames(element);
  if (names.length > 0) text += ` in <${names.join("> in <")}>`;
  return text;
};

/**
 * Wraps the global ResizeObserver to name the cause of each ResizeObserver loop error.
 * Each report lists the observed elements that a callback resized, took off the page,
 * or put back on it in the frame of the error, with the stack that created the
 * observer, and the targets observed after a callback, with the stack that observed
 * them. It lists only those at or above the shallowest target of the loop pass of that
 * callback.
 *
 * @param report - Receives each report.
 * @returns a function that puts back the previous ResizeObserver and stops reporting.
 * Observers created before then stay traced.
 */
export const traceResizeLoops = (report: (text: string) => void): (() => void) => {
  const Native = window.ResizeObserver;
  // The boxes that each traced observer watches on an element.
  const observed = new Map<Element, Map<ResizeObserver, ResizeObserverBoxOptions>>();
  let changes: Change[] = [];
  // Checks the last callback once more, at the next callback or the loop error. The
  // microtasks of a callback end before the next callback starts.
  let late: (() => void) | null = null;
  let frame: CSSNumberish | null = null;
  let pass: Pass | null = null;
  let observations = 0;

  // Every callback and the loop error of one rendering update share a timeline time.
  const enterFrame = (): void => {
    const now = document.timeline.currentTime;
    if (now === frame) return;
    frame = now;
    pass = null;
    changes = [];
    late = null;
  };

  const closeLate = (): void => {
    late?.();
    late = null;
  };

  const changed = (
    box: ResizeObserverBoxOptions,
    before: Size,
    after: Size,
  ): boolean => {
    const border = before.width !== after.width || before.height !== after.height;
    if (box === "border-box") return border;
    return before.inner !== after.inner || (border && before.frame === after.frame);
  };

  // Whether a box that some observer watches on the element changed.
  const resized = (element: Element, before: Size, after: Size): boolean =>
    [...(observed.get(element)?.values() ?? [])].some((box) =>
      changed(box, before, after),
    );

  // Whether the browser found a target of the entries changed when it gathered the
  // pass. If it did, the observer of the entries belongs to that pass.
  const gatheredIn = (
    current: Pass,
    targets: Map<Element, Observation>,
    entries: ResizeObserverEntry[],
  ): boolean =>
    entries.some(({ target }) => {
      const observation = targets.get(target);
      if (observation == null || observation.order >= current.observations)
        return false;
      if (depth(target) <= current.floor) return false;
      const size = current.sizes.get(target) ?? ZERO;
      return changed(observation.box, observation.reported, size);
    });

  const check = (before: Map<Element, Size>, context: Context, phase: string): void => {
    const { origin, targets, during } = context;
    const callback = `${phase} the callback for ${targets}, of the observer`;
    for (const [element, size] of before) {
      if (!element.isConnected) {
        if (size === ZERO) continue;
        before.set(element, ZERO);
        // The browser then reports a zero size from the depth of the element off the
        // page.
        if (observed.has(element) && (size.width > 0 || size.height > 0)) {
          const text = `${label(element)} left the page ${callback} created at:`;
          changes.push({
            text: `${text}\n${origin}`,
            depth: depth(element),
            pass: during,
          });
        }
        continue;
      }
      const after = measure(element);
      if (!resized(element, size, after)) continue;
      const same = after.width === size.width && after.height === size.height;
      // The browser reports an element put back on the page as a change from zero.
      const change =
        size === ZERO
          ? "came back on the page"
          : same
            ? `changed its content size from ${size.inner} to ${after.inner}`
            : `changed from ${formatSize(size)} to ${formatSize(after)}`;
      const text = `${label(element)} ${change} ${callback} created at:`;
      changes.push({ text: `${text}\n${origin}`, depth: depth(element), pass: during });
      before.set(element, after);
    }
  };

  const trace = (
    callback: ResizeObserverCallback,
    origin: string,
    targets: Map<Element, Observation>,
    entries: ResizeObserverEntry[],
    observer: ResizeObserver,
  ): void => {
    enterFrame();
    closeLate();
    const before = new Map<Element, Size>();
    for (const element of observed.keys())
      before.set(element, element.isConnected ? measure(element) : ZERO);
    // The browser gathers the changed targets of every observer before a pass, then
    // calls each observer with any at most once. No script runs between a pass and the
    // gathering of the next.
    if (
      pass == null ||
      pass.called.has(observer) ||
      !gatheredIn(pass, targets, entries)
    )
      pass = {
        floor: pass?.shallowest ?? 0,
        sizes: new Map(before),
        observations,
        called: new Set(),
        shallowest: Infinity,
      };
    pass.called.add(observer);
    pass.shallowest = Math.min(pass.shallowest, ...entries.map((e) => depth(e.target)));
    for (const entry of entries) {
      const observation = targets.get(entry.target);
      if (observation == null) continue;
      const gathered = pass.sizes.get(entry.target) ?? ZERO;
      const current = before.get(entry.target) ?? ZERO;
      // WebKit reports the size from the gathering of the pass, and Chromium the size
      // at the call.
      observation.reported = matches(entry, gathered) ? gathered : current;
    }
    const context: Context = {
      origin,
      targets: entries.map((e) => label(e.target)).join(", "),
      during: pass,
    };
    try {
      callback(entries, observer);
    } finally {
      check(before, context, "inside");
      let remaining = MICROTASKS;
      const next = (): void => {
        check(before, context, "in a microtask after");
        remaining -= 1;
        if (remaining > 0) queueMicrotask(next);
      };
      queueMicrotask(next);
      late = () => check(before, context, "later in the frame of");
    }
  };

  class Traced implements ResizeObserver {
    private readonly observer: ResizeObserver;
    private readonly targets = new Map<Element, Observation>();

    constructor(callback: ResizeObserverCallback) {
      const origin = callerStack();
      this.observer = new Native((entries) =>
        trace(callback, origin, this.targets, entries, this),
      );
    }

    observe(target: Element, options?: ResizeObserverOptions): void {
      const box = options?.box ?? "content-box";
      // The browser ignores a target observed again with the same box. A new target or
      // box reports its first size, so a shallow one ends the frame in a loop error.
      if (this.targets.get(target)?.box !== box) {
        if (pass != null && document.timeline.currentTime === frame) {
          const cause = "began to be observed after a callback, at:";
          const text = `${label(target)} ${cause}\n${callerStack()}`;
          changes.push({ text, depth: depth(target), pass });
        }
        this.targets.set(target, { box, order: observations++, reported: ZERO });
        const boxes = observed.get(target) ?? new Map();
        boxes.set(this, box);
        observed.set(target, boxes);
      }
      this.observer.observe(target, options);
    }

    unobserve(target: Element): void {
      this.release(target);
      this.observer.unobserve(target);
    }

    disconnect(): void {
      for (const target of this.targets.keys()) this.release(target);
      this.observer.disconnect();
    }

    private release(target: Element): void {
      if (!this.targets.delete(target)) return;
      const boxes = observed.get(target);
      boxes?.delete(this);
      if (boxes?.size === 0) observed.delete(target);
    }
  }

  // The browser skips a change in a pass at or above the shallowest target of that
  // pass, and a skipped change ends the frame in the loop error.
  const onError = (event: Event): void => {
    if (!(event instanceof ErrorEvent) || !LOOP_MESSAGE.test(event.message)) return;
    enterFrame();
    closeLate();
    const skipped = changes
      .filter((c) => c.depth <= c.pass.shallowest)
      .map((c) => c.text);
    const omitted = skipped.length - MAX_CAUSES;
    if (omitted > 0) skipped.splice(MAX_CAUSES, omitted, `And ${omitted} more.`);
    const causes =
      skipped.length === 0
        ? "No observed element at or above the targets of a callback changed size."
        : skipped.join("\n");
    changes = [];
    report(`ResizeObserver loop diagnostic:\n${causes}`);
  };

  window.ResizeObserver = Traced;
  // Vitest stops reporting unhandled errors while a listener added through
  // window.addEventListener exists. The prototype methods are not counted.
  EventTarget.prototype.addEventListener.call(window, "error", onError);
  return () => {
    window.ResizeObserver = Native;
    EventTarget.prototype.removeEventListener.call(window, "error", onError);
  };
};
