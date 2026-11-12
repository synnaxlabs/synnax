// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { AbsoluteFill, Img, staticFile } from "remotion";

import { StudioVideo, type StudioVideoProps } from "@/remotion/StudioVideo";

/** HERO is the size of a letter's hero image in CSS pixels. */
export const HERO = { width: 576, height: 320 };

/** SHOT is the size of a letter's feature image in CSS pixels. */
export const SHOT = { width: 512, height: 288 };

export type HeroProps = {
  /** source is the still, in the public dir, that the window shows. */
  source: string;
};

// The docs hero lays out its window in pixels of a 792 px wide hero.
const PX = HERO.width / 792;
const FROM = "#7c3aed";
const TO = "#ff007b";

const glow = (color: string, percent: number): string =>
  `color-mix(in oklab, ${color} ${percent}%, transparent)`;

/**
 * Hero draws a still as a window on the deep docs hero gradient, run off the right and
 * the bottom as in the docs `bleed` layout.
 */
export const Hero = ({ source }: HeroProps): ReactElement => (
  <AbsoluteFill
    style={{
      background: [
        `radial-gradient(75% 95% at 0% 0%, ${glow(FROM, 85)}, transparent 70%)`,
        `radial-gradient(70% 90% at 100% 100%, ${glow(TO, 65)}, transparent 70%)`,
        "#282c41",
      ].join(", "),
    }}
  >
    <div
      style={{
        position: "absolute",
        top: 64 * PX,
        left: 76 * PX,
        width: 880 * PX,
        overflow: "hidden",
        borderRadius: 12 * 0.75 * PX,
        boxShadow:
          "0 0 0 1px rgb(255 255 255 / 18%), 0 24px 56px -20px rgb(10 12 40 / 40%)",
      }}
    >
      <Img src={staticFile(source)} style={{ display: "block", width: "100%" }} />
    </div>
  </AbsoluteFill>
);

/** Region is a box in pixels of the full-size directed frame. */
export interface Region {
  x: number;
  y: number;
  width: number;
}

export type ShotProps = StudioVideoProps & { region: Region };

/** Shot draws a region of a directed video frame, at the full capture resolution. */
export const Shot = ({ region, ...props }: ShotProps): ReactElement => {
  const { width, height, dsf } = props.meta;
  const scale = SHOT.width / region.width;
  return (
    <AbsoluteFill style={{ overflow: "hidden" }}>
      <div
        style={{
          position: "absolute",
          width: width * dsf,
          height: height * dsf,
          transform: [
            `translate(${-region.x * scale}px, ${-region.y * scale}px)`,
            `scale(${scale})`,
          ].join(" "),
          transformOrigin: "0 0",
        }}
      >
        <StudioVideo {...props} />
      </div>
    </AbsoluteFill>
  );
};
