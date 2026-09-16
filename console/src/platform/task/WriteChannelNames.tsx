// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";

import { CSS } from "@/platform/css";
import { ChannelName, type ChannelNameProps } from "@/platform/task/ChannelName";
import { getChannelNameID } from "@/platform/task/getChannelNameID";

export interface CommandStatePair {
  command: channel.Key;
  state: channel.Key;
}

export interface WriteChannelNamesProps<D> extends Omit<
  ChannelNameProps<D>,
  "channel" | "defaultName" | "id" | "namePath" | "resolve"
> {
  cmdChannel: channel.Key;
  cmdNamePath: string;
  stateChannel: channel.Key;
  stateNamePath: string;
  itemKey: string;
  /** Returns the pair the device map binds the row to, 0 for each it lacks. */
  resolve: (device: D) => CommandStatePair;
}

export const WriteChannelNames = <D,>({
  cmdChannel,
  cmdNamePath,
  stateNamePath,
  stateChannel,
  itemKey,
  resolve,
  ...rest
}: WriteChannelNamesProps<D>) => (
  <>
    <ChannelName
      {...rest}
      channel={cmdChannel}
      resolve={(device: D) => resolve(device).command}
      id={getChannelNameID(itemKey, "cmd")}
      defaultName="No command channel"
      namePath={cmdNamePath}
    />
    <ChannelName
      {...rest}
      channel={stateChannel}
      resolve={(device: D) => resolve(device).state}
      className={CSS.B("state-channel")}
      defaultName="No state channel"
      namePath={stateNamePath}
      id={getChannelNameID(itemKey, "state")}
    />
  </>
);
