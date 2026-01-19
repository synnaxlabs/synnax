// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, type task } from "@synnaxlabs/client";
import { Arc, Task } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useCallback } from "react";

export interface UseTaskReturn {
  running: boolean;
  taskKey: task.Key;
  onStartStop: () => void;
  taskStatus: status.Status;
}

const NO_TASK_FOR_ARC = status.create({
  key: "no-task-for-arc",
  name: "No task for arc",
  variant: "disabled",
  message: "No task for arc",
});

export const useTask = (key: arc.Key): UseTaskReturn => {
  const tsk = Arc.useRetrieveTask({ arcKey: key });
  const cmd = Task.useCommand();
  const isRunning = tsk.data?.status?.details.running ?? false;
  const handleStartStop = useCallback(() => {
    if (tsk.data?.key == null) return;
    cmd.update([{ task: tsk.data.key, type: isRunning ? "stop" : "start" }]);
  }, [cmd, tsk.data?.key, isRunning]);
  if (tsk.variant !== "success")
    return {
      running: isRunning,
      taskKey: "",
      onStartStop: handleStartStop,
      taskStatus: tsk.status,
    };
  if (tsk.data == null)
    return {
      running: false,
      taskKey: "",
      onStartStop: () => {},
      taskStatus: NO_TASK_FOR_ARC,
    };
  return {
    running: isRunning,
    taskKey: tsk.data.key,
    onStartStop: handleStartStop,
    taskStatus: tsk.data.status ?? NO_TASK_FOR_ARC,
  };
};
