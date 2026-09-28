// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";
import { AbsoluteFill } from "remotion";

import {
  type CalloutFrame,
  easeOutQuint,
  SCOPE_WINDOW_S,
  type ScopeFrame,
  type ScopePlan,
} from "@/director";
import { FPS } from "@/film";
import { type Point } from "@/timeline";

export const MONO = '"Geist Mono", monospace';
export const SANS = '"Inter Variable", sans-serif';
const TEXT_KEY = "#F7F8F8";
const TEXT_REST = "#8A8F98";

/** Output px of the scope's trace. */
const TRACE_WIDTH = 560;
const TRACE_HEIGHT = 118;
/** Ticks between trace points. */
const TRACE_STEP = 2;

const clamp = (t: number): number => Math.min(1, Math.max(0, t));

/** Overshoots a little, then settles: a dot that pops in. */
const easeOutBack = (t: number): number => {
  const c = clamp(t) - 1;
  return 1 + 2.7 * c ** 3 + 1.7 * c ** 2;
};

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
}

/** Trace draws the scope's last seconds of values, oldest at the left. */
const Trace = ({ plan, tick }: TraceProps): ReactElement => {
  const span = SCOPE_WINDOW_S * FPS;
  const first = tick - span + 1;
  const x = (t: number): number => ((t - first) / (span - 1)) * TRACE_WIDTH;
  const y = (v: number): number =>
    TRACE_HEIGHT - ((v - plan.low) / (plan.high - plan.low)) * TRACE_HEIGHT;
  const points: string[] = [];
  for (let t = Math.max(0, first); t <= tick; t += TRACE_STEP)
    points.push(`${x(t).toFixed(1)},${y(plan.trace[t]).toFixed(1)}`);
  points.push(`${x(tick).toFixed(1)},${y(plan.trace[tick]).toFixed(1)}`);
  const line = `M${points.join("L")}`;
  const start = x(Math.max(0, first)).toFixed(1);
  const area = `${line}L${TRACE_WIDTH},${TRACE_HEIGHT}L${start},${TRACE_HEIGHT}Z`;
  const headY = y(plan.trace[tick]);
  return (
    <svg
      width={TRACE_WIDTH}
      height={TRACE_HEIGHT}
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
            width={TRACE_WIDTH + 40}
            height={TRACE_HEIGHT + 40}
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
          x2={TRACE_WIDTH}
          y1={f * TRACE_HEIGHT}
          y2={f * TRACE_HEIGHT}
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
        cx={TRACE_WIDTH}
        cy={headY}
        r={16}
        fill={plan.color}
        opacity={0.45}
        filter="url(#scope-glow)"
      />
      <circle cx={TRACE_WIDTH} cy={headY} r={5.5} fill="#fff" />
    </svg>
  );
};

interface ScopeViewProps {
  plan: ScopePlan;
  frame: ScopeFrame;
}

/**
 * ScopeView draws the live telemetry strip along the bottom of the frame: the channel,
 * its readout, and a trace of its last seconds.
 */
export const ScopeView = ({ plan, frame }: ScopeViewProps): ReactElement => {
  const { opacity, tick } = frame;
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
      <AbsoluteFill
        style={{
          top: "auto",
          height: 540,
          background:
            "linear-gradient(to top, rgba(6, 6, 7, 0.97) 0%, rgba(6, 6, 7, 0.9) 42%, rgba(6, 6, 7, 0.6) 68%, transparent 100%)",
        }}
      />
      <div
        style={{
          position: "absolute",
          left: 64,
          right: 64,
          bottom: 72,
          transform: `translateY(${(1 - opacity) * 10}px)`,
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
              width: 356,
              display: "flex",
              alignItems: "baseline",
              whiteSpace: "nowrap",
            }}
          >
            <span
              style={{
                fontFamily: SANS,
                fontSize: 108,
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
          <Trace plan={plan} tick={tick} />
        </div>
      </div>
    </AbsoluteFill>
  );
};

/** Output px of a callout's leader line when it has room. */
const LEADER = 64;
/** Output px between the anchor dot and the start of the leader. */
const GAP = 14;
const PILL_FONT = 31;
const PILL_PAD_X = 24;
const PILL_HEIGHT = 70;
/** Output px between a target's edge and the callout's dot. */
const DOT_CLEARANCE = 12;
/** Output px a callout keeps from the frame edges. */
const MARGIN = 56;
/** Output px above the frame bottom that the scope strip claims. */
const SCOPE_CLEARANCE = 380;

const smoothstep = (t: number): number => {
  const c = clamp(t);
  return c * c * (3 - 2 * c);
};

const within = (v: number, lo: number, hi: number): number =>
  Math.min(hi, Math.max(lo, v));

interface CalloutViewProps {
  callout: CalloutFrame;
  /** Where the target's edge lands in the output frame. */
  edge: Point;
  width: number;
  height: number;
}

/**
 * CalloutView labels a point on the window: a dot on the target, a leader, and a pill.
 * It keeps its pill inside the frame and clear of the scope, lifting it above the dot
 * when the pill has to cross it.
 */
export const CalloutView = ({
  callout,
  edge,
  width,
  height,
}: CalloutViewProps): ReactElement => {
  const { side, age, text } = callout;
  const right = side === "right";
  // The target's edge is often the end of a label, so the dot sits just clear of it.
  const anchor = { x: edge.x + (right ? DOT_CLEARANCE : -DOT_CLEARANCE), y: edge.y };
  const [measure] = useState(() => document.createElement("canvas").getContext("2d"));
  if (measure == null) throw new Error("canvas 2D context unavailable");
  measure.font = `520 ${PILL_FONT}px ${SANS}`;
  measure.letterSpacing = `${-0.012 * PILL_FONT}px`;
  const pill = measure.measureText(text).width + 2 * PILL_PAD_X + 2;
  const preferred = right ? anchor.x + GAP + LEADER : anchor.x - GAP - LEADER - pill;
  const x = within(preferred, MARGIN, width - MARGIN - pill);
  const encroach = right ? anchor.x + GAP + 24 - x : x + pill - (anchor.x - GAP - 24);
  const lift = smoothstep(encroach / 60) * (PILL_HEIGHT / 2 + 48);
  const y = within(
    anchor.y - lift,
    MARGIN + PILL_HEIGHT / 2,
    height - SCOPE_CLEARANCE - PILL_HEIGHT / 2,
  );
  const end = {
    x: within(anchor.x, x, x + pill),
    y: within(anchor.y, y - PILL_HEIGHT / 2, y + PILL_HEIGHT / 2),
  };
  const reach = Math.hypot(end.x - anchor.x, end.y - anchor.y);
  const unit =
    reach > 0
      ? { x: (end.x - anchor.x) / reach, y: (end.y - anchor.y) / reach }
      : { x: 0, y: 0 };
  const from = { x: anchor.x + unit.x * GAP, y: anchor.y + unit.y * GAP };
  const pop = easeOutBack(age / 0.3);
  const ring = clamp(age / 0.8);
  const draw = easeOutQuint((age - 0.08) / 0.3);
  const reveal = easeOutQuint((age - 0.22) / 0.45);
  const ringSize = 64 * (0.35 + ring);
  return (
    <AbsoluteFill style={{ opacity: callout.opacity }}>
      <svg width={width} height={height} style={{ position: "absolute", inset: 0 }}>
        <line
          x1={from.x}
          y1={from.y}
          x2={from.x + (end.x - from.x) * draw}
          y2={from.y + (end.y - from.y) * draw}
          stroke="rgba(255, 255, 255, 0.7)"
          strokeWidth={2}
          strokeLinecap="round"
        />
      </svg>
      <div
        style={{
          position: "absolute",
          left: anchor.x - ringSize / 2,
          top: anchor.y - ringSize / 2,
          width: ringSize,
          height: ringSize,
          borderRadius: "50%",
          border: "2px solid rgba(255, 255, 255, 0.85)",
          opacity: (1 - ring) * 0.9,
        }}
      />
      <div
        style={{
          position: "absolute",
          left: anchor.x - 7,
          top: anchor.y - 7,
          width: 14,
          height: 14,
          borderRadius: 7,
          background: "#fff",
          boxShadow:
            "0 0 0 4px rgba(255, 255, 255, 0.22), 0 0 18px rgba(255, 255, 255, 0.6)",
          transform: `scale(${pop})`,
        }}
      />
      <div
        style={{
          position: "absolute",
          left: x,
          top: y - PILL_HEIGHT / 2,
          width: pill,
          height: PILL_HEIGHT,
          boxSizing: "border-box",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          transform: `translate(${unit.x * (1 - reveal) * -14}px, ${unit.y * (1 - reveal) * -14}px)`,
          opacity: reveal,
          filter: `blur(${(1 - reveal) * 8}px)`,
          borderRadius: 16,
          background: "rgba(14, 16, 22, 0.9)",
          border: "1px solid rgba(255, 255, 255, 0.14)",
          boxShadow:
            "0 18px 50px rgba(0, 0, 0, 0.55), inset 0 1px 0 rgba(255, 255, 255, 0.06)",
          fontFamily: SANS,
          fontSize: PILL_FONT,
          fontWeight: 520,
          letterSpacing: "-0.012em",
          color: TEXT_KEY,
          whiteSpace: "nowrap",
        }}
      >
        {text}
      </div>
    </AbsoluteFill>
  );
};
