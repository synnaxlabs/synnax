// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/nebula/Nebula.css";

import { color } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useRef } from "react";

import { CSS } from "@/css";
import { Theming } from "@/theming";

// Module scope, so one seed lasts the page load and the nebula holds still as the
// surfaces above it swap.
const SEEDS = [
  7, 25, 61, 99, 122, 147, 225, 315, 483, 777, 1024, 2718, 3141, 4222, 5077, 6502, 7919,
  8674, 9253,
];
const SEED = SEEDS[Math.floor(Math.random() * SEEDS.length)];

/** Dot grid spacing in px. */
const PITCH = 5.5;
/** Dot diameter at full brightness as a fraction of the pitch; past ~1.1 dots merge. */
const DOT_SCALE = 1.05;
/** Cloud feature size; higher means smaller billows. */
const NOISE_SCALE = 2.8 / 1000;
/** Brightness curve exponent; higher means harder cloud edges. */
const GAMMA = 1.9;
/** Brightness cutoff that removes speckle in dark regions. */
const FLOOR = 0.08;
/** Strength of the falloff toward the top left. */
const FADE = 0.25;
/** Smallest crisply rendered radius; smaller dots hold it and fade by alpha. */
const MIN_RADIUS = 0.45;

interface Ink {
  color: string;
  intensity: number;
}

// Light backgrounds want darker ink.
const DARK_INK: Ink = { color: "#e6e6ea", intensity: 0.45 };
const LIGHT_INK: Ink = { color: "#63666c", intensity: 0.6 };

const hash = (x: number, y: number, seed: number): number => {
  let h = Math.imul(x, 0x27d4eb2d) ^ Math.imul(y, 0x165667b1) ^ seed;
  h = Math.imul(h ^ (h >>> 15), 0x85ebca6b);
  h ^= h >>> 13;
  h = Math.imul(h, 0xc2b2ae35);
  h ^= h >>> 16;
  return (h >>> 0) / 4294967295;
};

const smooth = (t: number): number => t * t * (3 - 2 * t);

const valueNoise = (x: number, y: number, seed: number): number => {
  const xi = Math.floor(x);
  const yi = Math.floor(y);
  const xf = x - xi;
  const yf = y - yi;
  const a = hash(xi, yi, seed);
  const b = hash(xi + 1, yi, seed);
  const c = hash(xi, yi + 1, seed);
  const d = hash(xi + 1, yi + 1, seed);
  const u = smooth(xf);
  const v = smooth(yf);
  return a + (b - a) * u + (c - a) * v + (a - b - c + d) * u * v;
};

const OCTAVES = 4;

const fbm = (x: number, y: number, seed: number): number => {
  let sum = 0;
  let amp = 0.5;
  let freq = 1;
  let norm = 0;
  for (let o = 0; o < OCTAVES; o++) {
    sum += amp * valueNoise(x * freq, y * freq, seed + o * 101);
    norm += amp;
    amp *= 0.5;
    freq *= 2.1;
  }
  return sum / norm;
};

// Reads the resolved surface color, so the ink follows a Theming provider and a static
// theme stylesheet alike.
const inkOf = (canvas: HTMLCanvasElement): Ink =>
  color.isLight(getComputedStyle(canvas).getPropertyValue("--pluto-gray-l0").trim())
    ? LIGHT_INK
    : DARK_INK;

const draw = (canvas: HTMLCanvasElement, seed: number): void => {
  const ctx = canvas.getContext("2d");
  if (ctx == null) return;
  const ink = inkOf(canvas);
  const dpr = window.devicePixelRatio;
  const w = canvas.clientWidth;
  const h = canvas.clientHeight;
  // Resizing the canvas also clears it and resets the context state.
  canvas.width = w * dpr;
  canvas.height = h * dpr;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.fillStyle = ink.color;
  const cols = Math.ceil(w / PITCH) + 2;
  const rows = Math.ceil(h / PITCH) + 2;
  for (let j = 0; j < rows; j++)
    for (let i = 0; i < cols; i++) {
      // Odd rows shift half a pitch into a diamond lattice.
      const x = i * PITCH + (j % 2) * (PITCH / 2);
      const y = j * PITCH;
      let v = fbm(x * NOISE_SCALE, y * NOISE_SCALE, seed);
      v = Math.max(0, (v - 0.5) * 1.8 + 0.5);
      v *= 1 - FADE * Math.min(1, (x / w) * 0.6 + (y / h) * 0.8);
      v = Math.max(0, (v - FLOOR) / (1 - FLOOR)) ** GAMMA;
      const r = (v * PITCH * DOT_SCALE) / 2;
      if (r < 0.1) continue;
      // Sub-pixel arcs render as fuzzy smears on WebKit, and any hard on/off
      // cutoff draws a visible level-set rim. Dots below the clean-rendering
      // radius keep that radius and fade by alpha instead.
      let radius = r;
      let alpha = ink.intensity;
      if (r < MIN_RADIUS) {
        radius = MIN_RADIUS;
        alpha = ink.intensity * (r / MIN_RADIUS) ** 2;
        if (alpha < 0.02) continue;
      }
      ctx.globalAlpha = alpha;
      ctx.beginPath();
      ctx.arc(x, y, Math.min(radius, PITCH * 0.72), 0, Math.PI * 2);
      ctx.fill();
    }
};

/**
 * Static halftone nebula: a noise cloud rendered through a dot screen, where dot
 * size follows cloud brightness. Fills its positioned parent and redraws on resize
 * and on a theme change; costs nothing at rest.
 */
export const Nebula = (): ReactElement => {
  const ref = useRef<HTMLCanvasElement>(null);
  const theme = Theming.use();
  useEffect(() => {
    const canvas = ref.current;
    if (canvas == null) return;
    // A single draw walks tens of thousands of dots, and a window drag notifies
    // the observer every frame, so at most one draw is ever pending.
    let frame: number | null = null;
    const schedule = (): void => {
      if (frame != null) return;
      frame = requestAnimationFrame(() => {
        frame = null;
        draw(canvas, SEED);
      });
    };
    // observe reports the first size at once, so re-subscribing on a theme change
    // redraws.
    const observer = new ResizeObserver(schedule);
    observer.observe(canvas);
    const scheme = window.matchMedia("(prefers-color-scheme: dark)");
    scheme.addEventListener("change", schedule);
    return () => {
      observer.disconnect();
      scheme.removeEventListener("change", schedule);
      if (frame != null) cancelAnimationFrame(frame);
    };
  }, [theme]);
  return <canvas ref={ref} className={CSS.B("nebula")} />;
};
