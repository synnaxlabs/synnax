import { type arc, status, type task } from "@synnaxlabs/client";
import { Arc, Task } from "@synnaxlabs/pluto";
import { useCallback } from "react";

export interface UseTaskReturn {
  running: boolean;
  taskKey: task.Key;
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
      onStartStop: handleStartStop,
      taskStatus: tsk.status,
    };
  if (tsk.data == null)
    return {
      running: false,
      taskKey: "",
      onStartStop: () => {},
      taskStatus: notDeployedYet(name),
    };
  return {
    running: isRunning,
    taskKey: tsk.data.key,
    onStartStop: handleStartStop,
    taskStatus: tsk.data.status ?? notDeployedYet(name),
  };
};
