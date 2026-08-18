// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/**
 * The timeline is the source-agnostic contract between producers (the Playwright
 * capture rig, or a hand-authored promo script) and the director/compositor. All
 * coordinates are CSS pixels of the captured viewport; all times are integer frame
 * ticks at `meta.fps`.
 */

export const pointZ = z.object({ x: z.number(), y: z.number() });
export type Point = z.infer<typeof pointZ>;

/** Bounding rect of an interaction's target element, in CSS px. */
export const rectZ = z.object({
  x: z.number(),
  y: z.number(),
  width: z.number().nonnegative(),
  height: z.number().nonnegative(),
});
export type Rect = z.infer<typeof rectZ>;

/** Sprite the synthetic cursor shows after arriving at a move's target. */
export const cursorKindZ = z.enum(["default", "pointer", "text"]);
export type CursorKind = z.infer<typeof cursorKindZ>;

/** Cursor travels to (x, y), arriving at `tick`, departing `tick - duration`. */
export const moveEventZ = z.object({
  type: z.literal("move"),
  tick: z.int().nonnegative(),
  x: z.number(),
  y: z.number(),
  /** Travel duration in ticks. Zero teleports the cursor. */
  duration: z.int().nonnegative(),
  /** Rect of the resolved target element, when the move targeted one. */
  rect: rectZ.optional(),
  /** Computed CSS cursor of the target, mapped to a sprite kind. */
  cursor: cursorKindZ.optional(),
});

export const pointerDownEventZ = z.object({
  type: z.literal("pointerdown"),
  tick: z.int().nonnegative(),
  x: z.number(),
  y: z.number(),
  button: z.enum(["left", "right", "middle"]).default("left"),
  /** Rect of the clicked element; the director frames zooms on it. */
  rect: rectZ.optional(),
});

export const pointerUpEventZ = z.object({
  type: z.literal("pointerup"),
  tick: z.int().nonnegative(),
  x: z.number(),
  y: z.number(),
  button: z.enum(["left", "right", "middle"]).default("left"),
});

export const keyEventZ = z.object({
  type: z.literal("key"),
  tick: z.int().nonnegative(),
  key: z.string(),
});

export const scrollEventZ = z.object({
  type: z.literal("scroll"),
  tick: z.int().nonnegative(),
  x: z.number(),
  y: z.number(),
  deltaY: z.number(),
});

/**
 * Authored camera override: forces the camera to the given focus and amount for
 * [tick, endTick], suppressing auto-zoom segments that overlap it.
 */
export const zoomOverrideEventZ = z
  .object({
    type: z.literal("zoom"),
    tick: z.int().nonnegative(),
    endTick: z.int().nonnegative(),
    /** Magnification. When omitted the director derives it from `rect`. */
    amount: z.number().min(1).optional(),
    x: z.number(),
    y: z.number(),
    /** Rect to frame; when present the camera fits it instead of centering (x, y). */
    rect: rectZ.optional(),
  })
  .refine((e) => e.amount != null || e.rect != null, {
    message: "zoom override requires an amount, a rect, or both",
  });

export const eventZ = z.discriminatedUnion("type", [
  moveEventZ,
  pointerDownEventZ,
  pointerUpEventZ,
  keyEventZ,
  scrollEventZ,
  zoomOverrideEventZ,
]);
export type Event = z.infer<typeof eventZ>;

export const metaZ = z.object({
  version: z.literal(1),
  fps: z.int().positive(),
  /** Captured viewport size in CSS pixels. */
  width: z.int().positive(),
  height: z.int().positive(),
  /** Device scale factor of the capture; frames are width*dsf x height*dsf. */
  dsf: z.number().positive(),
  theme: z.enum(["light", "dark"]),
  /** Total captured frame count. */
  frames: z.int().positive(),
});
export type Meta = z.infer<typeof metaZ>;

export const timelineZ = z.object({
  meta: metaZ,
  events: eventZ.array(),
  /** Cursor position at tick 0. */
  origin: pointZ,
});
export type Timeline = z.infer<typeof timelineZ>;

/** parse validates and returns a Timeline from unknown JSON data. */
export const parse = (data: unknown): Timeline => timelineZ.parse(data);

/** clicks returns pointerdown events in tick order. */
export const clicks = (tl: Timeline): z.infer<typeof pointerDownEventZ>[] =>
  tl.events.filter((e) => e.type === "pointerdown").sort((a, b) => a.tick - b.tick);
