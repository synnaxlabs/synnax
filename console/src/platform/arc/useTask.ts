// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type arc,
  DisconnectedError,
  type rack,
  status,
  type task,
} from "@synnaxlabs/client";
import { Arc, Status, Synnax, Task } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useCallback } from "react";

export interface UseTaskReturn {
  running: boolean;
  taskKey: task.Key;
  taskRack: rack.Key;
  /** Redeploys the arc to its rack, restamping the hash, then starts it. */
  onDeploy: () => void;
  /** Stops the running instance. */
  onStop: () => void;
  onStartStop: () => void;
  taskStatus: status.Status;
}

const notDeployedYet = (name: string) =>
  status.create({ name, variant: "disabled", message: "Not deployed yet" });

export const useTask = (key: arc.Key, name: string): UseTaskReturn => {
  const tsk = Arc.useRetrieveTask({ arcKey: key });
  const cmd = Task.useCommand();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const isRunning = tsk.data?.status?.details.running ?? false;
  const resolved = tsk.variant === "success" ? tsk.data : null;
  const taskKey = resolved?.key ?? "";
  const taskRack = resolved?.rack ?? 0;
  const onDeploy = useCallback(() => {
    if (!primitive.isNonZero(taskRack)) return;
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const deployed = await client.arcs.deploy(key, taskRack);
      if (deployed == null) return;
      await client.tasks.executeCommand({ task: deployed.key, type: "start" });
    }, `Failed to deploy ${name}`);
  }, [client, key, taskRack, handleError, name]);
  const onStop = useCallback(() => {
    if (taskKey === "") return;
    cmd.update([{ task: taskKey, type: "stop" }]);
  }, [cmd, taskKey]);
  const onStartStop = useCallback(
    () => (isRunning ? onStop() : onDeploy()),
    [isRunning, onStop, onDeploy],
  );
  let taskStatus: status.Status;
  if (tsk.variant !== "success") taskStatus = tsk.status;
  else taskStatus = resolved?.status ?? notDeployedYet(name);
  return {
    running: isRunning,
    taskKey,
    taskRack,
    onDeploy,
    onStop,
    onStartStop,
    taskStatus,
  };
};
