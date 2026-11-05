// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

/**
 * A playback clock. Every thread computes the time it shows from the wall clock, so the
 * clock changes only when playback starts, stops, seeks, or changes speed.
 */
export const clockZ = z.object({
  /** The time the clock showed at wall. */
  at: TimeStamp.z,
  /** Milliseconds since the Unix epoch. */
  wall: z.number(),
  /** Seconds of video that pass in one second of wall time. */
  speed: z.number(),
  playing: z.boolean(),
});
export interface Clock extends z.infer<typeof clockZ> {}

/** @returns a stopped clock at the time. */
export const stopped = (at: TimeStamp, wall: number = Date.now()): Clock => ({
  at,
  wall,
  speed: 1,
  playing: false,
});

/** @returns the time the clock shows at the wall time. */
export const timeAt = (c: Clock, wall: number = Date.now()): TimeStamp =>
  c.playing
    ? c.at.add(TimeSpan.nanoseconds(Math.round((wall - c.wall) * c.speed * 1e6)))
    : c.at;

/** @returns the clock, playing from the time it shows at the wall time. */
export const play = (c: Clock, wall: number = Date.now()): Clock => ({
  ...c,
  at: timeAt(c, wall),
  wall,
  playing: true,
});

/** @returns the clock, stopped at the time it shows at the wall time. */
export const pause = (c: Clock, wall: number = Date.now()): Clock => ({
  ...c,
  at: timeAt(c, wall),
  wall,
  playing: false,
});

/** @returns the clock, moved to the time. */
export const seek = (c: Clock, at: TimeStamp, wall: number = Date.now()): Clock => ({
  ...c,
  at,
  wall,
});

/** @returns the clock at the speed, from the time it shows at the wall time. */
export const setSpeed = (
  c: Clock,
  speed: number,
  wall: number = Date.now(),
): Clock => ({
  ...c,
  at: timeAt(c, wall),
  wall,
  speed,
});
