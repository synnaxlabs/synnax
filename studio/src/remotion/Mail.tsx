// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { AbsoluteFill, Img, OffthreadVideo, staticFile } from "remotion";

export type MailProps = {
  /** source is the file, in the public dir, that the window shows. */
  source: string;
  video: boolean;
  /** frames is the length of the composition, which ends on the frame to show. */
  frames: number;
  fps: number;
};

/** WIDTH and HEIGHT are the size of a mail copy in CSS pixels. */
export const WIDTH = 496;
export const HEIGHT = 310;

// The docs hero lays out its window in pixels of a 792 px wide hero.
const PX = WIDTH / 792;
const FROM = "#3774d0";
const TO = "#a855f7";

const glow = (color: string, percent: number): string =>
  `color-mix(in oklab, ${color} ${percent}%, transparent)`;

/**
 * Mail draws a still or a video frame as a window on the deep docs hero gradient, run
 * off the right and the bottom as in the docs `bleed` layout.
 */
export const Mail = ({ source, video }: MailProps): ReactElement => (
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
        top: 50 * PX,
        left: 56 * PX,
        width: 960 * PX,
        overflow: "hidden",
        borderRadius: 12 * 0.75 * PX,
        boxShadow:
          "0 0 0 1px rgb(255 255 255 / 18%), 0 24px 56px -20px rgb(10 12 40 / 40%)",
      }}
    >
      {video ? (
        <OffthreadVideo
          src={staticFile(source)}
          muted
          style={{ display: "block", width: "100%" }}
        />
      ) : (
        <Img src={staticFile(source)} style={{ display: "block", width: "100%" }} />
      )}
    </div>
  </AbsoluteFill>
);
