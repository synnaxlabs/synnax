// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { AbsoluteFill, Img, staticFile, useCurrentFrame } from "remotion";

import { crop } from "@/director/camera";
import { type Tracks } from "@/director/director";
import { Cursor } from "@/remotion/Cursor";
import { Ripple, RIPPLE_TICKS } from "@/remotion/Ripple";
import { type Meta, type Timeline } from "@/timeline";

export type StudioVideoProps = {
  meta: Meta;
  tracks: Tracks;
  events: Timeline["events"];
};

/**
 * StudioVideo composites one captured frame per output frame under the virtual
 * camera, then draws the synthetic cursor and click ripples in output space.
 */
export const StudioVideo = ({
  meta,
  tracks,
  events,
}: StudioVideoProps): ReactElement => {
  const frame = useCurrentFrame();
  const idx = Math.min(frame, tracks.camera.length - 1);
  const cam = tracks.camera[idx];
  const prev = tracks.camera[Math.max(0, idx - 1)];
  const cur = tracks.cursor[Math.min(frame, tracks.cursor.length - 1)];
  const { width, height, dsf } = meta;

  const c = crop(cam, width, height);
  const scale = cam.amount * dsf;
  const name = String(Math.min(frame, meta.frames - 1)).padStart(6, "0");

  // Directional motion blur from per-frame camera travel, plus a light
  // isotropic term while the zoom amount changes. Sub-threshold blur is
  // dropped so settled frames stay tack sharp.
  const cPrev = crop(prev, width, height);
  const zoomBlur = Math.abs(cam.amount - prev.amount) * 14 * dsf;
  const blurX = Math.min(
    12 * dsf,
    Math.abs(c.x - cPrev.x) * scale * 0.25 + zoomBlur,
  );
  const blurY = Math.min(
    12 * dsf,
    Math.abs(c.y - cPrev.y) * scale * 0.25 + zoomBlur,
  );
  const blurred = blurX > 0.4 * dsf || blurY > 0.4 * dsf;

  const toScreen = (x: number, y: number): { left: number; top: number } => ({
    left: (x - c.x) * scale,
    top: (y - c.y) * scale,
  });

  const ripples = events.filter(
    (e) => e.type === "pointerdown" && frame >= e.tick && frame < e.tick + RIPPLE_TICKS,
  );

  return (
    <AbsoluteFill style={{ backgroundColor: "black", overflow: "hidden" }}>
      {blurred && (
        <svg width={0} height={0} style={{ position: "absolute" }}>
          <filter id="camera-blur">
            <feGaussianBlur stdDeviation={`${blurX} ${blurY}`} />
          </filter>
        </svg>
      )}
      <Img
        src={staticFile(`frames/${name}.png`)}
        style={{
          position: "absolute",
          width: width * scale,
          height: height * scale,
          left: -c.x * scale,
          top: -c.y * scale,
          filter: blurred ? "url(#camera-blur)" : undefined,
        }}
      />
      {ripples.map((e, i) => {
        if (e.type !== "pointerdown") return null;
        const pos = toScreen(e.x, e.y);
        return (
          <Ripple
            key={`${e.tick}-${i}`}
            progress={(frame - e.tick) / RIPPLE_TICKS}
            left={pos.left}
            top={pos.top}
            amount={cam.amount}
            dsf={dsf}
          />
        );
      })}
      <Cursor
        {...toScreen(cur.x, cur.y)}
        scale={cur.scale}
        pressed={cur.pressed}
        dsf={dsf}
        kind={cur.kind}
        opacity={cur.opacity ?? 1}
      />
    </AbsoluteFill>
  );
};
