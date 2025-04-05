// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Channel, Text } from "@synnaxlabs/pluto";
import { type Optional } from "@synnaxlabs/x";

import { CSS } from "@/css";

export interface ChannelNameProps
  extends Optional<Text.TextProps<Text.Level>, "level"> {
  channel: channel.Key;
  defaultName?: string;
}

export const ChannelName = ({
  channel,
  defaultName = "No Channel",
  ...rest
}: ChannelNameProps) => {
  const name = Channel.useName(channel, defaultName);
  return (
    <Text.Text
      className={CSS.BE("task", "channel-name")}
      color={channel ? undefined : "var(--pluto-warning-m1)"}
      level="small"
      {...rest}
    >
      {name}
    </Text.Text>
  );
};
