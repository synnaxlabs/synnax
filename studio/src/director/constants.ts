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

/**
 * Margin (CSS px) kept around a focused element's rect when the camera frames
 * it: the rect plus this margin must fit inside the zoomed viewport.
 */
export const ZOOM_RECT_MARGIN_PX = 96;

/** Ceiling for zoom amounts derived from element rects (authored or auto). */
export const RECT_ZOOM_MAX = 2.5;

/** Minimum-jerk travel duration model: T = MIN + SCALE * sqrt(d / diagonal). */
export const TRAVEL_MIN_S = 0.25;
export const TRAVEL_SCALE_S = 0.35;
export const TRAVEL_MAX_S = 1.1;

/** Isotropic blur (px per dsf unit) per unit of zoom-amount change per frame. */
export const BLUR_ZOOM_GAIN = 14;

/** Directional blur as a fraction of the crop's per-frame travel in output px. */
export const BLUR_TRAVEL_GAIN = 0.25;

/** Per-axis motion blur ceiling, in output px per dsf unit. */
export const BLUR_MAX_PX = 12;

/** Blur radii at or below this (output px per dsf unit) are dropped entirely. */
export const BLUR_MIN_PX = 0.4;

/** Seconds the cursor must be still before it starts fading out. */
export const IDLE_FADE_DELAY_S = 1.5;

/** Seconds the idle fade-out takes. */
export const IDLE_FADE_OUT_S = 0.5;

/** Seconds the wake fade-in takes when the cursor moves again. */
export const IDLE_FADE_IN_S = 0.15;

/** Raw-path movement (px/frame) below which the cursor counts as still. */
export const IDLE_EPSILON_PX = 0.1;
