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
import { type status } from "@synnaxlabs/x";
import { useCallback, useState as useReactState } from "react";

import { shouldExecuteCommand } from "@/hardware/common/task/shouldExecuteCommand";
import {
  LOADING_STATUS,
  PAUSED_STATUS,
  RUNNING_STATUS,
  type Status,
} from "@/hardware/common/task/types";

export interface StateDetails {
  running: boolean;
  message?: string;
}

export interface State {
  status: Status;
  message?: string;
  variant?: status.Variant;
}

const parseState = <D extends StateDetails>(state?: task.State<D>): State => ({
  status: state?.details?.running ? RUNNING_STATUS : PAUSED_STATUS,
  message: state?.details?.message,
  variant: state?.variant,
});

/**
 * Explicit return type for the useState hook.
 * The object consists of:
 *  - state: The current state of the task.
 *  - triggerError: A function to set the state to "paused" with an error message.
 *  - triggerLoading: A function to set the state to "loading".
 */
export interface UseStateReturn {
  state: State;
  triggerError: (message: string) => void;
  triggerLoading: () => void;
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
export const useState = <D extends StateDetails>(
  key: task.Key,
  initialState?: task.State<D>,
): UseStateReturn => {
  const [state, setState] = useReactState<State>(() => parseState(initialState));
  const { status } = state;
  const keyRef = useSyncedRef(key);
  const statusRef = useSyncedRef(status);
  const handleStateUpdate = useCallback((state: task.State) => {
    if (state.task !== keyRef.current) return;
    setState(parseState(state as task.State<D>));
  }, []);
  Task.useStateSynchronizer(handleStateUpdate);
  const handleCommandUpdate = useCallback(({ task, type }: task.Command) => {
    if (task !== keyRef.current) return;
    if (shouldExecuteCommand(statusRef.current, type)) setState(LOADING_STATE);
  }, []);
  Task.useCommandSynchronizer(handleCommandUpdate);
  const triggerLoading = useCallback(() => setState(LOADING_STATE), []);
  const triggerError = useCallback(
    (message: string) => setState({ status: "paused", message, variant: "error" }),
    [],
  );
  return { state, triggerError, triggerLoading };
};

export const LOADING_STATE: State = { status: LOADING_STATUS, variant: "loading" };
