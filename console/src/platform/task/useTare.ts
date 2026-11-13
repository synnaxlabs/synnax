// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DisconnectedError, query } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Channel, Synnax } from "@synnaxlabs/pluto";

import { useIsSnapshot } from "@/platform/task/Form";
import { useKey } from "@/platform/task/useKey";
import { useIsRunning } from "@/platform/task/useStatus";

/** @returns a function that zeroes the task's channels at their present values. */
export const useTare = (): ((channels: channel.Key[]) => void) => {
  const client = Synnax.use();
  const key = useKey();
  const handleError = Status.useErrorHandler();
  return (keys) => {
    if (client == null) throw new DisconnectedError();
    if (key == null) throw new Error("Task has not been configured");
    handleError(
      async () =>
        await client.tasks.executeCommand({ task: key, type: "tare", args: { keys } }),
      "Failed to tare channels",
    );
  };
};

export interface Tareable {
  /** The float channels among the given ones. */
  floats: channel.Key[];
  /** The task runs, so a tare applies now. */
  running: boolean;
}

/**
 * @returns the float channels among the given ones and whether the task runs, or null
 * for a snapshot, which never tares. A key of 0 stands for no channel.
 */
export const useTareable = (channels: channel.Key[]): Tareable | null => {
  const keys = channels.filter((key) => key !== 0);
  const { data } = Channel.useResultMultiple(
    keys.length === 0 ? null : query.keysRequest(keys),
  );
  const isSnapshot = useIsSnapshot();
  const running = useIsRunning();
  if (isSnapshot) return null;
  const floats = (data ?? [])
    .filter(({ key, dataType }) => keys.includes(key) && dataType.isFloat)
    .map(({ key }) => key);
  return { floats, running };
};
