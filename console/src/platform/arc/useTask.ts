// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, type rack, status, type task } from "@synnaxlabs/client";
import { Arc, Task } from "@synnaxlabs/pluto";
import { useCallback } from "react";

export interface UseTaskReturn {
  running: boolean;
  taskKey: task.Key;
  taskRack: rack.Key;
  /** Starts the task. The driver rebuilds from the current config when it drifted. */
  onStart: () => void;
  /** Stops the running instance. */
  onStop: () => void;
  onStartStop: () => void;
  taskStatus: status.Status;
}

const notDeployedYet = (name: string) =>
  status.create({ name, variant: "disabled", message: "Not deployed yet" });

export const useTask = (key: arc.Key, name: string): UseTaskReturn => {
  const { data: tsk, variant, status: readStatus } = Arc.useResultTask({ arcKey: key });
  const cmd = Task.useCommand();
  const isRunning = tsk?.status?.details.running ?? false;
  const taskKey = tsk?.key ?? "";
  const taskRack = tsk?.rack ?? 0;
  const exec = useCallback(
    (type: "start" | "stop") => {
      if (taskKey === "") return;
      cmd.update([{ task: taskKey, type }]);
    },
    [cmd, taskKey],
  );
  const onStart = useCallback(() => exec("start"), [exec]);
  const onStop = useCallback(() => exec("stop"), [exec]);
  const onStartStop = useCallback(
    () => (isRunning ? onStop() : onStart()),
    [isRunning, onStop, onStart],
  );
  return {
    running: isRunning,
    taskKey,
    taskRack,
    onStart,
    onStop,
    onStartStop,
    taskStatus:
      variant !== "success" ? readStatus : (tsk?.status ?? notDeployedYet(name)),
  };
};
