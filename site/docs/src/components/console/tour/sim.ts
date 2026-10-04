// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The sample project's tank, stepped the way its Tank simulator and Press sequence Arc
// automations step it, so the tour's schematic and plot move like the real ones.

export const PERIOD_S = 0.1;
export const WINDOW_S = 30;
const IDLE_S = 3;
const HOLD_S = 10;

export type Stage = "idle" | "pressurize" | "hold" | "vent";

export interface Sample {
  t: number;
  pressure: number;
  press: number;
  vent: number;
}

export interface State {
  t: number;
  pressure: number;
  stage: Stage;
  stageStart: number;
  history: Sample[];
}

export const create = (): State => ({
  t: 0,
  pressure: 0,
  stage: "idle",
  stageStart: 0,
  history: [],
});

const commands = (stage: Stage): { press: number; vent: number } => ({
  press: stage === "pressurize" ? 1 : 0,
  vent: stage === "vent" ? 1 : 0,
});

const nextStage = (s: State): Stage => {
  const elapsed = s.t - s.stageStart;
  switch (s.stage) {
    case "idle":
      return elapsed >= IDLE_S ? "pressurize" : "idle";
    case "pressurize":
      return s.pressure > 300 ? "hold" : "pressurize";
    case "hold":
      return elapsed >= HOLD_S ? "vent" : "hold";
    case "vent":
      return s.pressure < 5 ? "idle" : "vent";
  }
};

/** Advances the tank by one simulator period. */
export const advance = (s: State): void => {
  const { press, vent } = commands(s.stage);
  if (press === 1) s.pressure += (500 - s.pressure) * 0.02;
  if (vent === 1) s.pressure -= s.pressure * 0.05;
  s.pressure -= s.pressure * 0.0002;
  s.t = Math.round((s.t + PERIOD_S) * 10) / 10;
  s.history.push({ t: s.t, pressure: s.pressure, press, vent });
  while (s.history.length > 0 && s.history[0].t < s.t - WINDOW_S) s.history.shift();
  const next = nextStage(s);
  if (next !== s.stage) {
    s.stage = next;
    s.stageStart = s.t;
  }
};

/** Returns a tank that has run for the given number of seconds. */
export const run = (seconds: number): State => {
  const s = create();
  for (let i = 0; i < seconds / PERIOD_S; i++) advance(s);
  return s;
};

export const current = (s: State): Sample =>
  s.history.at(-1) ?? { t: s.t, pressure: s.pressure, press: 0, vent: 0 };

export const LINES = [
  { key: "pressure", name: "sample_tank_pt", color: "#DC136C", axis: "y1" },
  { key: "press", name: "sample_press_vlv_state", color: "#20A4F3", axis: "y2" },
  { key: "vent", name: "sample_vent_vlv_state", color: "#7AC74F", axis: "y2" },
] as const;

const MARGIN = { left: 38, right: 30, top: 14, bottom: 24 };
const Y1 = { lower: 0, upper: 500, step: 100 };
const Y2 = { lower: -0.1, upper: 1.1, ticks: [0, 0.5, 1] };

// Seconds since midnight of the clock the x axis shows.
const CLOCK_START_S = 13 * 3600 + 8 * 60 + 40;

const pad = (n: number): string => String(n).padStart(2, "0");

const tickLabel = (t: number): string => {
  const total = Math.round(CLOCK_START_S + t);
  const s = total % 60;
  if (s !== 0) return `${s}s`;
  return `${pad(Math.floor(total / 3600) % 24)}:${pad(Math.floor(total / 60) % 60)}`;
};

/** Renders the Tank pressure line plot as SVG markup for a box of the given size. */
export const plot = (s: State, width: number, height: number): string => {
  const x0 = MARGIN.left;
  const x1 = width - MARGIN.right;
  const y0 = height - MARGIN.bottom;
  const y1 = MARGIN.top;
  const end = s.t;
  const start = end - WINDOW_S;
  const sx = (t: number): number => x0 + ((t - start) / WINDOW_S) * (x1 - x0);
  const sy = (v: number, lower: number, upper: number): number =>
    y0 - ((v - lower) / (upper - lower)) * (y0 - y1);
  const parts: string[] = [];
  for (let v = Y1.lower; v <= Y1.upper; v += Y1.step) {
    const y = sy(v, Y1.lower, Y1.upper).toFixed(1);
    parts.push(
      `<line class="docs-console__plot-grid" x1="${x0}" x2="${x1}" y1="${y}" y2="${y}"/>`,
      `<text class="docs-console__plot-tick" x="${x0 - 6}" y="${y}" text-anchor="end" dominant-baseline="middle">${v}</text>`,
    );
  }
  for (const v of Y2.ticks) {
    const y = sy(v, Y2.lower, Y2.upper).toFixed(1);
    parts.push(
      `<text class="docs-console__plot-tick" x="${x1 + 6}" y="${y}" dominant-baseline="middle">${v}</text>`,
    );
  }
  for (let t = Math.ceil(start / 5) * 5; t <= end; t += 5) {
    const x = sx(t).toFixed(1);
    parts.push(
      `<line class="docs-console__plot-axis" x1="${x}" x2="${x}" y1="${y0}" y2="${y0 + 4}"/>`,
      `<text class="docs-console__plot-tick" x="${x}" y="${y0 + 15}" text-anchor="middle">${tickLabel(t)}</text>`,
    );
  }
  parts.push(
    `<line class="docs-console__plot-axis" x1="${x0}" x2="${x0}" y1="${y1}" y2="${y0}"/>`,
    `<line class="docs-console__plot-axis" x1="${x1}" x2="${x1}" y1="${y1}" y2="${y0}"/>`,
    `<line class="docs-console__plot-axis" x1="${x0}" x2="${x1}" y1="${y0}" y2="${y0}"/>`,
  );
  for (const line of LINES) {
    const [lower, upper] =
      line.axis === "y1" ? [Y1.lower, Y1.upper] : [Y2.lower, Y2.upper];
    const points = s.history
      .map((p) => `${sx(p.t).toFixed(1)},${sy(p[line.key], lower, upper).toFixed(1)}`)
      .join(" ");
    parts.push(
      `<polyline class="docs-console__plot-line" stroke="${line.color}" points="${points}"/>`,
    );
  }
  return parts.join("");
};
