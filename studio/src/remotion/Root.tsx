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

export const Root = (): ReactElement => (
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
);
