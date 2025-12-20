// Copyright 2025 Synnax Labs, Inc.
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
  "channel" | "defaultName" | "id"
> {
  cmdChannel: channel.Key;
  stateChannel: channel.Key;
  itemKey: string;
}

export const WriteChannelNames = ({
  cmdChannel,
  stateChannel,
  itemKey,
  ...rest
}: WriteChannelNamesProps) => (
  <>
    <ChannelName
      channel={cmdChannel}
      defaultName="No Command Channel"
      id={getChannelNameID(itemKey, "cmd")}
      {...rest}
    />
    <ChannelName
      channel={stateChannel}
      className={CSS.B("state-channel")}
      defaultName="No State Channel"
      id={getChannelNameID(itemKey, "state")}
      {...rest}
    />
  </>
);
