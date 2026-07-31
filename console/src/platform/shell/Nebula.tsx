// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/shell/Nebula.css";

import { Theming } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useMemo, useRef, useState } from "react";

import { CSS } from "@/platform/css";

// Curated seeds; one is drawn per launch. Audition new candidates through the
// tuner's seed input.
const SEEDS = [
  7, 25, 61, 99, 122, 147, 225, 315, 483, 777, 1024, 2718, 3141, 4222, 5077, 6502, 7919,
  8674, 9253,
];
const SEED = SEEDS[Math.floor(Math.random() * SEEDS.length)];

// TEMPORARY: renders a live slider panel over the shell for tuning SETTINGS.
// Flip off (and eventually delete the Tuner) once values are finalized.
const TUNER = false as boolean;

type Lattice = "square" | "diamond" | "hex";
type FadeDirection = "top" | "topLeft" | "left" | "none";

interface Settings {
  /** Dot grid spacing in px: small = fine mesh, large = chunky print dots. */
  pitch: number;
  /** Dot diameter at full brightness as a fraction of pitch; past ~1.1 dots merge. */
  dotScale: number;
  /** Cloud feature size; higher = smaller billows. */
  cloudScale: number;
  /** Brightness curve exponent; higher = harder cloud edges. */
  gamma: number;
  /** Brightness cutoff that removes speckle in dark regions. */
  floor: number;
  /** Global dot opacity. */
  intensity: number;
  /** Strength of the directional falloff. */
  fade: number;
  /** Center darkening so the login card sits on quiet ground. */
  clearing: number;
  /** Smallest crisply-rendered radius; smaller dots hold it and fade by alpha. */
  minRadius: number;
  lattice: Lattice;
  fadeDir: FadeDirection;
  color: string;
}

const SETTINGS: Settings = {
  pitch: 5.5,
  dotScale: 1.05,
  cloudScale: 2.8,
  gamma: 1.9,
  floor: 0.08,
  intensity: 0.45,
  fade: 0.25,
  clearing: 0,
  minRadius: 0.45,
  lattice: "diamond",
  fadeDir: "topLeft",
  color: "#e6e6ea",
};

// SETTINGS is the dark-mode tuning; light backgrounds carry darker ink at a
// softer intensity, like halftone print on paper.
const LIGHT_OVERRIDES: Partial<Settings> = { color: "#63666c", intensity: 0.6 };

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

const fadeWeight = (nx: number, ny: number, dir: FadeDirection): number => {
  switch (dir) {
    case "top":
      return 1 - ny;
    case "left":
      return 1 - nx;
    case "topLeft":
      return 1 - Math.min(1, nx * 0.6 + ny * 0.8);
    case "none":
      return 1;
  }
};

const draw = (canvas: HTMLCanvasElement, s: Settings, seed: number): void => {
  const ctx = canvas.getContext("2d");
  if (ctx == null) return;
  const dpr = window.devicePixelRatio;
  const w = canvas.clientWidth;
  const h = canvas.clientHeight;
  canvas.width = w * dpr;
  canvas.height = h * dpr;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, w, h);
  const { pitch, dotScale, gamma, floor, intensity, fade, clearing, lattice } = s;
  ctx.fillStyle = s.color;
  ctx.globalAlpha = intensity;
  const noiseScale = s.cloudScale / 1000;
  const cols = Math.ceil(w / pitch) + 2;
  const rows = Math.ceil(h / pitch) + 2;
  const cx = w / 2;
  const cy = h / 2;
  const clearRadius = Math.min(w, h) * 0.42;
  for (let j = 0; j < rows; j++)
    for (let i = 0; i < cols; i++) {
      let x = i * pitch;
      let y = j * pitch;
      if (lattice !== "square" && j % 2 === 1) x += pitch / 2;
      if (lattice === "hex") y = j * pitch * 0.866;
      let v = fbm(x * noiseScale, y * noiseScale, seed);
      v = Math.max(0, (v - 0.5) * 1.8 + 0.5);
      v *= 1 - fade + fade * fadeWeight(x / w, y / h, s.fadeDir);
      const dist = Math.hypot(x - cx, y - cy);
      if (dist < clearRadius) v *= 1 - clearing * smooth(1 - dist / clearRadius);
      v = Math.max(0, (v - floor) / (1 - floor)) ** gamma;
      const r = (v * pitch * dotScale) / 2;
      if (r < 0.1) continue;
      // Sub-pixel arcs render as fuzzy smears on WebKit, and any hard on/off
      // cutoff draws a visible level-set rim. Dots below the clean-rendering
      // radius keep that radius and fade by alpha instead.
      let radius = r;
      let alpha = intensity;
      if (s.minRadius > 0 && r < s.minRadius) {
        radius = s.minRadius;
        alpha = intensity * (r / s.minRadius) ** 2;
        if (alpha < 0.02) continue;
      }
      ctx.globalAlpha = alpha;
      ctx.beginPath();
      ctx.arc(x, y, Math.min(radius, pitch * 0.72), 0, Math.PI * 2);
      ctx.fill();
    }
  ctx.globalAlpha = 1;
};

type NumericKey =
  | "pitch"
  | "dotScale"
  | "cloudScale"
  | "gamma"
  | "floor"
  | "intensity"
  | "fade"
  | "clearing"
  | "minRadius";

const SLIDERS: Array<{ key: NumericKey; min: number; max: number; step: number }> = [
  { key: "pitch", min: 2, max: 14, step: 0.5 },
  { key: "dotScale", min: 0.4, max: 1.6, step: 0.05 },
  { key: "cloudScale", min: 0.5, max: 8, step: 0.1 },
  { key: "gamma", min: 0.5, max: 4, step: 0.05 },
  { key: "floor", min: 0, max: 0.4, step: 0.01 },
  { key: "intensity", min: 0.1, max: 1, step: 0.05 },
  { key: "fade", min: 0, max: 1, step: 0.05 },
  { key: "clearing", min: 0, max: 1, step: 0.05 },
  { key: "minRadius", min: 0, max: 1, step: 0.05 },
];

interface TunerProps {
  settings: Settings;
  seed: number;
  onChange: (settings: Settings) => void;
  onSeedChange: (seed: number) => void;
}

const Tuner = ({
  settings,
  seed,
  onChange,
  onSeedChange,
}: TunerProps): ReactElement => (
  <div className={CSS.B("nebula-tuner")}>
    <label>
      seed {seed}
      <input
        type="number"
        value={seed}
        onChange={(e) => onSeedChange(Number(e.target.value))}
      />
    </label>
    {SLIDERS.map(({ key, min, max, step }) => (
      <label key={key}>
        {key} {settings[key]}
        <input
          type="range"
          min={min}
          max={max}
          step={step}
          value={settings[key]}
          onChange={(e) => onChange({ ...settings, [key]: Number(e.target.value) })}
        />
      </label>
    ))}
    <label>
      lattice
      <select
        value={settings.lattice}
        onChange={(e) => onChange({ ...settings, lattice: e.target.value as Lattice })}
      >
        <option value="square">square</option>
        <option value="diamond">diamond</option>
        <option value="hex">hex</option>
      </select>
    </label>
    <label>
      fade dir
      <select
        value={settings.fadeDir}
        onChange={(e) =>
          onChange({ ...settings, fadeDir: e.target.value as FadeDirection })
        }
      >
        <option value="top">top</option>
        <option value="topLeft">topLeft</option>
        <option value="left">left</option>
        <option value="none">none</option>
      </select>
    </label>
    <label>
      color
      <input
        type="color"
        value={settings.color}
        onChange={(e) => onChange({ ...settings, color: e.target.value })}
      />
    </label>
    <button
      onClick={() =>
        void navigator.clipboard.writeText(
          JSON.stringify({ seed, ...settings }, null, 2),
        )
      }
    >
      Copy settings
    </button>
  </div>
);

/**
 * Static halftone nebula: a noise cloud rendered through a dot screen, where dot
 * size follows cloud brightness. Draws once per resize; costs nothing at rest.
 */
export const Nebula = (): ReactElement => {
  const ref = useRef<HTMLCanvasElement>(null);
  const theme = Theming.use();
  const themed = useMemo(
    () =>
      color.isLight(theme.colors.gray.l0)
        ? { ...SETTINGS, ...LIGHT_OVERRIDES }
        : SETTINGS,
    [theme],
  );
  const [tuned, setTuned] = useState<Settings | null>(null);
  const settings = tuned ?? themed;
  const [seed, setSeed] = useState(SEED);
  const settingsRef = useRef(settings);
  settingsRef.current = settings;
  const seedRef = useRef(seed);
  seedRef.current = seed;
  useEffect(() => {
    const canvas = ref.current;
    if (canvas == null) return;
    const observer = new ResizeObserver(() =>
      draw(canvas, settingsRef.current, seedRef.current),
    );
    observer.observe(canvas);
    return () => observer.disconnect();
  }, []);
  useEffect(() => {
    if (ref.current != null) draw(ref.current, settings, seed);
  }, [settings, seed]);
  return (
    <>
      <canvas ref={ref} className={CSS.B("nebula")} />
      {TUNER && (
        <Tuner
          settings={settings}
          seed={seed}
          onChange={setTuned}
          onSeedChange={setSeed}
        />
      )}
    </>
  );
};
