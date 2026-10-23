// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import {
  type DetailedHTMLProps,
  type ImgHTMLAttributes,
  type ReactElement,
  type VideoHTMLAttributes,
} from "react";

import { mediaURL } from "@/components/media/url";
import { DARK } from "@/components/media/video";

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
    > {}

/**
 * Renders a looping, muted docs video that follows the reader's color scheme. The
 * script in `@/components/media/video` plays it while it is in view.
 */
export const Video = ({ id, themed = true, ...rest }: VideoProps): ReactElement => {
  // The time fragment makes iOS Safari render the first frame before playback.
  const src = (theme?: string): string => `${mediaURL(id, "mp4", theme)}#t=0.001`;
  return (
    <div className="docs-video">
      <video loop muted playsInline preload="metadata" {...rest}>
        {themed && <source src={src("dark")} media={DARK} type="video/mp4" />}
        <source src={src(themed ? "light" : undefined)} type="video/mp4" />
      </video>
      <div className="docs-video__play" aria-hidden>
        <Icon.Play />
      </div>
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
}

/** Renders a docs image that follows the reader's color scheme. */
export const Image = ({
  id,
  themed = true,
  extension = "png",
  loading = "lazy",
  decoding = "async",
  ...rest
}: ImageProps): ReactElement => {
  const img = (
    <img
      src={mediaURL(id, extension, themed ? "light" : undefined)}
      loading={loading}
      decoding={decoding}
      {...rest}
    />
  );
  if (!themed) return img;
  return (
    <picture className="docs-picture">
      <source srcSet={mediaURL(id, extension, "dark")} media={DARK} />
      {img}
    </picture>
  );
};
