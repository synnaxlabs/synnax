// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CSSProperties, type ReactElement, useState } from "react";
import { AbsoluteFill } from "remotion";

import {
  type CalloutFrame,
  easeOutQuint,
  SCOPE_WINDOW_S,
  type ScopeFrame,
  type ScopePlan,
} from "@/director";
import { type Format, FPS } from "@/film";

export const MONO = '"Geist Mono", monospace';
export const SANS = '"Inter Variable", sans-serif';
const TEXT_KEY = "#F7F8F8";
const TEXT_REST = "#8A8F98";

/** Glass behind a caption or a scope plate. */
const GLASS: CSSProperties = {
  background: "linear-gradient(180deg, rgba(34, 36, 43, 0.72), rgba(14, 15, 19, 0.84))",
  backdropFilter: "blur(20px) saturate(1.3)",
  border: "1px solid rgba(255, 255, 255, 0.12)",
  boxShadow: "0 24px 60px rgba(0, 0, 0, 0.5), inset 0 1px 0 rgba(255, 255, 255, 0.09)",
};

/** Placement and size of the scope in output px. */
interface ScopeLayout {
  /**
   * What sets the scope off the window: a scrim fading up from the frame's bottom, or a
   * glass plate around the scope itself.
   */
  backing: { scrim: CSSProperties } | { plate: CSSProperties };
  left: number;
  right?: number;
  bottom: number;
  readout: number;
  readoutWidth: number;
  trace: { width: number; height: number };
}

/**
 * A portrait scope spans the frame's bottom. A landscape scope sits in the bottom left,
 * leaving the rest of the wide frame to the window.
 */
const SCOPE_LAYOUT: Record<Format, ScopeLayout> = {
  portrait: {
    backing: {
      scrim: {
        width: "100%",
        height: 540,
        background:
          "linear-gradient(to top, rgba(6, 6, 7, 0.97) 0%, rgba(6, 6, 7, 0.9) 42%, rgba(6, 6, 7, 0.6) 68%, transparent 100%)",
      },
    },
    left: 64,
    right: 64,
    bottom: 72,
    readout: 108,
    readoutWidth: 356,
    trace: { width: 560, height: 118 },
  },
  landscape: {
    backing: {
      plate: { ...GLASS, padding: "22px 30px 26px", borderRadius: 20 },
    },
    left: 48,
    bottom: 48,
    readout: 80,
    readoutWidth: 270,
    trace: { width: 380, height: 84 },
  },
};
/** Ticks between trace points. */
const TRACE_STEP = 2;

/** clock formats seconds from a zero mark the way a test countdown reads. */
const clock = (seconds: number): string => {
  const abs = Math.abs(seconds);
  const minutes = String(Math.floor(abs / 60)).padStart(2, "0");
  const rest = (abs % 60).toFixed(1).padStart(4, "0");
  return `T${seconds < 0 ? "−" : "+"}${minutes}:${rest}`;
};

interface TraceProps {
  plan: ScopePlan;
  tick: number;
  width: number;
  height: number;
}

/** Trace draws the scope's last seconds of values, oldest at the left. */
const Trace = ({ plan, tick, width, height }: TraceProps): ReactElement => {
  const span = SCOPE_WINDOW_S * FPS;
  const first = tick - span + 1;
  const x = (t: number): number => ((t - first) / (span - 1)) * width;
  const y = (v: number): number =>
    height - ((v - plan.low) / (plan.high - plan.low)) * height;
  const points: string[] = [];
  for (let t = Math.max(0, first); t <= tick; t += TRACE_STEP)
    points.push(`${x(t).toFixed(1)},${y(plan.trace[t]).toFixed(1)}`);
  points.push(`${x(tick).toFixed(1)},${y(plan.trace[tick]).toFixed(1)}`);
  const line = `M${points.join("L")}`;
  const start = x(Math.max(0, first)).toFixed(1);
  const area = `${line}L${width},${height}L${start},${height}Z`;
  const headY = y(plan.trace[tick]);
  return (
    <svg
      width={width}
      height={height}
      style={{ overflow: "visible", display: "block" }}
    >
      <defs>
        <linearGradient id="scope-fade" x1="0" x2="1">
          <stop offset="0" stopColor="#fff" stopOpacity="0" />
          <stop offset="0.35" stopColor="#fff" stopOpacity="0.55" />
          <stop offset="1" stopColor="#fff" stopOpacity="1" />
        </linearGradient>
        <mask id="scope-mask" maskUnits="userSpaceOnUse">
          <rect
            x="0"
            y="-20"
            width={width + 40}
            height={height + 40}
            fill="url(#scope-fade)"
          />
        </mask>
        <linearGradient id="scope-area" x1="0" x2="0" y1="0" y2="1">
          <stop offset="0" stopColor={plan.color} stopOpacity="0.28" />
          <stop offset="1" stopColor={plan.color} stopOpacity="0" />
        </linearGradient>
        <filter id="scope-glow" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="5" />
        </filter>
      </defs>
      {[0, 0.5, 1].map((f) => (
        <line
          key={f}
          x1={0}
          x2={width}
          y1={f * height}
          y2={f * height}
          stroke="rgba(255, 255, 255, 0.07)"
          strokeWidth={1}
        />
      ))}
      <g mask="url(#scope-mask)">
        <path d={area} fill="url(#scope-area)" />
        <path
          d={line}
          fill="none"
          stroke={plan.color}
          strokeWidth={6}
          strokeOpacity={0.35}
          filter="url(#scope-glow)"
        />
        <path
          d={line}
          fill="none"
          stroke={plan.color}
          strokeWidth={2.5}
          strokeLinejoin="round"
          strokeLinecap="round"
        />
      </g>
      <circle
        cx={width}
        cy={headY}
        r={16}
        fill={plan.color}
        opacity={0.45}
        filter="url(#scope-glow)"
      />
      <circle cx={width} cy={headY} r={5.5} fill="#fff" />
    </svg>
  );
};

interface ScopeViewProps {
  plan: ScopePlan;
  frame: ScopeFrame;
  format: Format;
}

/**
 * ScopeView draws the live telemetry strip along the bottom of the frame: the channel,
 * its readout, and a trace of its last seconds.
 */
export const ScopeView = ({ plan, frame, format }: ScopeViewProps): ReactElement => {
  const { opacity, tick } = frame;
  const layout = SCOPE_LAYOUT[format];
  const label = {
    fontFamily: MONO,
    fontSize: 21,
    fontWeight: 500,
    letterSpacing: "0.14em",
    color: TEXT_REST,
    fontVariantNumeric: "tabular-nums",
  } as const;
  return (
    <AbsoluteFill style={{ opacity }}>
      {"scrim" in layout.backing && (
        <div
          style={{ position: "absolute", left: 0, bottom: 0, ...layout.backing.scrim }}
        />
      )}
      <div
        style={{
          position: "absolute",
          left: layout.left,
          right: layout.right,
          bottom: layout.bottom,
          transform: `translateY(${(1 - opacity) * 10}px)`,
          ...("plate" in layout.backing && layout.backing.plate),
        }}
      >
        <div style={{ display: "flex", alignItems: "center", ...label }}>
          <div
            style={{
              width: 10,
              height: 10,
              borderRadius: 5,
              marginRight: 14,
              background: plan.color,
              boxShadow: `0 0 12px ${plan.color}`,
            }}
          />
          {plan.label}
          <div style={{ flex: 1 }} />
          {frame.clock != null && clock(frame.clock)}
        </div>
        <div style={{ display: "flex", alignItems: "flex-end", marginTop: 18 }}>
          <div
            style={{
              width: layout.readoutWidth,
              display: "flex",
              alignItems: "baseline",
              whiteSpace: "nowrap",
            }}
          >
            <span
              style={{
                fontFamily: SANS,
                fontSize: layout.readout,
                fontWeight: 560,
                letterSpacing: "-0.04em",
                lineHeight: 0.9,
                color: TEXT_KEY,
                fontVariantNumeric: "tabular-nums",
              }}
            >
              {(Math.round(plan.readout[tick] * 10) / 10 || 0).toFixed(1)}
            </span>
            {plan.unit != null && (
              <span style={{ ...label, fontSize: 26, marginLeft: 14 }}>
                {plan.unit}
              </span>
            )}
          </div>
          <Trace plan={plan} tick={tick} {...layout.trace} />
        </div>
      </div>
    </AbsoluteFill>
  );
};

/** Capture CSS px a highlight extends past its target on every side. */
export const HIGHLIGHT_PAD = 8;
/** Capture CSS px of a highlight's corner radius. */
export const HIGHLIGHT_RADIUS = 10;
/** Output px between a highlight and its caption. */
const CAPTION_GAP = 20;
const CAPTION_FONT = 30;
const CAPTION_PAD_X = 22;
const CAPTION_HEIGHT = 64;
/** Output px a caption keeps from the frame edges. */
const MARGIN = 56;
/** Output px above the frame bottom that the scope claims, by format. */
const SCOPE_CLEARANCE: Record<Format, number> = { portrait: 380, landscape: 300 };
/** Seconds between the reveals of a caption's words. */
const WORD_STAGGER_S = 0.045;

const within = (v: number, lo: number, hi: number): number =>
  Math.min(hi, Math.max(lo, v));

/** Box is an axis-aligned region of the output frame. */
export interface Box {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

interface HighlightProps {
  callout: CalloutFrame;
  /** Capture device scale factor: plane px per capture CSS px. */
  dsf: number;
  /** Plane scale into output px, so strokes keep their output width. */
  k: number;
}

/**
 * Highlight outlines a callout's target on the window plane, so it tilts and moves with
 * the UI. Its stroke draws in from the top left corner.
 */
export const Highlight = ({ callout, dsf, k }: HighlightProps): ReactElement => {
  const { rect, age } = callout;
  const draw = easeOutQuint(age / 0.6);
  const x = (rect.x - HIGHLIGHT_PAD) * dsf;
  const y = (rect.y - HIGHLIGHT_PAD) * dsf;
  const w = (rect.width + 2 * HIGHLIGHT_PAD) * dsf;
  const h = (rect.height + 2 * HIGHLIGHT_PAD) * dsf;
  const r = HIGHLIGHT_RADIUS * dsf;
  const id = `highlight-${rect.x}-${rect.y}`;
  return (
    <svg
      style={{ position: "absolute", left: 0, top: 0, overflow: "visible" }}
      width={1}
      height={1}
      opacity={callout.opacity}
    >
      <defs>
        <linearGradient
          id={id}
          x1={x}
          y1={y}
          x2={x + w}
          y2={y + h}
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stopColor="#fff" stopOpacity="1" />
          <stop offset="1" stopColor="#fff" stopOpacity="0.4" />
        </linearGradient>
      </defs>
      <rect
        x={x}
        y={y}
        width={w}
        height={h}
        rx={r}
        fill={`rgba(255, 255, 255, ${0.05 * draw})`}
      />
      <rect
        x={x}
        y={y}
        width={w}
        height={h}
        rx={r}
        fill="none"
        stroke="#fff"
        strokeOpacity={0.5}
        strokeWidth={8 / k}
        pathLength={1}
        strokeDasharray="1 1"
        strokeDashoffset={1 - draw}
        style={{ filter: `blur(${10 / k}px)` }}
      />
      <rect
        x={x}
        y={y}
        width={w}
        height={h}
        rx={r}
        fill="none"
        stroke={`url(#${id})`}
        strokeWidth={1.75 / k}
        pathLength={1}
        strokeDasharray="1 1"
        strokeDashoffset={1 - draw}
      />
    </svg>
  );
};

interface CaptionViewProps {
  callout: CalloutFrame;
  /** Where the highlight lands in the output frame. */
  box: Box;
  format: Format;
  width: number;
  height: number;
}

/**
 * CaptionView labels a highlight with a glass caption, below it when there is room
 * above the scope and above it otherwise, aligned to its left edge. Its words reveal
 * one after another.
 */
export const CaptionView = ({
  callout,
  box,
  format,
  width,
  height,
}: CaptionViewProps): ReactElement => {
  const { age, text } = callout;
  const [measure] = useState(() => document.createElement("canvas").getContext("2d"));
  if (measure == null) throw new Error("canvas 2D context unavailable");
  measure.font = `520 ${CAPTION_FONT}px ${SANS}`;
  measure.letterSpacing = `${-0.014 * CAPTION_FONT}px`;
  const captionWidth = measure.measureText(text).width + 2 * CAPTION_PAD_X + 2;
  const floor = height - SCOPE_CLEARANCE[format];
  const below = box.bottom + CAPTION_GAP + CAPTION_HEIGHT <= floor;
  const top = within(
    below ? box.bottom + CAPTION_GAP : box.top - CAPTION_GAP - CAPTION_HEIGHT,
    MARGIN,
    floor - CAPTION_HEIGHT,
  );
  const left = within(box.left, MARGIN, width - MARGIN - captionWidth);
  const rise = easeOutQuint((age - 0.12) / 0.5);
  const words = text.split(" ");
  return (
    <div
      style={{
        position: "absolute",
        left,
        top,
        height: CAPTION_HEIGHT,
        boxSizing: "border-box",
        padding: `0 ${CAPTION_PAD_X}px`,
        display: "flex",
        alignItems: "center",
        opacity: callout.opacity * rise,
        transform: `translateY(${(1 - rise) * (below ? -12 : 12)}px)`,
        borderRadius: 14,
        ...GLASS,
        fontFamily: SANS,
        fontSize: CAPTION_FONT,
        fontWeight: 520,
        letterSpacing: "-0.014em",
        color: TEXT_KEY,
        whiteSpace: "pre",
      }}
    >
      {words.map((word, i) => {
        const shown = easeOutQuint((age - 0.2 - i * WORD_STAGGER_S) / 0.4);
        return (
          <span
            key={i}
            style={{
              display: "inline-block",
              opacity: shown,
              transform: `translateY(${(1 - shown) * 8}px)`,
              filter: `blur(${(1 - shown) * 6}px)`,
            }}
          >
            {i < words.length - 1 ? `${word} ` : word}
          </span>
        );
      })}
    </div>
  );
};
