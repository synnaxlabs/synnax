// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CameraTrack, simulate } from "@/director/camera";
import { type CursorTrack, synthesize } from "@/director/cursor";
import { plan, type Segment } from "@/director/zoom";
import { type Timeline } from "@/timeline";

/** Tracks is the director's output: everything the compositor needs per frame. */
export interface Tracks {
  camera: CameraTrack;
  cursor: CursorTrack;
  segments: Segment[];
}

/**
 * direct turns a timeline into per-frame camera and cursor tracks. Pure and
 * deterministic: the same timeline always yields the same tracks.
 */
export const direct = (tl: Timeline): Tracks => {
  const segments = plan(tl);
  return { camera: simulate(tl, segments), cursor: synthesize(tl), segments };
};
