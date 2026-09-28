// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@fontsource-variable/inter";
import "@fontsource/geist-mono/500.css";

import wordmark from "@synnaxlabs/media/static/logo/title-white-transparent.svg";
import { type ReactElement, useEffect, useState } from "react";
import {
  AbsoluteFill,
  Img,
  staticFile,
  useCurrentFrame,
  useDelayRender,
} from "remotion";

import {
  type CalloutFrame,
  type CursorTrack,
  type OverlayPlan,
  type Plane,
  project,
  type StagePlan,
} from "@/director";
import { type Card as CardShot, type End as EndShot } from "@/film";
import { Cursor } from "@/remotion/Cursor";
import { CalloutView, MONO, SANS, ScopeView } from "@/remotion/Overlay";
import { Ripple, RIPPLE_TICKS } from "@/remotion/Ripple";
import { type Event, type Meta, type Point } from "@/timeline";

// A type alias, not an interface: Remotion requires props assignable to a record.
export type FilmProps = {
  meta: Meta;
  plan: StagePlan;
  overlays: OverlayPlan;
  cursor: CursorTrack;
  events: Event[];
};

const GROUND = "#060607";
const FONT = SANS;
const TEXT_KEY = "#F7F8F8";
const TEXT_REST = "#8A8F98";
/** Synnax primary blue, the ground of the end card. */
const BRAND = "#3470CC";
/** Window corner radius in CSS px of the capture. */
const WINDOW_RADIUS = 12;

const frameName = (tick: number): string => String(tick).padStart(6, "0");

/** anchorOf returns the middle of the target edge a callout points at, in CSS px. */
const anchorOf = ({ rect, side }: CalloutFrame): Point => ({
  x: side === "right" ? rect.x + rect.width : rect.x,
  y: rect.y + rect.height / 2,
});

interface PlaneViewProps {
  meta: Meta;
  plan: StagePlan;
  plane: Plane;
  tick: number;
  cursor: CursorTrack;
  events: Event[];
}

const PlaneView = ({
  meta,
  plan,
  plane,
  tick,
  cursor,
  events,
}: PlaneViewProps): ReactElement => {
  const { cx, cy, scale: s, tilt } = plane;
  // The plane lays out at the capture's pixel size and moves only by transform.
  // Layout boxes snap to whole pixels, which jitters a slow push or pan.
  const d = meta.dsf;
  const k = s / d;
  const cur = cursor[Math.min(tick, cursor.length - 1)];
  const ripples = events.filter(
    (e) => e.type === "pointerdown" && tick >= e.tick && tick < e.tick + RIPPLE_TICKS,
  );
  return (
    <AbsoluteFill style={{ perspective: `${plan.perspective}px` }}>
      <div
        style={{
          position: "absolute",
          left: 0,
          top: 0,
          width: meta.width * d,
          height: meta.height * d,
          transformOrigin: "0 0",
          transform:
            `translate(${plan.width / 2}px, ${plan.height / 2}px) ` +
            `rotateX(${tilt.x}deg) rotateZ(${tilt.z}deg) ` +
            `translate(${-cx * s}px, ${-cy * s}px) scale(${k})`,
          borderRadius: WINDOW_RADIUS * d,
          overflow: "hidden",
          boxShadow:
            `0 0 0 ${1 / k}px rgba(255, 255, 255, 0.08), ` +
            `0 ${60 / k}px ${160 / k}px rgba(0, 0, 0, 0.7)`,
        }}
      >
        <Img
          src={staticFile(`frames/${frameName(tick)}.png`)}
          style={{
            display: "block",
            width: "100%",
            height: "100%",
            // Lifts the Console's dark theme off the near-black stage.
            filter: "brightness(1.1) contrast(1.03)",
          }}
        />
        {ripples.map((e, i) => {
          if (e.type !== "pointerdown") return null;
          return (
            <Ripple
              key={`${e.tick}-${i}`}
              progress={(tick - e.tick) / RIPPLE_TICKS}
              left={e.x * d}
              top={e.y * d}
              amount={1}
              dsf={d}
              theme={meta.theme}
            />
          );
        })}
        <Cursor
          left={cur.x * d}
          top={cur.y * d}
          scale={cur.scale}
          pressed={cur.pressed}
          dsf={d}
          kind={cur.kind}
          opacity={cur.opacity}
        />
        {/* Rim light: a lit top-left edge and a faint sheen across the glass. */}
        <div
          style={{
            position: "absolute",
            inset: 0,
            borderRadius: WINDOW_RADIUS * d,
            boxShadow: `inset 0 0 0 ${1 / k}px rgba(255, 255, 255, 0.14)`,
            background:
              "linear-gradient(135deg, rgba(255, 255, 255, 0.08), transparent 40%)",
            mixBlendMode: "screen",
          }}
        />
      </div>
    </AbsoluteFill>
  );
};

const Vignette = (): ReactElement => (
  <AbsoluteFill
    style={{
      background: `radial-gradient(ellipse at center, transparent 50%, ${GROUND}C0 100%)`,
    }}
  />
);

interface GrainProps {
  seed: number;
  opacity: number;
  blend: "overlay" | "screen";
}

/** Grain re-seeds every frame, which also breaks up H.264 banding on dark ground. */
const Grain = ({ seed, opacity, blend }: GrainProps): ReactElement => (
  <AbsoluteFill style={{ opacity, mixBlendMode: blend }}>
    <svg width="100%" height="100%">
      <filter id={`grain-${seed}`}>
        <feTurbulence
          type="fractalNoise"
          baseFrequency="0.85"
          numOctaves={2}
          seed={seed}
          stitchTiles="stitch"
        />
        <feColorMatrix type="saturate" values="0" />
      </filter>
      <rect width="100%" height="100%" filter={`url(#grain-${seed})`} />
    </svg>
  </AbsoluteFill>
);

interface CardViewProps {
  shot: CardShot;
  opacity: number;
  offset: number;
}

const CardView = ({ shot, opacity, offset }: CardViewProps): ReactElement => (
  <AbsoluteFill
    style={{
      alignItems: "center",
      justifyContent: "center",
      padding: "0 96px",
      opacity,
      transform: `translateY(${offset}px)`,
    }}
  >
    {shot.lines.map((line, i) => (
      <div
        key={line}
        style={{
          fontFamily: FONT,
          fontSize: 68,
          fontWeight: 510,
          letterSpacing: "-0.022em",
          lineHeight: 1.12,
          textAlign: "center",
          color: i === 0 ? TEXT_KEY : TEXT_REST,
        }}
      >
        {line}
      </div>
    ))}
  </AbsoluteFill>
);

interface EndViewProps {
  shot: EndShot;
  opacity: number;
  seed: number;
}

const EndView = ({ shot, opacity, seed }: EndViewProps): ReactElement => (
  <AbsoluteFill
    style={{
      background: [
        "radial-gradient(60% 45% at 0% 0%, rgba(255, 255, 255, 0.5), transparent 70%)",
        "radial-gradient(55% 40% at 100% 100%, rgba(255, 255, 255, 0.32), transparent 70%)",
        "radial-gradient(45% 35% at 0% 100%, rgba(255, 255, 255, 0.28), transparent 70%)",
        BRAND,
      ].join(", "),
    }}
  >
    <Grain seed={seed} opacity={0.22} blend="screen" />
    <AbsoluteFill
      style={{ alignItems: "center", justifyContent: "center", gap: 56, opacity }}
    >
      <Img src={wordmark} style={{ width: "54%" }} />
      <div
        style={{
          fontFamily: FONT,
          fontSize: 40,
          fontWeight: 450,
          letterSpacing: "-0.01em",
          color: "rgba(255, 255, 255, 0.88)",
          textAlign: "center",
          padding: "0 120px",
        }}
      >
        {shot.tagline}
      </div>
    </AbsoluteFill>
  </AbsoluteFill>
);

/**
 * Film draws one frame of a feature film from the director's stage plan: the capture
 * on its tilted plane, a text card, or the end card, over the dark stage.
 */
export const Film = ({
  meta,
  plan,
  overlays,
  cursor,
  events,
}: FilmProps): ReactElement => {
  const frame = useCurrentFrame();
  const { delayRender, continueRender, cancelRender } = useDelayRender();
  // Remotion needs the handle before the first frame is captured, so it cannot wait
  // for an effect.
  const [fontHandle] = useState(() => delayRender("fonts"));
  useEffect(() => {
    Promise.all([
      document.fonts.load(`510 68px ${FONT}`),
      document.fonts.load(`500 21px ${MONO}`),
    ])
      .then(() => continueRender(fontHandle))
      .catch((err: unknown) => cancelRender(err));
  }, [fontHandle, continueRender, cancelRender]);

  const index = Math.min(frame, plan.samples.length - 1);
  const sample = plan.samples[index];
  const layer = overlays.frames[index];
  const shot = plan.shots[sample.shot];
  let body: ReactElement;
  switch (sample.type) {
    case "take":
      body = (
        <PlaneView
          meta={meta}
          plan={plan}
          plane={sample.plane}
          tick={sample.tick}
          cursor={cursor}
          events={events}
        />
      );
      break;
    case "card":
      if (shot.type !== "card") throw new Error(`sample ${frame} is not a card shot`);
      body = <CardView shot={shot} opacity={sample.opacity} offset={sample.offset} />;
      break;
    case "end":
      if (shot.type !== "end") throw new Error(`sample ${frame} is not an end shot`);
      return <EndView shot={shot} opacity={sample.opacity} seed={frame} />;
  }
  return (
    <AbsoluteFill style={{ backgroundColor: GROUND, overflow: "hidden" }}>
      {body}
      <Vignette />
      {sample.type === "take" &&
        layer.callouts.map((callout) => (
          <CalloutView
            key={callout.text}
            callout={callout}
            edge={project(
              sample.plane,
              anchorOf(callout),
              plan.width,
              plan.height,
              plan.perspective,
            )}
            width={plan.width}
            height={plan.height}
          />
        ))}
      {overlays.scope != null && layer.scope != null && (
        <ScopeView plan={overlays.scope} frame={layer.scope} />
      )}
      <Grain seed={frame} opacity={0.05} blend="overlay" />
    </AbsoluteFill>
  );
};
