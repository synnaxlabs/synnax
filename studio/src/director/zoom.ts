// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ZOOM_MAX, ZOOM_RECT_MARGIN_PX } from "@/director/constants";
import { type Point, type Rect, type Timeline } from "@/timeline";

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
  /** The authored framing the segment opens on. */
  focus: Focus;
  /**
   * Cursor targets within the segment, in tick order, each at its move's departure.
   * The camera pans to keep each one in frame.
   */
  follow: Focus[];
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
 * rect. Either is capped at ZOOM_MAX.
 */
const overrideAmount = (o: ZoomOverride, width: number, height: number): number => {
  if (o.amount != null) return Math.min(ZOOM_MAX, o.amount);
  if (o.rect == null)
    throw new Error("zoom override without an amount requires a rect");
  return Math.min(ZOOM_MAX, fitAmount(o.rect, width, height));
};

/**
 * plan returns the camera's zoom segments: one per authored zoom override. Clicks
 * never zoom; the camera holds the full frame unless a shot asks for a zoom.
 */
export const plan = (tl: Timeline): Segment[] => {
  const { width, height } = tl.meta;
  const moves = tl.events
    .filter((e) => e.type === "move")
    .sort((a, b) => a.tick - b.tick);
  return tl.events
    .filter((e) => e.type === "zoom")
    .map((o) => ({
      start: o.tick,
      end: o.endTick,
      amount: overrideAmount(o, width, height),
      focus: { tick: o.tick, point: { x: o.x, y: o.y }, rect: o.rect },
      follow: moves
        .filter((m) => m.tick > o.tick && m.tick - m.duration <= o.endTick)
        .map((m) => ({
          tick: m.tick - m.duration,
          point: { x: m.x, y: m.y },
          rect: m.rect,
        })),
    }))
    .sort((a, b) => a.start - b.start);
};
