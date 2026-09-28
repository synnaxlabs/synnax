// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  CALLOUT_OUT_S,
  CALLOUT_S,
  SCOPE_FADE_S,
  SCOPE_PAD,
  SCOPE_SMOOTH_S,
} from "@/director/constants";
import { type Plane, type StageSample } from "@/director/stage";
import { FPS, type Overlays } from "@/film";
import { type Mark, marks, type Point, type Rect, type Timeline } from "@/timeline";

/** The scope's data for the whole film. Values are indexed by capture tick. */
export interface ScopePlan {
  label: string;
  unit?: string;
  color: string;
  /** Raw track values, drawn as the trace. */
  trace: number[];
  /** Smoothed track values, shown as the readout so its digits do not flicker. */
  readout: number[];
  /** Value range the trace spans, bottom to top. */
  low: number;
  high: number;
}

export interface ScopeFrame {
  tick: number;
  opacity: number;
  /** Seconds since the scope's zero mark, when it has one. */
  clock?: number;
}

export interface CalloutFrame {
  text: string;
  side: "left" | "right";
  /** Rect of the callout's target, in CSS px of the capture. */
  rect: Rect;
  /** Seconds since the callout appeared. */
  age: number;
  opacity: number;
}

export interface OverlayFrame {
  scope?: ScopeFrame;
  callouts: CalloutFrame[];
}

/** OverlayPlan is everything the overlays draw, one frame per stage sample. */
export interface OverlayPlan {
  scope?: ScopePlan;
  frames: OverlayFrame[];
}

const clamp = (t: number): number => Math.min(1, Math.max(0, t));

const smoothstep = (t: number): number => {
  const c = clamp(t);
  return c * c * (3 - 2 * c);
};

/** smooth averages each value with its neighbors in a centered window of `seconds`. */
const smooth = (values: number[], seconds: number): number[] => {
  const half = Math.round((seconds * FPS) / 2);
  return values.map((_, i) => {
    const lo = Math.max(0, i - half);
    const hi = Math.min(values.length - 1, i + half);
    let sum = 0;
    for (let j = lo; j <= hi; j++) sum += values[j];
    return sum / (hi - lo + 1);
  });
};

/**
 * fades returns each frame's opacity for a layer shown on the frames where `visible`
 * holds: it ramps in over `frames` from the start of each visible run and out over
 * `frames` before the run ends.
 */
const fades = (visible: boolean[], frames: number): number[] => {
  const opacity = visible.map(() => 0);
  let start = 0;
  for (let i = 0; i <= visible.length; i++) {
    if (i < visible.length && visible[i]) {
      if (i === 0 || !visible[i - 1]) start = i;
      continue;
    }
    if (i === 0 || !visible[i - 1]) continue;
    for (let j = start; j < i; j++)
      opacity[j] = smoothstep(Math.min(j - start + 1, i - j) / frames);
  }
  return opacity;
};

/**
 * overlay plans a film's overlays against its stage samples and capture. Overlays draw
 * only over takes. Pure and deterministic. Throws when an overlay names a missing mark
 * or track, a scope spans no ticks, or a callout targets a mark without a rect.
 */
export const overlay = (
  layers: Overlays,
  samples: StageSample[],
  tl: Timeline,
): OverlayPlan => {
  const byName = marks(tl);
  const find = (owner: string, mark: string): Mark => {
    const found = byName.get(mark);
    if (found == null)
      throw new Error(`${owner} names mark "${mark}", which the capture never set`);
    return found;
  };
  const ticks = samples.map((s) => (s.type === "take" ? s.tick : null));
  const frames: OverlayFrame[] = samples.map(() => ({ callouts: [] }));

  let scope: ScopePlan | undefined;
  if (layers.scope != null) {
    const { track, label, unit, color } = layers.scope;
    const trace = tl.tracks?.[track];
    if (trace == null)
      throw new Error(`scope reads track "${track}", which the capture never recorded`);
    const from = find("scope", layers.scope.from).tick;
    const to = find("scope", layers.scope.to).tick;
    if (to <= from)
      throw new Error(
        `scope runs from "${layers.scope.from}" to "${layers.scope.to}", which is empty`,
      );
    const zero =
      layers.scope.zero != null ? find("scope", layers.scope.zero).tick : undefined;
    const shown = trace.slice(from, to);
    const lo = Math.min(...shown);
    const span = Math.max(...shown) - lo || 1;
    scope = {
      label,
      unit,
      color,
      trace,
      readout: smooth(trace, SCOPE_SMOOTH_S),
      low: lo - SCOPE_PAD * span,
      high: lo + span + SCOPE_PAD * span,
    };
    const visible = ticks.map((t) => t != null && t >= from && t < to);
    const opacity = fades(visible, SCOPE_FADE_S * FPS);
    ticks.forEach((tick, i) => {
      if (tick == null || !visible[i]) return;
      frames[i].scope = {
        tick,
        opacity: opacity[i],
        ...(zero != null && { clock: (tick - zero) / FPS }),
      };
    });
  }

  layers.callouts.forEach((callout, c) => {
    const owner = `callout ${c}`;
    const at = find(owner, callout.at).tick;
    const { rect } = find(owner, callout.target);
    if (rect == null)
      throw new Error(`${owner} targets mark "${callout.target}", which has no rect`);
    const end = at + Math.round((callout.seconds ?? CALLOUT_S) * FPS);
    ticks.forEach((tick, i) => {
      if (tick == null || tick < at || tick >= end) return;
      frames[i].callouts.push({
        text: callout.text,
        side: callout.side,
        rect,
        age: (tick - at) / FPS,
        opacity: smoothstep((end - tick) / (CALLOUT_OUT_S * FPS)),
      });
    });
  });

  return { scope, frames };
};

/**
 * project returns where a point of the capture, in CSS px, lands in the output frame
 * for a plane on a stage of the given size and perspective distance.
 */
export const project = (
  plane: Plane,
  point: Point,
  width: number,
  height: number,
  perspective: number,
): Point => {
  const x = (point.x - plane.cx) * plane.scale;
  const y = (point.y - plane.cy) * plane.scale;
  const rz = (plane.tilt.z * Math.PI) / 180;
  const rx = (plane.tilt.x * Math.PI) / 180;
  const turnedX = x * Math.cos(rz) - y * Math.sin(rz);
  const turnedY = x * Math.sin(rz) + y * Math.cos(rz);
  const depth = turnedY * Math.sin(rx);
  const f = perspective / (perspective - depth);
  return {
    x: width / 2 + turnedX * f,
    y: height / 2 + turnedY * Math.cos(rx) * f,
  };
};
