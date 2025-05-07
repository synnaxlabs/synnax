// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Channel, Status, Text } from "@synnaxlabs/pluto";
import { type Optional } from "@synnaxlabs/x";

import { CSS } from "@/css";

export interface ChannelNameProps
  extends Optional<Omit<Text.MaybeEditableProps<Text.Level>, "value">, "level"> {
  channel: channel.Key;
  defaultName?: string;
}

export const ChannelName = ({
  channel,
  defaultName = "No Channel",
  className,
  ...rest
}: ChannelNameProps) => {
  const [name, rename] = Channel.useName(channel, defaultName);
  const handleError = Status.useErrorHandler();
  const handleChange = (newName: string) => {
    handleError(
      async () => await rename(newName),
      `Failed to rename ${name} to ${newName}`,
    );
  };
  return (
    <Text.MaybeEditable
      className={CSS(className, CSS.BE("task", "channel-name"))}
      color={channel ? undefined : "var(--pluto-warning-m1)"}
      level="small"
      value={name}
      onChange={handleChange}
      allowDoubleClick={false}
      {...rest}
    />
  );
};
