// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/** Output width of every film in px; films are 4:5. */
export const WIDTH = 1080;
/** Output height of every film in px. */
export const HEIGHT = 1350;
/** Output and capture frame rate. */
export const FPS = 60;
/** Capture device scale factor, which sets how far a shot can magnify and stay sharp. */
export const DSF = 3;

/** Plane rotation in degrees: X tips the top away, Z turns the plane clockwise. */
export const tiltZ = z.object({ x: z.number(), z: z.number() });
export type Tilt = z.infer<typeof tiltZ>;

/**
 * One beat of a take. From the tick of mark `at`, the camera moves to frame the rect of
 * mark `frame` (default `at`), or the whole window when `wide`. `fill` is the share of
 * the frame width the framed region takes. `pace` is the seconds the move takes to
 * settle; by default it grows with the size of the move. `push` is the share the camera
 * zooms in per second while it holds on the beat.
 */
export const beatZ = z.object({
  at: z.string().min(1),
  frame: z.string().min(1).optional(),
  wide: z.boolean().optional(),
  fill: z.number().positive().optional(),
  pace: z.number().positive().optional(),
  push: z.number().optional(),
});
export type Beat = z.infer<typeof beatZ>;

/**
 * One continuous camera take on the real window, cut from the capture between two
 * marks. The take opens framing its first beat, which must fall on `from`, and moves
 * at each later beat. With `tilt`, the plane opens at that tilt and settles to rest, a
 * reveal.
 */
export const takeZ = z.object({
  type: z.literal("take"),
  from: z.string().min(1),
  to: z.string().min(1),
  beats: beatZ.array().min(1),
  tilt: tiltZ.optional(),
});
export type Take = z.infer<typeof takeZ>;

/** One or two lines of text on the stage. */
export const cardZ = z.object({
  type: z.literal("card"),
  lines: z.string().min(1).array().min(1).max(2),
  seconds: z.number().positive().optional(),
});
export type Card = z.infer<typeof cardZ>;

/** The closing card: the Synnax wordmark and a tagline on the brand gradient. */
export const endZ = z.object({
  type: z.literal("end"),
  tagline: z.string().min(1),
  seconds: z.number().positive().optional(),
});
export type End = z.infer<typeof endZ>;

export const shotZ = z.discriminatedUnion("type", [takeZ, cardZ, endZ]);
export type Shot = z.infer<typeof shotZ>;

export const editZ = shotZ.array().min(1);
export type Edit = z.infer<typeof editZ>;

/** edit validates a film's shot list, throwing on the first invalid shot. */
export const edit = (shots: unknown): Edit => editZ.parse(shots);

/**
 * A live telemetry strip along the bottom of the frame: a channel's name, its value,
 * and a trace of its last seconds, drawn from the capture's `track`. It shows during
 * takes from mark `from` to mark `to`, and carries across cuts. With `zero`, it shows a
 * T+ clock counted from that mark.
 */
export const scopeZ = z.object({
  track: z.string().min(1),
  label: z.string().min(1),
  unit: z.string().min(1).optional(),
  color: z.string().regex(/^#[0-9a-fA-F]{6}$/),
  from: z.string().min(1),
  to: z.string().min(1),
  zero: z.string().min(1).optional(),
});
export type Scope = z.infer<typeof scopeZ>;

/**
 * A label pinned beside the rect of mark `target`, on the window itself. It appears at
 * mark `at` and holds for `seconds`. `side` is the side of the rect it sits on.
 */
export const calloutZ = z.object({
  at: z.string().min(1),
  target: z.string().min(1),
  text: z.string().min(1),
  side: z.enum(["left", "right"]).default("right"),
  seconds: z.number().positive().optional(),
});
export type Callout = z.infer<typeof calloutZ>;

/** The layers a film draws over its shots. */
export const overlaysZ = z.object({
  scope: scopeZ.optional(),
  callouts: calloutZ.array().default([]),
});
export type Overlays = z.infer<typeof overlaysZ>;

/** overlays validates a film's overlays, throwing on the first invalid one. */
export const overlays = (layers: unknown): Overlays => overlaysZ.parse(layers);
