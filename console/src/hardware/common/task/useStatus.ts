// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Observe, Synnax, useSyncedRef } from "@synnaxlabs/pluto";
import { useCallback, useState as useReactState } from "react";
import { type z } from "zod/v4";

import { shouldExecuteCommand } from "@/hardware/common/task/shouldExecuteCommand";

/**
 * Explicit return type for the useState hook.
 * The object consists of:
 *  - state: The current state of the task.
 *  - triggerLoading: A function to set the state to "loading".
 */
export interface UseStatusReturn<StatusData extends z.ZodType = z.ZodType> {
  status: task.Status<StatusData>;
  triggerError: (message: string) => void;
  triggerLoading: (message: string) => void;
}

/**
 * useState takes in a task key and an optional initial state.
 *
 * @param key - The unique identifier for the task.
 * @param initialState - The optional initial state of the task.
 *
 * @returns An object containing:
 *   - state: The current state of the task, which includes:
 *     - status: A string that can be "loading", "running", or "paused".
 *     - message: An optional message string.
 *     - variant: An optional variant of type status.Variant.
 *   - triggerLoading: A function to set the state to "loading".
 */
export const useStatus = <StatusData extends z.ZodType = z.ZodType>(
  key: task.Key,
  initialState: task.Status<StatusData>,
  commandMessages: Record<string, string>,
): UseStatusReturn<StatusData> => {
  const [status, setStatus] = useReactState<task.Status<StatusData>>(initialState);
  const client = Synnax.use();
  // const status = state?.details.running ? RUNNING_STATUS : PAUSED_STATUS;
  const keyRef = useSyncedRef(key);
  const statusRef = useSyncedRef(status);
  const triggerLoading = useCallback(
    (message: string) =>
      setStatus((prev) => ({ ...prev, variant: "loading", message })),
    [],
  );
  Observe.useListener({
    key: [client?.key],
    open: async () => client?.hardware.tasks.openCommandObserver(),
    onChange: ({ task, type }) => {
      if (task !== keyRef.current || statusRef.current == null) return;
      if (shouldExecuteCommand<StatusData>(statusRef.current, type))
        triggerLoading(commandMessages[type]);
    },
  });
  Observe.useListener({
    key: [client?.key],
    open: async () => await client?.hardware.tasks.openStateObserver<StatusData>(),
    onChange: (status) => {
      if (status.details.task !== keyRef.current) return;
      setStatus(status);
    },
  });

  const triggerError = useCallback(
    (message: string) => setStatus((prev) => ({ ...prev, message, variant: "error" })),
    [],
  );
  return { status, triggerError, triggerLoading };
};
