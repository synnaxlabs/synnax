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
const MAX_CHANGES = 20;
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

// A ResizeObserver loop can come from a change of less than a pixel. The offset size
// rounds it away, and Chromium writes a computed size to six digits, so only the
// layout box keeps it. A callback that changes a scale reads as a resize.
const measure = (element: Element): Size => {
  const { width, height } = element.getBoundingClientRect();
  const style = getComputedStyle(element);
  const x = parseFloat(style.paddingLeft) + parseFloat(style.paddingRight);
  const y = parseFloat(style.paddingTop) + parseFloat(style.paddingBottom);
  const inner = `${element.clientWidth - x}x${element.clientHeight - y}`;
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

interface Pass {
  /** The depth of the shallowest target delivered in the pass. */
  shallowest: number;
  /** The creation order of the last observer called in the pass. */
  last: number;
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
 * Each report lists the observed elements that a callback resized or took off the page
 * in the frame of the error, with the stack that created the observer, and the targets
 * observed after a callback, with the stack that observed them. It lists only those at
 * or above the shallowest target of the loop pass of that callback.
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
  let created = 0;

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

  const record = (text: string, at: number, during: Pass): void => {
    changes.push({ text, depth: at, pass: during });
    if (changes.length > MAX_CHANGES) changes.shift();
  };

  // Whether a box that some observer watches on the element changed.
  const resized = (element: Element, before: Size, after: Size): boolean => {
    const border = before.width !== after.width || before.height !== after.height;
    const content =
      before.inner !== after.inner || (border && before.frame === after.frame);
    for (const box of observed.get(element)?.values() ?? [])
      if (box === "border-box" ? border : content) return true;
    return false;
  };

  const check = (before: Map<Element, Size>, context: Context, phase: string): void => {
    const { origin, targets, during } = context;
    const callback = `${phase} the callback for ${targets}, of the observer`;
    for (const [element, size] of before) {
      if (!element.isConnected) {
        before.delete(element);
        // The browser then reports a zero size from the depth of the element off the
        // page.
        if (observed.has(element) && (size.width > 0 || size.height > 0)) {
          const text = `${label(element)} left the page ${callback} created at:`;
          record(`${text}\n${origin}`, depth(element), during);
        }
        continue;
      }
      const after = measure(element);
      if (!resized(element, size, after)) continue;
      const same = after.width === size.width && after.height === size.height;
      const resize = same
        ? `its content size from ${size.inner} to ${after.inner}`
        : `from ${formatSize(size)} to ${formatSize(after)}`;
      const text = `${label(element)} changed ${resize} ${callback} created at:`;
      record(`${text}\n${origin}`, depth(element), during);
      before.set(element, after);
    }
  };

  const trace = (
    callback: ResizeObserverCallback,
    origin: string,
    order: number,
    entries: ResizeObserverEntry[],
    observer: ResizeObserver,
  ): void => {
    enterFrame();
    closeLate();
    // The browser calls each observer at most once per pass, in the order they were
    // created.
    if (pass == null || order <= pass.last) pass = { shallowest: Infinity, last: -1 };
    pass.last = order;
    pass.shallowest = Math.min(pass.shallowest, ...entries.map((e) => depth(e.target)));
    const context: Context = {
      origin,
      targets: entries.map((e) => label(e.target)).join(", "),
      during: pass,
    };
    const before = new Map<Element, Size>();
    for (const element of observed.keys())
      if (element.isConnected) before.set(element, measure(element));
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
    private readonly targets = new Map<Element, ResizeObserverBoxOptions>();

    constructor(callback: ResizeObserverCallback) {
      const origin = callerStack();
      const order = created++;
      this.observer = new Native((entries) =>
        trace(callback, origin, order, entries, this),
      );
    }

    observe(target: Element, options?: ResizeObserverOptions): void {
      const box = options?.box ?? "content-box";
      // The browser ignores a target observed again with the same box. A new target or
      // box reports its first size, so a shallow one ends the frame in a loop error.
      if (this.targets.get(target) !== box) {
        if (pass != null && document.timeline.currentTime === frame) {
          const cause = "began to be observed after a callback, at:";
          record(`${label(target)} ${cause}\n${callerStack()}`, depth(target), pass);
        }
        this.targets.set(target, box);
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
    const skipped = changes.filter((c) => c.depth <= c.pass.shallowest);
    const causes =
      skipped.length === 0
        ? "No observed element at or above the targets of a callback changed size."
        : skipped.map((c) => c.text).join("\n");
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
