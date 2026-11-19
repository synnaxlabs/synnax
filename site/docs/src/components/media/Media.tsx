// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Progress } from "@synnaxlabs/lyra/progress";
import {
  type DetailedHTMLProps,
  type ImgHTMLAttributes,
  type ReactElement,
  type VideoHTMLAttributes,
} from "react";

import { CDN } from "@/components/media/cdn";
import { DARK } from "@/components/media/video";
import videos from "@/components/media/videos.json";

interface Uploaded {
  version: string;
  width: number;
  height: number;
}

// Studio's upload writes the version and size of each video here.
const VIDEOS: Record<string, Uploaded> = videos;

interface MediaProps {
  id: string;
  themed?: boolean;
}

export interface VideoProps
  extends
    MediaProps,
    Omit<
      DetailedHTMLProps<VideoHTMLAttributes<HTMLVideoElement>, HTMLVideoElement>,
      "id"
    > {
  /** Describes what the video shows, for screen readers. */
  label: string;
}

/**
 * Renders a looping, muted docs video that follows the reader's color scheme. The
 * script in `@/components/media/video` plays it while it is in view, and the script in
 * `@/components/media/lightbox` expands it.
 * @throws {Error} if Studio has uploaded no version of the video.
 */
export const Video = ({
  id,
  label,
  themed = true,
  style,
  ...rest
}: VideoProps): ReactElement => {
  const uploaded = VIDEOS[id];
  if (uploaded == null)
    throw new Error(
      `no version of video ${id}: add an entry to studio/videos.ts, then run pnpm upload ${id} in studio/`,
    );
  const { version, width, height } = uploaded;
  // The time fragment makes iOS Safari render the first frame before playback.
  const src = (theme?: string): string => `${CDN.video(id, version, theme)}#t=0.001`;
  return (
    <div className="docs-video">
      <video
        loop
        muted
        playsInline
        preload="metadata"
        aria-label={label}
        style={{ aspectRatio: `${width} / ${height}`, ...style }}
        {...rest}
      >
        {themed && <source src={src("dark")} media={DARK} type="video/mp4" />}
        <source src={src(themed ? "light" : undefined)} type="video/mp4" />
      </video>
      <div className="docs-video__play" aria-hidden>
        <Icon.Play />
      </div>
      <div className="docs-video__progress" aria-hidden>
        <Progress.Progress thin value={0} />
      </div>
      <Flex.Box x gap="small" className="docs-video__controls">
        <Button.Button
          variant="outlined"
          size="small"
          className="docs-video__play-button"
          aria-label="Play video"
          propagateClick
        >
          <Icon.Play />
        </Button.Button>
        <Button.Button
          variant="outlined"
          size="small"
          className="docs-video__pause-button"
          aria-label="Pause video"
          propagateClick
        >
          <Icon.Pause />
        </Button.Button>
        <Button.Button
          variant="outlined"
          size="small"
          className="docs-video__replay"
          aria-label="Replay video"
          propagateClick
        >
          <Icon.Replay />
        </Button.Button>
        <Button.Button
          variant="outlined"
          size="small"
          className="docs-video__expand"
          aria-label="Expand video"
          propagateClick
        >
          <Icon.Expand />
        </Button.Button>
      </Flex.Box>
    </div>
  );
};

export interface ImageProps
  extends
    MediaProps,
    Omit<
      DetailedHTMLProps<ImgHTMLAttributes<HTMLImageElement>, HTMLImageElement>,
      "id"
    > {
  extension?: "png" | "jpg" | "jpeg" | "webp" | "svg";
  /** Describes what the image shows. Use an empty string only for decoration. */
  alt: string;
}

/**
 * Renders a docs image that follows the reader's color scheme. The script in
 * `@/components/media/lightbox` expands it.
 */
export const Image = ({
  id,
  themed = true,
  extension = "png",
  loading = "lazy",
  decoding = "async",
  className,
  ...rest
}: ImageProps): ReactElement => {
  const img = (
    <img
      src={CDN.url(id, extension, themed ? "light" : undefined)}
      loading={loading}
      decoding={decoding}
      className={className == null ? "docs-image" : `docs-image ${className}`}
      {...rest}
    />
  );
  if (!themed) return img;
  return (
    <picture className="docs-picture">
      <source srcSet={CDN.url(id, extension, "dark")} media={DARK} />
      {img}
    </picture>
  );
};
