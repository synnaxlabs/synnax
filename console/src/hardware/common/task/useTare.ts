import { type channel, task as clientTask } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { useCallback } from "react";

interface TareableChannel {
  key: string;
  enabled: boolean;
  channel: channel.Key;
}

export interface UseTareProps<C extends TareableChannel> {
  task: clientTask.Task | clientTask.Payload;
  isChannelTareable?: (channel: C) => boolean;
  isRunning: boolean;
}

export const useTare = <C extends TareableChannel>({
  task,
  isChannelTareable,
  isRunning,
}: UseTareProps<C>): [
  (key: channel.Key) => void,
  (keys: string[], channels: C[]) => boolean,
  (keys: string[], channels: C[]) => void,
] => {
  const client = Synnax.use();
  const handleException = Status.useExceptionHandler();
  const tare = useMutation({
    onError: (e) => handleException(e, "Failed to tare channels"),
    mutationFn: async (key: channel.Key[]) => {
      if (client == null) return;
      if (!(task instanceof clientTask.Task)) return;
      await task.executeCommand("tare", { keys: [key] });
    },
  }).mutate;
  const getTareableChannels = useCallback((keys: string[], channels: C[]) => {
    const keySet = new Set(keys);
    return channels.filter(
      (channel) => keySet.has(channel.key) && isChannelTareable?.(channel) !== false,
    );
  }, []);
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
