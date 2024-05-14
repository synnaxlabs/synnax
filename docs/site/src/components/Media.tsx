// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState, type DetailedHTMLProps, type ReactElement } from "react";

import { Video as Core } from "@synnaxlabs/pluto/video";

export interface VideoProps
  extends DetailedHTMLProps<
    React.VideoHTMLAttributes<HTMLVideoElement>,
    HTMLVideoElement
  > {
  id: string;
  themed?: boolean;
}

const CDN_ROOT = "https://synnax.nyc3.cdn.digitaloceanspaces.com/docs";

export const Video = ({ id, ...props }: VideoProps): ReactElement => {
  const theme = window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
  const href = `${CDN_ROOT}/${id}-${theme}.mp4`;
  return <Core.Video href={href} loop autoPlay muted {...props} />;
};

export const Image = ({ id, themed = true, ...props }: VideoProps): ReactElement => {
  const theme = window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
  let url = `${CDN_ROOT}/${id}`;
  if (themed) url += `-${theme}`;
  url += ".png";
  return <img src={url} className="image" {...props} />;
};
