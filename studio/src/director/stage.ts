// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  CARD_IN_S,
  CARD_RISE_PX,
  CARD_S,
  END_IN_S,
  END_S,
  FORESHORTEN_MARGIN,
  HEIGHT_FILL,
  LEAN_DEG,
  LEAN_MAX_DEG,
  LEAN_S,
  LEAN_ZOOM_DEG,
  PACE_MAX_S,
  PACE_MIN_S,
  PACE_PER_WIDTH_S,
  PACE_PER_ZOOM_S,
  PERSPECTIVE_PX,
  PUSH_PER_S,
  REVEAL_S,
  STAGE_REST,
  TARGET_FILL,
  WINDOW_FILL,
} from "@/director/constants";
import { at, type SpringParams, type SpringState, step } from "@/director/spring";
import { type Edit, FPS, HEIGHT, type Shot, type Take, type Tilt, WIDTH } from "@/film";
import { type Mark, type Rect, type Timeline } from "@/timeline";

const clamp = (t: number, lo = 0, hi = 1): number => Math.min(hi, Math.max(lo, t));

/** Ease-out quint: fast arrival, long settle. */
export const easeOutQuint = (t: number): number => 1 - (1 - clamp(t)) ** 5;

/**
 * Where the capture sits on the stage for one frame. (`cx`, `cy`) is the capture
 * point, in CSS px, placed at the frame center. `scale` is output px per CSS px.
 */
export interface Plane {
  cx: number;
  cy: number;
  scale: number;
  tilt: Tilt;
}

export type StageSample =
  | { shot: number; type: "take"; tick: number; plane: Plane }
  | { shot: number; type: "card"; opacity: number; offset: number }
  | { shot: number; type: "end"; opacity: number };

/** StagePlan is everything the film compositor draws, one sample per output frame. */
export interface StagePlan {
  width: number;
  height: number;
  fps: number;
  perspective: number;
  shots: Shot[];
  samples: StageSample[];
}

const markLookup = (tl: Timeline): Map<string, Mark> => {
  const marks = new Map<string, Mark>();
  for (const e of tl.events) if (e.type === "mark") marks.set(e.name, e);
  return marks;
};

const describe = (index: number, shot: Shot): string => `shot ${index} (${shot.type})`;

/** settling returns a critically damped spring that settles in about `pace` seconds. */
const settling = (pace: number): SpringParams => {
  const omega = 5.8 / pace;
  return { stiffness: omega * omega, damping: 2 * omega, mass: 1 };
};

interface Framing {
  tick: number;
  name: string;
  cx: number;
  cy: number;
  /** Natural log of the plane scale. */
  zoom: number;
  pace?: number;
  /** Natural log of the zoom factor per second. */
  push: number;
}

const planTake = (
  index: number,
  shot: Take,
  tl: Timeline,
  marks: Map<string, Mark>,
): StageSample[] => {
  const name = describe(index, shot);
  const find = (mark: string): Mark => {
    const found = marks.get(mark);
    if (found == null)
      throw new Error(`${name} cuts at mark "${mark}", which the capture never set`);
    return found;
  };
  const from = find(shot.from);
  const to = find(shot.to);
  if (to.tick <= from.tick)
    throw new Error(`${name} runs from "${shot.from}" to "${shot.to}", which is empty`);
  const { width, height, dsf } = tl.meta;
  const framings = shot.beats.map((beat, b): Framing => {
    const { tick } = find(beat.at);
    if (b === 0 && tick !== from.tick)
      throw new Error(`${name} opens on beat "${beat.at}", not on "${shot.from}"`);
    if (tick >= to.tick)
      throw new Error(`${name} beat ${b} falls at "${beat.at}", after the take ends`);
    const prev = b > 0 ? find(shot.beats[b - 1].at).tick : tick;
    if (tick < prev)
      throw new Error(`${name} beat ${b} at "${beat.at}" comes before beat ${b - 1}`);
    let region: Rect = { x: 0, y: 0, width, height };
    let fill = beat.fill ?? WINDOW_FILL;
    if (beat.wide !== true) {
      const framed = beat.frame ?? beat.at;
      const { rect } = find(framed);
      if (rect == null)
        throw new Error(`${name} beat ${b} frames mark "${framed}", which has no rect`);
      region = rect;
      fill = beat.fill ?? TARGET_FILL;
    }
    const scale = Math.min(
      (fill * WIDTH) / region.width,
      (HEIGHT_FILL * HEIGHT) / region.height,
    );
    return {
      tick,
      name: beat.at,
      cx: region.x + region.width / 2,
      cy: region.y + region.height / 2,
      zoom: Math.log(scale),
      pace: beat.pace,
      push: Math.log(1 + (beat.push ?? PUSH_PER_S)),
    };
  });

  const [first] = framings;
  let x: SpringState = at(first.cx);
  let y: SpringState = at(first.cy);
  let zoom: SpringState = at(first.zoom);
  const rest = STAGE_REST;
  const open = shot.tilt ?? rest;
  let tiltX: SpringState = at(open.x);
  let tiltZ: SpringState = at(open.z);
  let beat = 0;
  let params = settling(PACE_MIN_S);
  const dt = 1 / FPS;
  const samples: StageSample[] = [];
  for (let tick = from.tick; tick < to.tick; tick++) {
    while (beat + 1 < framings.length && framings[beat + 1].tick <= tick) {
      beat++;
      const target = framings[beat];
      const scale = Math.exp(target.zoom);
      const travel = Math.hypot(target.cx - x.position, target.cy - y.position);
      const doublings = Math.abs(target.zoom - zoom.position) / Math.LN2;
      const pace =
        target.pace ??
        clamp(
          PACE_MIN_S +
            (PACE_PER_WIDTH_S * travel * scale) / WIDTH +
            PACE_PER_ZOOM_S * doublings,
          PACE_MIN_S,
          PACE_MAX_S,
        );
      params = settling(pace);
    }
    const target = framings[beat];
    const scale = Math.exp(zoom.position);
    const peak = scale * FORESHORTEN_MARGIN;
    if (peak > dsf)
      throw new Error(
        `${name} magnifies the capture ${peak.toFixed(2)}x at beat "${target.name}", ` +
          `past its ${dsf}x detail; capture at dsf ${Math.ceil(peak)} or lower the fill`,
      );
    samples.push({
      shot: index,
      type: "take",
      tick,
      plane: {
        cx: x.position,
        cy: y.position,
        scale,
        tilt: { x: tiltX.position, z: tiltZ.position },
      },
    });
    // The plane leans into the camera's travel, the way a camera on a jib banks.
    const pan = (x.velocity * scale) / WIDTH;
    const tiltPan = (y.velocity * scale) / WIDTH;
    const dolly = zoom.velocity / Math.LN2;
    const leanZ = clamp(-LEAN_DEG * pan, -LEAN_MAX_DEG, LEAN_MAX_DEG);
    const leanX = clamp(
      LEAN_ZOOM_DEG * dolly - LEAN_DEG * tiltPan,
      -LEAN_MAX_DEG,
      LEAN_MAX_DEG,
    );
    const revealing = shot.tilt != null && (tick - from.tick) / FPS < REVEAL_S;
    const tiltParams = settling(revealing ? REVEAL_S : LEAN_S);
    tiltX = step(tiltX, rest.x + leanX, tiltParams, dt);
    tiltZ = step(tiltZ, rest.z + leanZ, tiltParams, dt);
    const held = (tick + 1 - target.tick) / FPS;
    x = step(x, target.cx, params, dt);
    y = step(y, target.cy, params, dt);
    zoom = step(zoom, target.zoom + target.push * held, params, dt);
  }
  return samples;
};

const planCard = (index: number, seconds: number): StageSample[] => {
  const samples: StageSample[] = [];
  const frames = Math.round(seconds * FPS);
  for (let f = 0; f < frames; f++) {
    const inT = easeOutQuint(f / FPS / CARD_IN_S);
    samples.push({
      shot: index,
      type: "card",
      opacity: inT,
      offset: CARD_RISE_PX * (1 - inT),
    });
  }
  return samples;
};

const planEnd = (index: number, seconds: number): StageSample[] => {
  const samples: StageSample[] = [];
  const frames = Math.round(seconds * FPS);
  for (let f = 0; f < frames; f++)
    samples.push({
      shot: index,
      type: "end",
      opacity: easeOutQuint(f / FPS / END_IN_S),
    });
  return samples;
};

/**
 * stage plans a film: it resolves each shot of the edit against the capture's marks
 * and returns one sample per output frame, shots joined by hard cuts. Pure and
 * deterministic. Throws when a shot names a missing mark, spans no frames, has beats
 * out of order or outside it, frames a mark without a rect, or magnifies the capture
 * past its detail.
 */
export const stage = (edit: Edit, tl: Timeline): StagePlan => {
  if (tl.meta.fps !== FPS)
    throw new Error(
      `films capture at ${FPS} fps, but this capture ran at ${tl.meta.fps}`,
    );
  const marks = markLookup(tl);
  const samples = edit.flatMap((shot, i): StageSample[] => {
    switch (shot.type) {
      case "take":
        return planTake(i, shot, tl, marks);
      case "card":
        return planCard(i, shot.seconds ?? CARD_S);
      case "end":
        return planEnd(i, shot.seconds ?? END_S);
    }
  });
  return {
    width: WIDTH,
    height: HEIGHT,
    fps: FPS,
    perspective: PERSPECTIVE_PX,
    shots: edit,
    samples,
  };
};
