// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  AUTO_ZOOM_AMOUNT,
  RECT_ZOOM_MAX,
  RECT_ZOOM_MIN,
  ZOOM_END_MARGIN_S,
  ZOOM_IGNORE_TAIL_S,
  ZOOM_LEAD_MAX_S,
  ZOOM_MERGE_GAP_S,
  ZOOM_POST_S,
  ZOOM_PRE_S,
  ZOOM_RECT_MARGIN_PX,
} from "@/director/constants";
import { clicks, type Event, type Point, type Rect, type Timeline } from "@/timeline";

/** A focus the camera frames: a point, optionally with its element's rect. */
export interface Focus {
  tick: number;
  point: Point;
  rect?: Rect;
}

/** A planned zoom segment: the camera holds `amount` over [start, end] ticks. */
export interface Segment {
  start: number;
  end: number;
  amount: number;
  /** Focuses (CSS px) within the segment, in tick order. */
  focus: Focus[];
}

/**
 * fitAmount returns the largest zoom at which rect plus the framing margin
 * still fits inside the viewport, floored at 1 (no zoom for oversized rects).
 */
export const fitAmount = (rect: Rect, width: number, height: number): number =>
  Math.max(
    1,
    Math.min(
      width / (rect.width + 2 * ZOOM_RECT_MARGIN_PX),
      height / (rect.height + 2 * ZOOM_RECT_MARGIN_PX),
    ),
  );

interface ZoomOverride {
  amount?: number;
  rect?: Rect;
}

/**
 * overrideAmount resolves an authored override's magnification: the authored
 * amount when given, otherwise as tight as the framing margin allows for the
 * rect, capped at RECT_ZOOM_MAX.
 */
const overrideAmount = (o: ZoomOverride, width: number, height: number): number => {
  if (o.amount != null) return o.amount;
  if (o.rect == null)
    throw new Error("zoom override without an amount requires a rect");
  return Math.min(RECT_ZOOM_MAX, fitAmount(o.rect, width, height));
};

/**
 * approachStart returns the tick the cursor departed toward the click: the
 * start of the move that arrives at the click's point. Anchoring the zoom
 * there lets the camera settle before the click instead of flying through it.
 */
const approachStart = (events: Event[], c: Point & { tick: number }): number | null => {
  let departure: number | null = null;
  for (const e of events) {
    if (e.type !== "move" || e.tick > c.tick) continue;
    if (Math.abs(e.x - c.x) > 1 || Math.abs(e.y - c.y) > 1) continue;
    departure = e.tick - e.duration;
  }
  return departure;
};

/**
 * plan derives zoom segments from the timeline: one segment per click starting
 * at the cursor's approach (falling back to a fixed pre-roll), holding for the
 * post window, merged when gaps are small, clamped away from the tail.
 * Authored zoom overrides clip any auto segment they overlap.
 */
export const plan = (tl: Timeline): Segment[] => {
  const { fps, frames, width, height } = tl.meta;
  const overrides = tl.events.filter((e) => e.type === "zoom");
  const auto: Segment[] = [];

  for (const c of clicks(tl)) {
    if (c.zoom === false) continue;
    if (c.tick >= frames - ZOOM_IGNORE_TAIL_S * fps) continue;
    if (c.rect != null && fitAmount(c.rect, width, height) < RECT_ZOOM_MIN) continue;
    if (overrides.some((o) => c.tick >= o.tick && c.tick <= o.endTick)) continue;
    const preRoll = Math.round(c.tick - ZOOM_PRE_S * fps);
    const approach = approachStart(tl.events, c);
    const start = Math.min(
      preRoll,
      Math.max(approach ?? preRoll, Math.round(c.tick - ZOOM_LEAD_MAX_S * fps)),
    );
    auto.push({
      start: Math.max(0, start),
      end: Math.min(
        Math.round(frames - ZOOM_END_MARGIN_S * fps),
        Math.round(c.tick + ZOOM_POST_S * fps),
      ),
      amount: AUTO_ZOOM_AMOUNT,
      focus: [{ tick: c.tick, point: { x: c.x, y: c.y }, rect: c.rect }],
    });
  }

  const merged: Segment[] = [];
  for (const seg of auto) {
    const last = merged.at(-1);
    if (last != null && seg.start - last.end <= ZOOM_MERGE_GAP_S * fps) {
      last.end = Math.max(last.end, seg.end);
      last.amount = Math.max(last.amount, seg.amount);
      last.focus.push(...seg.focus);
    } else merged.push(seg);
  }

  const clipped = merged
    .map((seg) => {
      for (const o of overrides) {
        if (seg.end < o.tick || seg.start > o.endTick) continue;
        if (seg.start < o.tick) seg.end = Math.min(seg.end, o.tick - 1);
        else seg.start = Math.max(seg.start, o.endTick + 1);
      }
      return seg;
    })
    .filter((seg) => seg.end > seg.start);

  for (const o of overrides)
    clipped.push({
      start: o.tick,
      end: o.endTick,
      amount: overrideAmount(o, width, height),
      focus: [{ tick: o.tick, point: { x: o.x, y: o.y }, rect: o.rect }],
    });

  return clipped.sort((a, b) => a.start - b.start);
};
