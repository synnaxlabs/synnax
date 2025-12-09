// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DisconnectedError } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useIsRunning } from "@/hardware/common/task/Form";
import { useKey } from "@/hardware/common/task/useKey";

export interface TareableChannel {
  key: string;
  channel: channel.Key;
}

export type UseTareProps<C extends TareableChannel> = {
  isChannelTareable?: (channel: C) => boolean;
};

export type UseTareReturn<C extends TareableChannel> = [
  tare: (key: channel.Key) => void,
  allowTare: (keys: string[], channels: C[]) => boolean,
  handleTare: (keys: string[], channels: C[]) => void,
];

export const useTare = <C extends TareableChannel>({
  isChannelTareable,
}: UseTareProps<C> = {}): UseTareReturn<C> => {
  const client = Synnax.use();
  const key = useKey();
  const isRunning = useIsRunning();
  const handleError = Status.useErrorHandler();
  const tare = useCallback(
    (keys: channel.Key[]) => {
      if (client == null) throw new DisconnectedError();
      if (key == null) throw new Error("Task has not been configured");
      const args = { keys };
      handleError(
        async () =>
          await client.tasks.executeCommand({ task: key, type: "tare", args }),
        "Failed to tare channels",
      );
    },
    [client, key, handleError],
  );
  const getTareableChannels = useCallback(
    (keys: string[], channels: C[]) => {
      const keySet = new Set(keys);
      return channels.filter(
        (channel) => keySet.has(channel.key) && isChannelTareable?.(channel) !== false,
      );
    },
    [isChannelTareable],
  );
  const tareSingle = useCallback((key: channel.Key) => tare([key]), [tare]);
  const allowTare = useCallback(
    (keys: string[], channels: C[]) =>
      getTareableChannels(keys, channels).length > 0 && isRunning,
    [getTareableChannels, isRunning],
  );
  const handleTare = useCallback(
    (keys: string[], channels: C[]) => {
      const tareableChannels = getTareableChannels(keys, channels);
      tare(tareableChannels.map(({ channel }) => channel));
    },
    [getTareableChannels, tare],
  );
  return [tareSingle, allowTare, handleTare];
};
