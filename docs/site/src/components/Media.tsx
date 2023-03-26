// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Video as PVideo } from "@synnaxlabs/pluto";

export interface VideoProps {
  id: string;
}

const CDN_ROOT = "https://synnax.nyc3.cdn.digitaloceanspaces.com/docs";

export const Video = ({ id }: VideoProps): JSX.Element => {
  const theme = localStorage.getItem("theme") ?? "light";
  const modifier = theme?.toLowerCase().includes("dark") ? "dark" : "light";
  return <PVideo href={`${CDN_ROOT}/${id}-${modifier}.mp4`} loop autoPlay />;
};

export const Image = ({ id }: VideoProps): JSX.Element => {
  const theme = localStorage.getItem("theme") ?? "light";
  const modifier = theme?.toLowerCase().includes("dark") ? "dark" : "light";
  return <img src={`${CDN_ROOT}/${id}-${modifier}.png`} />;
};
