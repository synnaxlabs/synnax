// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import {
  caseconv,
  type CrudeTimeSpan,
  TimeSpan,
  type UnknownRecord,
} from "@synnaxlabs/x";
import { useEffect, useRef, useState } from "react";

import { Synch } from "@/synch";
import { Synnax } from "@/synnax";

interface CommandExecutorParams {
  args?: UnknownRecord;
  taskKey: task.Key;
  taskName?: string;
  timeout: CrudeTimeSpan;
  type: string;
}

interface CommandExecutor {
  (params: CommandExecutorParams): Promise<void>;
}

export interface UseCommandExecutorReturn {
  executeCommand: CommandExecutor;
  state: task.State | "loading" | null;
}

export const useCommandExecutor = (): UseCommandExecutorReturn => {
  const client = Synnax.use();
  const [state, setState] = useState<task.State | null | "loading">(null);
  const [key, setKey] = useState<string | null>(null);
  const timerRef = useRef<NodeJS.Timeout | null>(null);
  const handleExecuteCommand = async ({
    args,
    taskKey,
    taskName,
    timeout: crudeTimeout,
    type,
  }: CommandExecutorParams) => {
    const timeout = new TimeSpan(crudeTimeout);
    if (client == null) throw new Error("Client not found");
    setState("loading");
    const cmdKey = await client.hardware.tasks.executeCommand(taskKey, type, args);
    setKey(cmdKey);
    if (timerRef.current != null) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      setState({
        key: cmdKey,
        task: taskKey,
        variant: "error",
        details: {
          message: `${caseconv.capitalize(type)} command to ${
            taskName ?? `task with key ${taskKey}`
          } timed out after ${timeout.toString()}`,
        },
      });
      setKey(null);
    }, timeout.milliseconds);
  };
  const addListener = Synch.useAddListener();
  useEffect(() => {
    if (key == null) return;
    return addListener({
      channels: "sy_task_set",
      handler: (frame) => {
        frame
          .get("sy_task_set")
          .parseJSON(task.stateZ)
          .forEach((s) => {
            if (s.key === key) {
              if (timerRef.current != null) clearTimeout(timerRef.current);
              setState(s);
              setKey(null);
            }
          });
      },
    });
  }, [key]);

  useEffect(
    () => () => {
      if (timerRef.current != null) clearTimeout(timerRef.current);
    },
    [],
  );
  return { state, executeCommand: handleExecuteCommand };
};
