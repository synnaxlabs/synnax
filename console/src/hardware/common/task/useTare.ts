// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, task as clientTask } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { useCallback } from "react";

import { NULL_CLIENT_ERROR } from "@/errors";

export interface TareableChannel {
  key: string;
  channel: channel.Key;
}

export interface UseTareProps<C extends TareableChannel> {
  task: clientTask.Task | clientTask.Payload;
  isChannelTareable?: (channel: C) => boolean;
  isRunning: boolean;
}

export type UseTareReturn<C extends TareableChannel> = [
  tare: (key: channel.Key) => void,
  allowTare: (keys: string[], channels: C[]) => boolean,
  handleTare: (keys: string[], channels: C[]) => void,
];

export const useTare = <C extends TareableChannel>({
  task,
  isChannelTareable,
  isRunning,
}: UseTareProps<C>): UseTareReturn<C> => {
  const client = Synnax.use();
  const handleException = Status.useExceptionHandler();
  const tare = useMutation({
    onError: (e) => handleException(e, "Failed to tare channels"),
    mutationFn: async (key: channel.Key[]) => {
      if (client == null) throw NULL_CLIENT_ERROR;
      if (!(task instanceof clientTask.Task))
        throw new Error("Task has not been configured");
      await task.executeCommand("tare", { keys: [key] });
    },
  }).mutate;
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
