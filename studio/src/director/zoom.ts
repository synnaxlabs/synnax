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
  ZOOM_END_MARGIN_S,
  ZOOM_IGNORE_TAIL_S,
  ZOOM_MERGE_GAP_S,
  ZOOM_POST_S,
  ZOOM_PRE_S,
} from "@/director/constants";
import { clicks, type Point, type Rect, type Timeline } from "@/timeline";

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
 * plan derives zoom segments from the timeline: one segment per click expanded by
 * the pre/post windows, merged when gaps are small, clamped away from the tail.
 * Authored zoom overrides clip any auto segment they overlap.
 */
export const plan = (tl: Timeline): Segment[] => {
  const { fps, frames } = tl.meta;
  const overrides = tl.events.filter((e) => e.type === "zoom");
  const auto: Segment[] = [];

  for (const c of clicks(tl)) {
    if (c.tick >= frames - ZOOM_IGNORE_TAIL_S * fps) continue;
    if (overrides.some((o) => c.tick >= o.tick && c.tick <= o.endTick)) continue;
    auto.push({
      start: Math.max(0, Math.round(c.tick - ZOOM_PRE_S * fps)),
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
      amount: o.amount,
      focus: [{ tick: o.tick, point: { x: o.x, y: o.y }, rect: o.rect }],
    });

  return clipped.sort((a, b) => a.start - b.start);
};
