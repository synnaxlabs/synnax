// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Status, Synnax, Text, useAsyncEffect } from "@synnaxlabs/pluto";
import { type Optional } from "@synnaxlabs/x";
import { useState } from "react";

import { CSS } from "@/css";
import { NULL_CLIENT_ERROR } from "@/errors";

export interface ChannelNameProps
  extends Optional<Omit<Text.MaybeEditableProps<Text.Level>, "value">, "level"> {
  channel: channel.Key;
  defaultName?: string;
}

export const ChannelName = ({
  channel,
  defaultName = "No Channel",
  ...rest
}: ChannelNameProps) => {
  const [name, setName] = useState(defaultName);
  const client = Synnax.use();
  useAsyncEffect(async () => {
    if (channel === 0) return;
    const ch = await client?.channels.retrieve(channel);
    if (ch != null) setName(ch.name);
  }, [channel]);
  const handleError = Status.useErrorHandler();
  const handleChange = (newName: string) => {
    const oldName = name;
    handleError(async () => {
      if (client == null) throw NULL_CLIENT_ERROR;
      setName(newName);
      try {
        await client.channels.rename(channel, newName);
      } catch (e) {
        setName(oldName);
        throw e;
      }
    }, `Failed to rename ${oldName} to ${newName}`);
  };
  return (
    <Text.MaybeEditable
      className={CSS.BE("task", "channel-name")}
      color={channel ? undefined : "var(--pluto-warning-m1)"}
      level="small"
      value={name}
      onChange={handleChange}
      allowDoubleClick={false}
      {...rest}
    />
  );
};
