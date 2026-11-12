// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DisconnectedError } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";

import { useKey } from "@/platform/task/useKey";

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
