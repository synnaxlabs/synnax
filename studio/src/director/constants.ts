// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SpringParams } from "@/director/spring";

/**
 * Tuning constants for the Screen Studio look. Sources: Screen Studio's landing
 * bundle (its literal in-app spring presets), the reverse-engineered Screen Studio
 * project format, and Cap's clean-room implementation of the same behavior. See
 * docs/tech/video-automation/research/screen-studio-anatomy.md for provenance.
 */

/** Default spring the synthetic cursor uses to chase the raw path. */
export const CURSOR_SPRING: SpringParams = { stiffness: 470, damping: 70, mass: 3 };

/** Stiffer spring engaged just before a click so the cursor arrives crisply. */
export const CLICK_SPRING: SpringParams = { stiffness: 530, damping: 40, mass: 1 };

/** Seconds before a click at which the spring retargets to the click position. */
export const CLICK_ANTICIPATION_S = 0.5;

/** Seconds before a click at which the stiffer click spring takes over. */
export const CLICK_STIFFEN_S = 0.175;

/** Cursor scale while the primary button is held. */
export const CLICK_SHRINK_SCALE = 0.8;

/** Seconds over which the click shrink animates (each direction). */
export const CLICK_SHRINK_S = 0.13;

/** Camera spring for both zoom amount and framing center. */
export const CAMERA_SPRING: SpringParams = { stiffness: 200, damping: 40, mass: 2.25 };

/** Camera simulation step in seconds. */
export const CAMERA_SIM_DT = 0.008;

/** Default auto-zoom magnification. */
export const AUTO_ZOOM_AMOUNT = 2.0;

/** Seconds of zoom lead-in before the click that triggered the segment. */
export const ZOOM_PRE_S = 0.3;

/** Seconds the zoom holds after the triggering click. */
export const ZOOM_POST_S = 2.5;

/** Segments closer than this (seconds) merge into one. */
export const ZOOM_MERGE_GAP_S = 2.5;

/** Clicks within this many seconds of the end of the video do not create zooms. */
export const ZOOM_IGNORE_TAIL_S = 1.0;

/** Segment ends clamp to at least this many seconds before the end of the video. */
export const ZOOM_END_MARGIN_S = 0.8;

/** Focus in the outer band of the frame pins the camera flush to that edge. */
export const EDGE_SNAP_RATIO = 0.25;

/**
 * Dead-zone box for camera follow while zoomed, as a fraction of the visible
 * viewport: the camera holds until the focus leaves this box.
 */
export const FOLLOW_DEADZONE_W = 0.5;
export const FOLLOW_DEADZONE_H = 0.7;

/** Amounts at or below this are treated as fully zoomed out (pre-aim active). */
export const PRE_AIM_EPSILON = 1.0005;

/** Minimum-jerk travel duration model: T = MIN + SCALE * sqrt(d / diagonal). */
export const TRAVEL_MIN_S = 0.25;
export const TRAVEL_SCALE_S = 0.35;
export const TRAVEL_MAX_S = 1.1;
