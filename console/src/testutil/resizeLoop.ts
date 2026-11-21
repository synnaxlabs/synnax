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
}

const measure = (element: Element): Size => {
  const { width, height } = element.getBoundingClientRect();
  return { width, height };
};

const formatSize = ({ width, height }: Size): string => `${width}x${height}`;

const depth = (node: Node): number => {
  let d = 0;
  for (let n = node.parentNode; n != null; n = n.parentNode) d++;
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
 * Wraps ResizeObserver so that a ResizeObserver loop error logs, through console.info,
 * each observed element that a callback resized at or above the shallowest target of
 * that callback, with the stack that created the observer. Call it once from a browser
 * setup file.
 */
export const traceResizeLoops = (): void => {
  const Native = window.ResizeObserver;
  const observed = new Map<Element, number>();
  const changes: string[] = [];

  const check = (
    before: Map<Element, Size>,
    origin: string,
    targets: string,
    phase: string,
  ): void => {
    for (const [element, size] of before) {
      if (!element.isConnected) continue;
      const after = measure(element);
      if (after.width === size.width && after.height === size.height) continue;
      const resize = `from ${formatSize(size)} to ${formatSize(after)}`;
      const callback = `${phase} the callback for ${targets}, of the observer`;
      changes.push(
        `${label(element)} changed ${resize} ${callback} created at:\n${origin}`,
      );
      if (changes.length > MAX_CHANGES) changes.shift();
      before.set(element, after);
    }
  };

  const trace = (
    callback: ResizeObserverCallback,
    origin: string,
    entries: ResizeObserverEntry[],
    observer: ResizeObserver,
  ): void => {
    const shallowest = Math.min(...entries.map((e) => depth(e.target)));
    const before = new Map<Element, Size>();
    for (const element of observed.keys())
      if (element.isConnected && depth(element) <= shallowest)
        before.set(element, measure(element));
    const targets = entries.map((e) => label(e.target)).join(", ");
    try {
      callback(entries, observer);
    } finally {
      check(before, origin, targets, "inside");
      let remaining = MICROTASKS;
      const next = (): void => {
        check(before, origin, targets, "in a microtask after");
        remaining -= 1;
        if (remaining > 0) queueMicrotask(next);
      };
      queueMicrotask(next);
    }
  };

  class Traced implements ResizeObserver {
    private readonly observer: ResizeObserver;
    private readonly targets = new Set<Element>();

    constructor(callback: ResizeObserverCallback) {
      const origin = (new Error().stack ?? "")
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => !["", "Error"].includes(line) && !line.includes("resizeLoop"))
        .slice(0, STACK_FRAMES)
        .join("\n");
      this.observer = new Native((entries) => trace(callback, origin, entries, this));
    }

    observe(target: Element, options?: ResizeObserverOptions): void {
      if (!this.targets.has(target)) {
        this.targets.add(target);
        observed.set(target, (observed.get(target) ?? 0) + 1);
      }
      this.observer.observe(target, options);
    }

    unobserve(target: Element): void {
      this.release(target);
      this.observer.unobserve(target);
    }

    disconnect(): void {
      for (const target of this.targets) this.release(target);
      this.observer.disconnect();
    }

    private release(target: Element): void {
      if (!this.targets.delete(target)) return;
      const count = (observed.get(target) ?? 1) - 1;
      if (count === 0) observed.delete(target);
      else observed.set(target, count);
    }
  }

  window.ResizeObserver = Traced;
  // Vitest stops reporting unhandled errors while a listener added through
  // window.addEventListener exists. The prototype method is not counted.
  EventTarget.prototype.addEventListener.call(window, "error", (event: Event) => {
    if (!(event instanceof ErrorEvent) || !LOOP_MESSAGE.test(event.message)) return;
    const report =
      changes.length === 0
        ? "No observed element at or above the targets of a callback changed size."
        : changes.join("\n");
    changes.length = 0;
    console.info(`ResizeObserver loop diagnostic:\n${report}`);
  });
};
