// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { Composition } from "remotion";

import { Film, type FilmProps } from "@/remotion/Film";
import {
  HERO,
  Hero,
  type HeroProps,
  SHOT,
  Shot,
  type ShotProps,
} from "@/remotion/Mail";
import { StudioVideo, type StudioVideoProps } from "@/remotion/StudioVideo";

/** Placeholder metadata; the CLI overrides everything via calculateMetadata. */
const DEFAULT_PROPS: StudioVideoProps = {
  meta: {
    version: 1,
    fps: 60,
    width: 1920,
    height: 1080,
    dsf: 2,
    theme: "light",
    frames: 60,
  },
  tracks: { camera: [], cursor: [], segments: [] },
  events: [],
};

/** Placeholder film; the CLI supplies the real plan via inputProps. */
const DEFAULT_FILM_PROPS: FilmProps = {
  meta: { ...DEFAULT_PROPS.meta, theme: "dark" },
  plan: {
    format: "portrait",
    width: 1080,
    height: 1350,
    fps: 60,
    perspective: 1800,
    shots: [{ type: "card", lines: ["Placeholder"] }],
    samples: [{ shot: 0, type: "card", opacity: 1, offset: 0 }],
  },
  overlays: { frames: [{ callouts: [] }] },
  cursor: [],
  events: [],
};

/** Placeholder hero; the CLI supplies the still via inputProps. */
const DEFAULT_HERO_PROPS: HeroProps = { source: "source.webp" };

/** Placeholder shot; the CLI supplies the capture via inputProps. */
const DEFAULT_SHOT_PROPS: ShotProps = {
  ...DEFAULT_PROPS,
  region: { x: 0, y: 0, width: 1920 },
};

export const Root = (): ReactElement => (
  <>
    <Composition
      id="studio"
      component={StudioVideo}
      durationInFrames={60}
      fps={60}
      width={3840}
      height={2160}
      defaultProps={DEFAULT_PROPS}
      calculateMetadata={({ props }) => ({
        durationInFrames: props.meta.frames,
        fps: props.meta.fps,
        width: Math.round(props.meta.width * props.meta.dsf),
        height: Math.round(props.meta.height * props.meta.dsf),
        props,
      })}
    />
    <Composition
      id="film"
      component={Film}
      durationInFrames={1}
      fps={60}
      width={1080}
      height={1350}
      defaultProps={DEFAULT_FILM_PROPS}
      calculateMetadata={({ props }) => ({
        durationInFrames: props.plan.samples.length,
        fps: props.plan.fps,
        width: props.plan.width,
        height: props.plan.height,
        props,
      })}
    />
    <Composition
      id="mail-hero"
      component={Hero}
      durationInFrames={1}
      fps={30}
      width={HERO.width}
      height={HERO.height}
      defaultProps={DEFAULT_HERO_PROPS}
    />
    <Composition
      id="mail-shot"
      component={Shot}
      durationInFrames={60}
      fps={60}
      width={SHOT.width}
      height={SHOT.height}
      defaultProps={DEFAULT_SHOT_PROPS}
      calculateMetadata={({ props }) => ({
        durationInFrames: props.meta.frames,
        fps: props.meta.fps,
        props,
      })}
    />
  </>
);
