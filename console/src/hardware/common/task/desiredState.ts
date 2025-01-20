// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Observe, Synnax } from "@synnaxlabs/pluto";
import { type Dispatch, type SetStateAction, useState } from "react";

export type DesiredState = "running" | "paused" | null;

export const useDesiredState = (
  initialState: DesiredState = null,
  taskKey?: task.Key,
): [DesiredState, Dispatch<SetStateAction<DesiredState>>] => {
  const [state, setState] = useState<DesiredState>(initialState);
  const client = Synnax.use();
  Observe.useListener({
    key: [client?.key, setState, taskKey],
    open: async () => client?.hardware.tasks.openStateObserver(),
    onChange: (state) => {
      if (state.task !== taskKey) return;
      const nowRunning = state.details?.running;
      const newState =
        nowRunning === true ? "running" : nowRunning === false ? "paused" : null;
      setState(newState);
    },
  });
  Observe.useListener({
    key: [client?.key, taskKey, setState],
    open: async () => client?.hardware.tasks.openCommandObserver(),
    onChange: (command) => {
      if (command.task !== taskKey) return;
      const type = command.type;
      if (type !== "start" && type !== "stop") return;
      const newState = type === "start" ? "running" : "paused";
      setState(newState);
    },
  });
  return [state, setState];
};

export const checkDesiredStateMatch = (
  desiredState: DesiredState,
  running: boolean | undefined,
): boolean => {
  if (desiredState == null) return true;
  if (desiredState === "running") return running === true;
  return running === false;
};
