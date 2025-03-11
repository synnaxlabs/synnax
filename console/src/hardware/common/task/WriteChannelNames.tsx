// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";

import { ChannelName } from "@/hardware/common/task/ChannelName";

export interface WriteChannelNamesProps {
  cmdChannel: channel.Key;
  stateChannel: channel.Key;
}

export const WriteChannelNames = ({
  cmdChannel,
  stateChannel,
}: WriteChannelNamesProps) => (
  <>
    <ChannelName channel={cmdChannel} defaultName="No Command Channel" />
    <ChannelName channel={stateChannel} defaultName="No State Channel" />
  </>
);
