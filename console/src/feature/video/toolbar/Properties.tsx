// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/video/toolbar/Properties.css";

import { type channel, type ranger, video } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { type Select } from "@synnaxlabs/lyra/select";
import { Access, Channel, Ranger, Video } from "@synnaxlabs/pluto";
import { DataType } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

const VIDEO_QUERY: Channel.ListQuery = {
  where: { dataType: { eq: DataType.VIDEO } },
};

const LIVE_TRIGGER_PROPS: Select.SingleTriggerProps = { placeholder: "Live" };

export const Properties = (): ReactElement => {
  const key = Video.useKey();
  const dispatch = Video.useSingleDispatch();
  const value = Video.useChannel();
  const ref = Video.useRange();
  const editable = Access.useUpdateGranted(video.ontologyID(key));
  const handleChange = (channel: channel.Key | undefined) =>
    dispatch(video.setChannel({ channel: channel ?? 0 }));
  const handleRangeChange = (range: ranger.Key | null) =>
    dispatch(
      video.setRange({
        range: range == null ? undefined : { variant: "key", key: range },
      }),
    );
  return (
    <Flex.Box x className={CSS.BE("video", "toolbar", "properties")}>
      <Input.Item label="Channel">
        <Channel.SelectSingle
          value={value === 0 ? undefined : value}
          onChange={handleChange}
          query={VIDEO_QUERY}
          allowNone
          disabled={!editable}
          className={CSS.BE("video", "channel-select")}
        />
      </Input.Item>
      <Input.Item label="Range">
        <Ranger.Select
          value={ref?.variant === "key" ? ref.key : undefined}
          onChange={handleRangeChange}
          triggerProps={LIVE_TRIGGER_PROPS}
          allowNone
          disabled={!editable}
          className={CSS.BE("video", "range-select")}
        />
      </Input.Item>
    </Flex.Box>
  );
};
