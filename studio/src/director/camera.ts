// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  BLUR_MAX_PX,
  BLUR_MIN_PX,
  BLUR_TRAVEL_GAIN,
  BLUR_ZOOM_GAIN,
  CAMERA_SIM_DT,
  CAMERA_SPRING,
  EDGE_SNAP_RATIO,
  FOLLOW_DEADZONE_H,
  FOLLOW_DEADZONE_W,
  PRE_AIM_EPSILON,
  ZOOM_RECT_MARGIN_PX,
} from "@/director/constants";
import { at, at2D, step, step2D } from "@/director/spring";
import { fitAmount, type Focus, type Segment } from "@/director/zoom";
import { type Point, type Rect, type Timeline } from "@/timeline";

/**
 * One camera sample per output frame. `cx`/`cy` are travel-space coordinates in
 * [0, 1]: 0 pins the crop flush to the left/top edge, 1 flush to the right/bottom.
 * The visible crop in CSS px is derived via {@link crop}.
 */
export interface CameraSample {
  amount: number;
  cx: number;
  cy: number;
}

export type CameraTrack = CameraSample[];

export interface Crop {
  x: number;
  y: number;
  width: number;
  height: number;
}

/** crop converts a camera sample into the visible source rectangle in CSS px. */
export const crop = (s: CameraSample, width: number, height: number): Crop => ({
  x: s.cx * (width - width / s.amount),
  y: s.cy * (height - height / s.amount),
  width: width / s.amount,
  height: height / s.amount,
});

/**
 * focusToCenter maps a focus point (CSS px) to the travel-space center that puts
 * the point at the middle of the crop, with edge snapping: focus in the outer
 * band pins flush to the edge.
 */
export const focusToCenter = (
  focus: Point,
  amount: number,
  width: number,
  height: number,
): Point => {
  const map = (u: number, z: number): number => {
    if (z <= 1) return 0.5;
    const naive = (u - 1 / (2 * z)) / (1 - 1 / z);
    const snapped = (naive - EDGE_SNAP_RATIO) / (1 - 2 * EDGE_SNAP_RATIO);
    return Math.min(1, Math.max(0, snapped));
  };
  return { x: map(focus.x / width, amount), y: map(focus.y / height, amount) };
};

/** Per-axis gaussian blur radii in output pixels. */
export interface Blur {
  x: number;
  y: number;
}

/**
 * motionBlur returns the blur the compositor should apply for the camera travel
 * from prev to cur: directional with the crop's per-frame velocity, plus an
 * isotropic term while the zoom amount changes. Returns null when the travel is
 * below threshold so settled frames stay tack sharp.
 */
export const motionBlur = (
  prev: CameraSample,
  cur: CameraSample,
  width: number,
  height: number,
  dsf: number,
): Blur | null => {
  const a = crop(prev, width, height);
  const b = crop(cur, width, height);
  const scale = cur.amount * dsf;
  const zoom = Math.abs(cur.amount - prev.amount) * BLUR_ZOOM_GAIN * dsf;
  const x = Math.min(
    BLUR_MAX_PX * dsf,
    Math.abs(b.x - a.x) * scale * BLUR_TRAVEL_GAIN + zoom,
  );
  const y = Math.min(
    BLUR_MAX_PX * dsf,
    Math.abs(b.y - a.y) * scale * BLUR_TRAVEL_GAIN + zoom,
  );
  if (x <= BLUR_MIN_PX * dsf && y <= BLUR_MIN_PX * dsf) return null;
  return { x, y };
};

/**
 * frameRect maps a target rect to the travel-space center that keeps the whole
 * rect (plus margin) inside the crop at the given amount, preferring the rect's
 * center when it fits with room to spare.
 */
export const frameRect = (
  rect: Rect,
  amount: number,
  width: number,
  height: number,
): Point => {
  const axis = (pos: number, size: number, span: number): number => {
    if (amount <= 1) return 0.5;
    const cropSpan = span / amount;
    let c = pos + size / 2;
    const lo = pos + size + ZOOM_RECT_MARGIN_PX - cropSpan / 2;
    const hi = pos - ZOOM_RECT_MARGIN_PX + cropSpan / 2;
    if (lo <= hi) c = Math.min(hi, Math.max(lo, c));
    const travel = (c - cropSpan / 2) / (span - cropSpan);
    return Math.min(1, Math.max(0, travel));
  };
  return {
    x: axis(rect.x, rect.width, width),
    y: axis(rect.y, rect.height, height),
  };
};

/**
 * segmentAmount caps a segment's zoom so the active focus rect (when present)
 * fits inside the viewport with margin.
 */
const segmentAmount = (
  seg: Segment,
  focus: Focus,
  width: number,
  height: number,
): number =>
  focus.rect == null
    ? seg.amount
    : Math.min(seg.amount, fitAmount(focus.rect, width, height));

/**
 * simulate runs the spring-driven virtual camera over the planned segments and
 * samples it once per output frame. Targets are step functions; velocity carries
 * across retargets. While fully zoomed out the center pre-aims at the next focus
 * so zoom-ins scale straight toward their subject. Within a segment the center
 * retargets only when a new focus point leaves the dead-zone box.
 */
export const simulate = (tl: Timeline, segments: Segment[]): CameraTrack => {
  const { fps, frames, width, height } = tl.meta;
  let amount = at(1);
  let center = at2D(0.5, 0.5);

  let activeFocus: Focus | null = null;
  const track: CameraTrack = [];
  const stepsPerFrame = Math.max(1, Math.round(1 / fps / CAMERA_SIM_DT));
  const dt = 1 / fps / stepsPerFrame;

  const segmentAt = (tick: number): Segment | undefined =>
    segments.find((s) => tick >= s.start && tick <= s.end);
  const nextSegment = (tick: number): Segment | undefined =>
    segments.find((s) => s.start >= tick);

  for (let f = 0; f < frames; f++) {
    for (let i = 0; i < stepsPerFrame; i++) {
      const tick = f + i / stepsPerFrame;
      const seg = segmentAt(tick);

      let targetAmount = 1;
      let targetCenter: Point | null = null;

      if (seg != null) {
        const passed = seg.focus.filter((p) => p.tick <= tick);
        const focus = passed.at(-1) ?? seg.focus[0];
        if (activeFocus == null) activeFocus = focus;
        else {
          const halfW = (FOLLOW_DEADZONE_W * width) / seg.amount / 2;
          const halfH = (FOLLOW_DEADZONE_H * height) / seg.amount / 2;
          const dx = Math.abs(focus.point.x - activeFocus.point.x);
          const dy = Math.abs(focus.point.y - activeFocus.point.y);
          if (dx > halfW || dy > halfH || focus.rect !== activeFocus.rect)
            activeFocus = focus;
        }
        targetAmount = segmentAmount(seg, activeFocus, width, height);
        targetCenter =
          activeFocus.rect != null
            ? frameRect(activeFocus.rect, targetAmount, width, height)
            : focusToCenter(activeFocus.point, targetAmount, width, height);
      } else {
        activeFocus = null;
        const next = nextSegment(tick);
        if (next != null && amount.position <= PRE_AIM_EPSILON) {
          const first = next.focus[0];
          const nextAmount = segmentAmount(next, first, width, height);
          const c =
            first.rect != null
              ? frameRect(first.rect, nextAmount, width, height)
              : focusToCenter(first.point, nextAmount, width, height);
          center = at2D(c.x, c.y);
        }
      }

      amount = step(amount, targetAmount, CAMERA_SPRING, dt);
      if (targetCenter != null)
        center = step2D(center, targetCenter, CAMERA_SPRING, dt);
    }
    track.push({
      amount: Math.max(1, amount.position),
      cx: Math.min(1, Math.max(0, center.x.position)),
      cy: Math.min(1, Math.max(0, center.y.position)),
    });
  }
  return track;
};
