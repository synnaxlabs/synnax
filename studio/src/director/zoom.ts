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
import { clicks, type Point, type Timeline } from "@/timeline";

/** A planned zoom segment: the camera holds `amount` over [start, end] ticks. */
export interface Segment {
  start: number;
  end: number;
  amount: number;
  /** Focus points (CSS px) within the segment, in tick order. */
  focus: { tick: number; point: Point }[];
}

/**
 * plan derives zoom segments from the timeline: one segment per click expanded by
 * the pre/post windows, merged when gaps are small, clamped away from the tail.
 * Authored zoom overrides replace any auto segment they overlap.
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
      focus: [{ tick: c.tick, point: { x: c.x, y: c.y } }],
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

  for (const o of overrides)
    merged.push({
      start: o.tick,
      end: o.endTick,
      amount: o.amount,
      focus: [{ tick: o.tick, point: { x: o.x, y: o.y } }],
    });

  return merged.sort((a, b) => a.start - b.start);
};
