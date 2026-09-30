// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type SpringParams } from "@/director/spring";
import { type Format, type Tilt } from "@/film";

/**
 * Tuning constants for the Screen Studio look. Sources: Screen Studio's landing
 * bundle (its literal in-app spring presets), the reverse-engineered Screen Studio
 * project format, and Cap's clean-room implementation of the same behavior. See
 * docs/tech/video-automation/decisions.md for provenance.
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
export const CAMERA_SPRING: SpringParams = { stiffness: 130, damping: 42, mass: 3 };

/** Camera simulation step in seconds. */
export const CAMERA_SIM_DT = 0.008;

/** Default auto-zoom magnification. */
export const AUTO_ZOOM_AMOUNT = 2.0;

/** Seconds of zoom lead-in before the click that triggered the segment. */
export const ZOOM_PRE_S = 0.3;

/**
 * Ceiling on how far before its click a zoom may start when anchored to the
 * cursor's approach, so a long hover does not hold the zoom absurdly early.
 */
export const ZOOM_LEAD_MAX_S = 1.5;

/** Seconds the zoom holds after the triggering click. */
export const ZOOM_POST_S = 1.2;

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

/**
 * Floor for rect-derived zoom during auto segments. A click whose target rect
 * fits only below this creates no zoom: near-full-width targets are mostly
 * empty space, so punching into them frames nothing.
 */
export const RECT_ZOOM_MIN = 1.4;

/** Seconds a zoomed click waits after its zoom starts, so the camera lands first. */
export const CLICK_ZOOM_SETTLE_S = 0.9;

/** Seconds the cursor rests on a zoomed click's target before it presses. */
export const CLICK_DWELL_S = 0.35;

/** Seconds an authored zoom holds past its last click, so the click reads. */
export const CLICK_HOLD_S = 0.6;

/** Minimum-jerk travel duration model: T = MIN + SCALE * sqrt(d / diagonal). */
export const TRAVEL_MIN_S = 0.25;
export const TRAVEL_SCALE_S = 0.35;
export const TRAVEL_MAX_S = 1.1;

/** Isotropic blur (px per dsf unit) per unit of zoom-amount change per frame. */
export const BLUR_ZOOM_GAIN = 4;

/** Directional blur as a fraction of the crop's per-frame travel in output px. */
export const BLUR_TRAVEL_GAIN = 0.08;

/** Per-axis motion blur ceiling, in output px per dsf unit. */
export const BLUR_MAX_PX = 3;

/** Blur radii at or below this (output px per dsf unit) are dropped entirely. */
export const BLUR_MIN_PX = 0.4;

/** Seconds the cursor must be still before it starts fading out. */
export const IDLE_FADE_DELAY_S = 0.75;

/** Seconds the idle fade-out takes. */
export const IDLE_FADE_OUT_S = 0.25;

/** Seconds the wake fade-in takes when the cursor moves again. */
export const IDLE_FADE_IN_S = 0.15;

/** Raw-path movement (px/frame) below which the cursor counts as still. */
export const IDLE_EPSILON_PX = 0.1;

/** Film stage: the tilt a reveal opens at. */
export const STAGE_TILT: Tilt = { x: 22, z: -10 };

/** Film stage: the reading tilt the plane settles to between moves. */
export const STAGE_REST: Tilt = { x: 4, z: -2 };

/** Seconds a reveal's tilt takes to settle to rest. */
export const REVEAL_S = 1.8;

/** Seconds the plane's lean takes to follow the camera's motion. */
export const LEAN_S = 0.7;

/** Degrees the plane leans per frame width per second of camera travel. */
export const LEAN_DEG = 4;

/** Degrees the plane leans per doubling of zoom per second. */
export const LEAN_ZOOM_DEG = 3;

/** Largest lean, in degrees, on either axis. */
export const LEAN_MAX_DEG = 6;

/** Bounds on a beat's default pace, the seconds its move takes to settle. */
export const PACE_MIN_S = 0.8;
export const PACE_MAX_S = 1.9;

/** Seconds of pace a beat's move adds per frame width traveled and per zoom doubling. */
export const PACE_PER_WIDTH_S = 0.35;
export const PACE_PER_ZOOM_S = 0.4;

/** Default share the camera zooms in per second while it holds on a beat. */
export const PUSH_PER_S = 0.01;

/**
 * Share of the frame width the whole window takes on a wide beat. A portrait frame crops
 * the window's sides; a landscape frame holds all of it with a margin.
 */
export const WINDOW_FILL: Record<Format, number> = { portrait: 1.3, landscape: 0.72 };

/** Share of the frame width a framed rect takes. */
export const TARGET_FILL = 0.72;

/** Ceiling on the share of the frame height a framed region takes. */
export const HEIGHT_FILL = 0.8;

/**
 * Headroom for the tilt's perspective, which magnifies the near edge of the plane
 * past its base scale.
 */
export const FORESHORTEN_MARGIN = 1.15;

/** CSS perspective distance of the stage, in output px. */
export const PERSPECTIVE_PX = 1800;

/** Default seconds a text card holds. */
export const CARD_S = 2.2;

/** Seconds a card's text takes to fade and rise in. */
export const CARD_IN_S = 0.55;

/** Distance in output px a card's text rises while it fades in. */
export const CARD_RISE_PX = 10;

/** Default seconds the end card holds. */
export const END_S = 3.5;

/** Seconds the end card's wordmark takes to fade in. */
export const END_IN_S = 0.6;

/** Seconds of history a scope's trace spans. */
export const SCOPE_WINDOW_S = 10;

/** Seconds a scope takes to fade in or out. */
export const SCOPE_FADE_S = 0.4;

/**
 * Seconds of the centered window that smooths a scope's readout. Centered, so the
 * readout never lags the Console.
 */
export const SCOPE_SMOOTH_S = 0.3;

/** Share of a track's range a scope leaves free above and below the trace. */
export const SCOPE_PAD = 0.12;

/** Default seconds a callout holds. */
export const CALLOUT_S = 2.8;

/** Seconds a callout takes to fade out. */
export const CALLOUT_OUT_S = 0.35;
