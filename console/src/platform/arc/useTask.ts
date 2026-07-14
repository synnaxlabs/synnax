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
      taskRack: 0,
      onStartStop: handleStartStop,
      taskStatus: tsk.status,
    };
  if (tsk.data == null)
    return {
      running: false,
      taskKey: "",
      taskRack: 0,
      onStartStop: () => {},
      taskStatus: notDeployedYet(name),
    };
  return {
    running: isRunning,
    taskKey: tsk.data.key,
    taskRack: tsk.data.rack,
    onStartStop: handleStartStop,
    taskStatus: tsk.data.status ?? notDeployedYet(name),
  };
};
