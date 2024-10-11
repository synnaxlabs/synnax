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
import { id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { useEffect, useRef } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { useSelectCursor } from "@/playback/selector";
import { Range } from "@/range";

export const LAYOUT_TYPE = "video";

export const SELECTABLE: Layout.Selectable = {
  key: LAYOUT_TYPE,
  title: "Video",
  icon: <Icon.Video />,
  create: (key) => create({ key }),
};

export const Video: Layout.Renderer = () => {
  const ref = useRef<HTMLVideoElement>(null);
  const pos = useSelectCursor();
  const tr = Range.useSelect();
  useEffect(() => {
    if (ref.current == null || tr == null || tr.variant != "static") return;
    ref.current.currentTime = new TimeSpan(pos - tr.timeRange.start).seconds;
  }, [pos]);

  return (
    <Align.Space align="center" justify="center" style={{ height: "100%" }}>
      <video ref={ref} className={CSS(CSS.B("video"))} muted style={{ width: "100%" }}>
        <source
          src="http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4"
          type="video/mp4"
        />
      </video>
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
