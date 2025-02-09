// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Observe, type Status, Synnax } from "@synnaxlabs/pluto";
import {
  type Dispatch,
  type SetStateAction,
  useCallback,
  useState as useReactState,
} from "react";

export interface Details {
  running: boolean;
  message?: string;
}

export type State = "loading" | "running" | "paused";

export interface ReturnState {
  state: State;
  message?: string;
  variant?: Status.Variant;
}

export const useState = <D extends Details>(
  key: task.Key,
  initialState?: task.State<D>,
): [ReturnState, Dispatch<SetStateAction<State>>] => {
  // isRunning tracks if the task is actually running, based off of the state observer
  // on the driver.
  const [isRunning, setIsRunning] = useReactState(
    initialState?.details?.running ?? false,
  );
  const [state, setState] = useReactState<ReturnState>({
    state: isRunning ? "running" : "paused",
    message: initialState?.details?.message,
    variant: (initialState?.variant as Status.Variant) ?? undefined,
  });
  const client = Synnax.use();
  Observe.useListener({
    key: [client?.key, key, setState],
    open: async () => client?.hardware.tasks.openCommandObserver(),
    onChange: ({ task, type }) => {
      if (task !== key) return;
      if (type === (isRunning ? "stop" : "start"))
        setState((s) => ({ ...s, state: "loading" }));
    },
  });
  Observe.useListener({
    key: [client?.key, setIsRunning, setState, key],
    open: async () => client?.hardware.tasks.openStateObserver(),
    onChange: (state) => {
      if (state.task !== key) return;
      const { details, variant } = state as task.State<D>;
      const nowRunning = details?.running ?? false;
      setIsRunning(nowRunning);
      setState({
        state: nowRunning ? "running" : "paused",
        message: details?.message,
        variant: (variant as Status.Variant) ?? undefined,
      });
    },
  });
  const setDesiredState: Dispatch<SetStateAction<State>> = useCallback(
    (s) =>
      setState(({ state, ...rest }) => ({
        ...rest,
        state: typeof s === "string" ? s : s(state),
      })),
    [setState],
  );
  return [state, setDesiredState];
};
