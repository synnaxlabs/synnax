// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  CLICK_ANTICIPATION_S,
  CLICK_SHRINK_S,
  CLICK_SHRINK_SCALE,
  CLICK_SPRING,
  CLICK_STIFFEN_S,
  CURSOR_SPRING,
  IDLE_EPSILON_PX,
  IDLE_FADE_DELAY_S,
  IDLE_FADE_IN_S,
  IDLE_FADE_OUT_S,
} from "@/director/constants";
import { at2D, settleLag, step2D } from "@/director/spring";
import { clicks, type CursorKind, type Point, type Timeline } from "@/timeline";

/** One cursor sample per output frame. */
export interface CursorSample {
  x: number;
  y: number;
  /** Velocity in px/s, for motion smear. */
  vx: number;
  vy: number;
  /** Sprite scale, animating to CLICK_SHRINK_SCALE while pressed. */
  scale: number;
  pressed: boolean;
  /** Sprite kind: the last arrived move's target cursor. */
  kind: CursorKind;
  /** Sprite opacity in [0, 1]: fades out after the cursor sits still. */
  opacity: number;
}

export type CursorTrack = CursorSample[];

/**
 * minimumJerk evaluates the classic minimum-jerk position profile at normalized
 * time tau in [0, 1].
 */
export const minimumJerk = (tau: number): number => {
  const t = Math.min(1, Math.max(0, tau));
  return 10 * t ** 3 - 15 * t ** 4 + 6 * t ** 5;
};

interface RawSampleArgs {
  tl: Timeline;
  tick: number;
}

/**
 * rawPosition returns the un-smoothed cursor position at a tick: piecewise
 * minimum-jerk travel between move waypoints, holds elsewhere.
 */
export const rawPosition = ({ tl, tick }: RawSampleArgs): Point => {
  let prev: Point = tl.origin;
  for (const e of tl.events) {
    if (e.type !== "move") continue;
    const start = e.tick - e.duration;
    if (tick < start) return prev;
    if (tick < e.tick) {
      const tau = e.duration === 0 ? 1 : (tick - start) / e.duration;
      const s = minimumJerk(tau);
      return { x: prev.x + (e.x - prev.x) * s, y: prev.y + (e.y - prev.y) * s };
    }
    prev = { x: e.x, y: e.y };
  }
  return prev;
};

/**
 * synthesize produces the smoothed per-frame cursor track: minimum-jerk raw path
 * chased by the Screen Studio spring, with click anticipation, the stiffer click
 * spring near clicks, phase-lead compensation, and the press shrink animation.
 */
export const synthesize = (tl: Timeline): CursorTrack => {
  const { fps, frames } = tl.meta;
  const dt = 1 / fps;
  const downs = clicks(tl);
  const ups = tl.events.filter((e) => e.type === "pointerup");

  let state = at2D(tl.origin.x, tl.origin.y);
  let scale = 1;
  let opacity = 1;
  let idleS = 0;
  const moves = tl.events.filter((e) => e.type === "move");
  const track: CursorTrack = [];

  for (let f = 0; f < frames; f++) {
    const tSec = f / fps;
    const nextDown = downs.find((d) => d.tick >= f);
    const untilClickS = nextDown == null ? Infinity : nextDown.tick / fps - tSec;

    const params = untilClickS <= CLICK_STIFFEN_S ? CLICK_SPRING : CURSOR_SPRING;
    const lead = settleLag(params);

    let target: Point;
    if (nextDown != null && untilClickS <= CLICK_ANTICIPATION_S)
      target = { x: nextDown.x, y: nextDown.y };
    else target = rawPosition({ tl, tick: f + lead * fps });

    const prev = state;
    state = step2D(state, target, params, dt);

    const lastDown = [...downs].reverse().find((d) => d.tick <= f);
    const lastUp = [...ups].reverse().find((u) => u.tick <= f);
    const pressed = lastDown != null && (lastUp == null || lastUp.tick < lastDown.tick);

    const shrinkPerFrame = (1 - CLICK_SHRINK_SCALE) / (CLICK_SHRINK_S * fps);
    if (pressed) scale = Math.max(CLICK_SHRINK_SCALE, scale - shrinkPerFrame);
    else scale = Math.min(1, scale + shrinkPerFrame);

    const movedPx = Math.hypot(
      state.x.position - prev.x.position,
      state.y.position - prev.y.position,
    );
    if (movedPx > IDLE_EPSILON_PX || pressed) {
      idleS = 0;
      opacity = Math.min(1, opacity + dt / IDLE_FADE_IN_S);
    } else {
      idleS += dt;
      if (idleS >= IDLE_FADE_DELAY_S)
        opacity = Math.max(0, opacity - dt / IDLE_FADE_OUT_S);
    }

    const arrived = [...moves].reverse().find((m) => m.tick <= f);
    track.push({
      x: state.x.position,
      y: state.y.position,
      vx: (state.x.position - prev.x.position) * fps,
      vy: (state.y.position - prev.y.position) * fps,
      scale,
      pressed,
      kind: arrived?.cursor ?? "default",
      opacity,
    });
  }
  return track;
};
