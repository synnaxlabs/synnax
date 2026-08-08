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
  onStartStop: () => void;
  taskStatus: status.Status;
}

const notDeployedYet = (name: string) =>
  status.create({ name, variant: "disabled", message: "Not deployed yet" });

export const useTask = (key: arc.Key, name: string): UseTaskReturn => {
  const { data: tsk } = Arc.useResultTask({ arcKey: key });
  const cmd = Task.useCommand();
  const isRunning = tsk?.status?.details.running ?? false;
  const taskKey = tsk?.key;
  const handleStartStop = useCallback(() => {
    if (taskKey == null) return;
    cmd.update([{ task: taskKey, type: isRunning ? "stop" : "start" }]);
  }, [cmd, taskKey, isRunning]);
  if (tsk == null)
    return {
      running: false,
      taskKey: "",
      taskRack: 0,
      onStartStop: handleStartStop,
      taskStatus: notDeployedYet(name),
    };
  return {
    running: isRunning,
    taskKey: tsk.key,
    taskRack: tsk.rack,
    onStartStop: handleStartStop,
    taskStatus: tsk.status ?? notDeployedYet(name),
  };
};
