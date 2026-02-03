// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";

import { CSS } from "@/css";
import { ChannelName, type ChannelNameProps } from "@/hardware/common/task/ChannelName";
import { getChannelNameID } from "@/hardware/common/task/getChannelNameID";

export interface WriteChannelNamesProps extends Omit<
  ChannelNameProps,
  "channel" | "defaultName" | "id" | "namePath"
> {
  cmdChannel: channel.Key;
  cmdNamePath: string;
  stateChannel: channel.Key;
  stateNamePath: string;
  itemKey: string;
}

export const WriteChannelNames = ({
  cmdChannel,
  cmdNamePath,
  stateNamePath,
  stateChannel,
  itemKey,
  ...rest
}: WriteChannelNamesProps) => (
  <>
    <ChannelName
      {...rest}
      channel={cmdChannel}
      id={getChannelNameID(itemKey, "cmd")}
      defaultName="No Command Channel"
      namePath={cmdNamePath}
    />
    <ChannelName
      {...rest}
      channel={stateChannel}
      className={CSS.B("state-channel")}
      defaultName="No State Channel"
      namePath={stateNamePath}
      id={getChannelNameID(itemKey, "state")}
    />
  </>
);
