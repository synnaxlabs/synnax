// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Observe, Status as PStatus, Synnax } from "@synnaxlabs/pluto";
import { useCallback, useMemo, useState as useReactState } from "react";

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
  variant?: PStatus.Variant;
}

/**
 * Explicit return type for the useState hook.
 * The tuple consists of:
 *  - state: The current state of the task.
 *  - triggerLoading: A function to set the state to "loading".
 */
export type UseStateReturn = [state: State, triggerLoading: () => void];

/**
 * useState takes in a task key and an optional initial state.
 *
 * @param key - The unique identifier for the task.
 * @param initialState - The optional initial state of the task.
 *
 * @returns A tuple containing:
 *   - state: The current state of the task, which includes:
 *     - status: A string that can be "loading", "running", or "paused".
 *     - message: An optional message string.
 *     - variant: An optional variant of type PStatus.Variant.
 *   - triggerLoading: A function to set the state to "loading".
 */
export const useState = <D extends StateDetails>(
  key: task.Key,
  initialState?: task.State<D>,
): UseStateReturn => {
  const [state, setState] = useReactState<State>({
    status: initialState?.details?.running ? RUNNING_STATUS : PAUSED_STATUS,
    message: initialState?.details?.message,
    variant: parseVariant(initialState?.variant),
  });
  const client = Synnax.use();
  const status = state.status;
  Observe.useListener({
    key: [client?.key, key, status],
    open: async () => client?.hardware.tasks.openCommandObserver(),
    onChange: ({ task, type }) => {
      if (task !== key) return;
      if (shouldExecuteCommand(status, type)) setState(LOADING_STATE);
    },
  });
  Observe.useListener({
    key: [client?.key, key],
    open: async () => client?.hardware.tasks.openStateObserver(),
    onChange: (state) => {
      if (state.task !== key) return;
      const { details, variant } = state as task.State<D>;
      setState({
        status: details?.running ? RUNNING_STATUS : PAUSED_STATUS,
        message: details?.message,
        variant: parseVariant(variant),
      });
    },
  });
  const triggerLoading = useCallback(() => setState(LOADING_STATE), []);
  return useMemo<UseStateReturn>(
    () => [state, triggerLoading],
    [state, triggerLoading],
  );
};

const parseVariant = (variant?: string): PStatus.Variant | undefined =>
  PStatus.variantZ.safeParse(variant).data ?? undefined;

export const LOADING_STATE: State = { status: LOADING_STATUS, variant: "loading" };
