// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Seconds of signal before the signal stops. */
export const HISTORY = 20;
/** Seconds from the end of the signal to the first frame. */
export const GAP = 6;
/** The top of the y axis. */
export const MAX = 60;
const RATE = 20;
const STEPS = [1, 2, 5, 10, 15, 30, 60, 120, 300, 600, 900, 1800, 3600];
// The fewest pixels between two time ticks.
const TICK_SPACING = 80;
const REDUCED_MOTION = "(prefers-reduced-motion: reduce)";

// Seeded, so the trace the server draws and the values the cursor reads agree.
const random = (seed: number): (() => number) => {
  let state = seed;
  return () => {
    state = (state + 0x6d2b79f5) | 0;
    let t = Math.imul(state ^ (state >>> 15), 1 | state);
    t ^= t + Math.imul(t ^ (t >>> 7), 61 | t);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
};

const smoothstep = (from: number, to: number, x: number): number => {
  const t = Math.min(Math.max((x - from) / (to - from), 0), 1);
  return t * t * (3 - 2 * t);
};

/**
 * Returns a pressurization, sampled from HISTORY seconds before the end of the signal
 * to its end.
 */
export const samples = (): number[] => {
  const next = random(404);
  return Array.from({ length: HISTORY * RATE + 1 }, (_, i) => {
    const t = i / RATE - HISTORY;
    const rise = smoothstep(-15, -11.5, t);
    const settled = t + 11.5;
    const ring =
      settled > 0 ? 6 * Math.exp(-settled / 2.2) * Math.sin(settled * 3.4) : 0;
    const drift = 1.8 * Math.sin(t * 1.9) * rise;
    return 8 + 34 * rise + ring + drift + (next() - 0.5) * 1.6;
  });
};

/** Returns the height of a value as a percentage from the top of the plot. */
export const top = (value: number): number => 100 - (value / MAX) * 100;

/** Returns the trace of samples as SVG path data, with x in seconds. */
export const path = (values: number[]): string =>
  values
    .map((v, i) => {
      const x = (i / RATE - HISTORY).toFixed(2);
      return `${i === 0 ? "M" : "L"}${x} ${top(v).toFixed(2)}`;
    })
    .join("");

/** Returns the transform that fits the trace into a plot that spans span seconds. */
export const transform = (span: number): string =>
  `scale(${1000 / span} 1) translate(${HISTORY} 0)`;

/** Returns at most count tick times in milliseconds: whole steps from start to end. */
const ticks = (start: number, end: number, count: number): number[] => {
  const seconds = (end - start) / 1000;
  const step =
    (STEPS.find((s) => seconds / s <= count) ?? STEPS[STEPS.length - 1]) * 1000;
  const times: number[] = [];
  for (let t = Math.ceil(start / step) * step; t <= end; t += step) times.push(t);
  return times;
};

const pad = (n: number, width = 2): string => String(n).padStart(width, "0");

/** Labels a tick as the Console does: ":SS" in a minute, "HH:MM" on the minute. */
const tickLabel = (ms: number): string => {
  const d = new Date(ms);
  if (d.getSeconds() === 0) return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
  return `:${pad(d.getSeconds())}`;
};

const timeLabel = (ms: number): string => {
  const d = new Date(ms);
  const clock = [d.getHours(), d.getMinutes(), d.getSeconds()].map((n) => pad(n));
  return `${clock.join(":")}.${pad(d.getMilliseconds(), 3)}`;
};

/** Decodes a pathname for display, or returns it as is when it does not decode. */
export const decode = (pathname: string): string => {
  try {
    return decodeURIComponent(pathname);
  } catch {
    return pathname;
  }
};

const query = <E extends Element>(root: Element, selector: string): E => {
  const el = root.querySelector<E>(selector);
  if (el == null) throw new Error(`no element matches ${selector}`);
  return el;
};

const run = (root: HTMLElement): (() => void) => {
  const plot = query<HTMLElement>(root, ".lost-signal__plot");
  const trace = query<SVGGElement>(root, ".lost-signal__trace");
  const tickBox = query<HTMLElement>(root, ".lost-signal__ticks");
  const cursor = query<HTMLElement>(root, ".lost-signal__cursor");
  const time = query<HTMLElement>(root, "[data-time]");
  const value = query<HTMLElement>(root, "[data-value]");
  const toggle = query<HTMLButtonElement>(root, ".lost-signal__live");
  for (const label of root.querySelectorAll("[data-pathname]"))
    label.textContent = decode(location.pathname);

  const values = samples();
  const opened = performance.now();
  const lostAt = Date.now() - GAP * 1000;
  let width = plot.clientWidth;
  let span = HISTORY + GAP;
  let pointer: number | null = null;
  let visible = true;
  let paused = matchMedia(REDUCED_MOTION).matches;
  let frame = 0;
  const shown = new Map<number, HTMLElement>();

  const drawTicks = (): void => {
    const start = lostAt - HISTORY * 1000;
    const end = start + span * 1000;
    const next = ticks(start, end, Math.max(2, Math.floor(width / TICK_SPACING)));
    for (const [t, el] of shown)
      if (!next.includes(t)) {
        el.remove();
        shown.delete(t);
      }
    for (const t of next) {
      let el = shown.get(t);
      if (el == null) {
        el = document.createElement("span");
        el.className = "lost-signal__tick";
        el.dataset.label = tickLabel(t);
        tickBox.append(el);
        shown.set(t, el);
      }
      el.style.left = `${((t - start) / (end - start)) * 100}%`;
    }
  };

  const drawCursor = (): void => {
    cursor.hidden = pointer == null;
    if (pointer == null) return;
    const t = pointer * span - HISTORY;
    const empty = t > 0;
    cursor.style.left = `${pointer * 100}%`;
    cursor.classList.toggle("lost-signal__cursor--empty", empty);
    cursor.classList.toggle("lost-signal__cursor--flipped", pointer > 0.6);
    time.textContent = timeLabel(lostAt + t * 1000);
    if (empty) {
      value.textContent = "No data";
      return;
    }
    const v = values[Math.round((t + HISTORY) * RATE)];
    value.textContent = v.toFixed(2);
    cursor.style.setProperty("--lost-signal-value", `${top(v)}%`);
  };

  const draw = (now: number): void => {
    if (!paused) span = HISTORY + GAP + (now - opened) / 1000;
    root.style.setProperty("--lost-signal-loss", `${(HISTORY / span) * 100}%`);
    trace.setAttribute("transform", transform(span));
    drawTicks();
    drawCursor();
  };

  const loop = (now: number): void => {
    draw(now);
    frame = requestAnimationFrame(loop);
  };

  const sync = (): void => {
    cancelAnimationFrame(frame);
    root.classList.toggle("lost-signal--paused", paused);
    toggle.setAttribute("aria-pressed", String(!paused));
    if (!paused && visible && !document.hidden) frame = requestAnimationFrame(loop);
    else draw(performance.now());
  };

  const resize = new ResizeObserver(([entry]) => {
    width = entry.contentRect.width;
    if (paused) draw(performance.now());
  });
  resize.observe(plot);
  const intersection = new IntersectionObserver(([entry]) => {
    visible = entry.isIntersecting;
    sync();
  });
  intersection.observe(root);
  document.addEventListener("visibilitychange", sync);

  const point = (e: PointerEvent): void => {
    const rect = plot.getBoundingClientRect();
    pointer = Math.min(Math.max((e.clientX - rect.left) / rect.width, 0), 1);
    drawCursor();
  };
  plot.addEventListener("pointermove", point);
  plot.addEventListener("pointerdown", point);
  plot.addEventListener("pointerleave", () => {
    pointer = null;
    drawCursor();
  });
  toggle.addEventListener("click", () => {
    paused = !paused;
    sync();
  });

  sync();
  return () => {
    cancelAnimationFrame(frame);
    resize.disconnect();
    intersection.disconnect();
    document.removeEventListener("visibilitychange", sync);
  };
};

/**
 * Starts the lost signal plot on the page: its time axis follows the clock, and a
 * pointer over it reads values. Call it once; it follows page swaps by itself.
 */
export const start = (): void => {
  let stop: (() => void) | undefined;
  const bind = (): void => {
    stop?.();
    const root = document.querySelector<HTMLElement>(".lost-signal");
    stop = root == null ? undefined : run(root);
  };
  document.addEventListener("astro:after-swap", bind);
  bind();
};
