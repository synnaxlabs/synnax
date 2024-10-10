// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Video as Core } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";

import { Layout } from "@/layout";

export const LAYOUT_TYPE = "video";

export const SELECTABLE: Layout.Selectable = {
  key: LAYOUT_TYPE,
  title: "Video",
  icon: <Icon.Video />,
  create: (key) => create({ key }),
};

export const Video: Layout.Renderer = () => {
  return (
    <Align.Space align="center" justify="center" style={{ height: "100%" }}>
      <Core.Video
        autoPlay
        href="http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4"
      />
    </Align.Space>
  );
};

export const create = (initial: Omit<Partial<Layout.State>, "type">): Layout.State => {
  const key = initial.key ?? id.id();
  return {
    key,
    name: "Video",
    icon: "Video",
    location: "mosaic",
    type: LAYOUT_TYPE,
    windowKey: key,
  };
};
