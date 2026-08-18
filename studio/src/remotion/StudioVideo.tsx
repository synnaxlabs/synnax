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
  const cam = tracks.camera[Math.min(frame, tracks.camera.length - 1)];
  const cur = tracks.cursor[Math.min(frame, tracks.cursor.length - 1)];
  const { width, height, dsf } = meta;

  const c = crop(cam, width, height);
  const scale = cam.amount * dsf;
  const name = String(Math.min(frame, meta.frames - 1)).padStart(6, "0");

  const toScreen = (x: number, y: number): { left: number; top: number } => ({
    left: (x - c.x) * scale,
    top: (y - c.y) * scale,
  });

  const ripples = events.filter(
    (e) => e.type === "pointerdown" && frame >= e.tick && frame < e.tick + RIPPLE_TICKS,
  );

  return (
    <AbsoluteFill style={{ backgroundColor: "black", overflow: "hidden" }}>
      <Img
        src={staticFile(`frames/${name}.png`)}
        style={{
          position: "absolute",
          width: width * scale,
          height: height * scale,
          left: -c.x * scale,
          top: -c.y * scale,
          imageRendering: cam.amount * dsf <= dsf ? "auto" : "auto",
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
      />
    </AbsoluteFill>
  );
};
