// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Observe, type Status, Synnax } from "@synnaxlabs/pluto";
import { type Dispatch, type SetStateAction, useState } from "react";

export type StateDetails = { running: boolean; message?: string };

export type State = "loading" | "running" | "paused";

export interface ReturnState {
  state: State;
  message?: string;
  variant?: Status.Variant;
}

export const useDesiredState = <D extends StateDetails>(
  taskKey: task.Key,
  initialState?: task.State<D>,
): [ReturnState, Dispatch<SetStateAction<State>>] => {
  const [isRunning, setIsRunning] = useState(initialState?.details?.running ?? false);
  const [desiredState, setDesiredState] = useState<ReturnState>({
    state: isRunning ? "running" : "paused",
    message: initialState?.details?.message,
  });
  const client = Synnax.use();
  Observe.useListener({
    key: [client?.key, taskKey, setDesiredState],
    open: async () => client?.hardware.tasks.openCommandObserver(),
    onChange: (command) => {
      if (command.task !== taskKey) return;
      const type = command.type;
      if (type === "start" && !isRunning)
        setDesiredState((state) => ({ ...state, state: "loading" }));
      if (type === "stop" && isRunning)
        setDesiredState((state) => ({ ...state, state: "loading" }));
    },
  });
  Observe.useListener({
    key: [client?.key, setIsRunning, setDesiredState, taskKey],
    open: async () => client?.hardware.tasks.openStateObserver<D>(),
    onChange: (state) => {
      if (state.task !== taskKey) return;
      const nowRunning = state.details?.running ?? false;
      setIsRunning(nowRunning);
      setDesiredState({
        state: nowRunning ? "running" : "paused",
        message: state.details?.message,
        variant: (state.variant as Status.Variant) ?? undefined,
      });
    },
  });
  const setDesiredStateReturn: Dispatch<SetStateAction<State>> = (s) =>
    setDesiredState(({ state, ...rest }) => ({
      ...rest,
      state: typeof s === "string" ? s : s(state),
    }));
  return [desiredState, setDesiredStateReturn];
};
