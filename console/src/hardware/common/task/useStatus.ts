// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Task, useSyncedRef } from "@synnaxlabs/pluto";
import { useCallback, useState as useReactState } from "react";
import { type z } from "zod";

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
 * @param commandLoadingMessages - A record of command types to messages that should
 * be used when the command is being executed but has not received a result yet (i.e. loading).
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
): UseStatusReturn<StatusData> => {
  const [status, setStatus] = useReactState<task.Status<StatusData>>(() => ({
    ...initialState,
  }));
  const keyRef = useSyncedRef(key);
  const triggerLoading = useCallback(
    (message: string) =>
      setStatus((prev) => ({ ...prev, variant: "loading", message })),
    [],
  );
  const handleStatusUpdate = useCallback((status: task.Status) => {
    if (keyRef.current.length == 0) {
      setTimeout(() => handleStatusUpdate(status), 100);
      return;
    }
    if (status.details.task !== keyRef.current) return;
    setStatus(status);
  }, []);
  Task.useStatusSynchronizer(handleStatusUpdate);

  const triggerError = useCallback(
    (message: string) => setStatus((prev) => ({ ...prev, message, variant: "error" })),
    [],
  );
  return { status, triggerError, triggerLoading };
};
