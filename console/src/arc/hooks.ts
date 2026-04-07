// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, type task, UnexpectedError } from "@synnaxlabs/client";
import { Arc, type Flux, type List, Task } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { Modals } from "@/modals";

export interface UseTaskReturn {
  running: boolean;
  taskKey: task.Key;
  onStartStop: () => void;
  taskStatus: status.Status;
}

const notDeployedYet = (name: string) =>
  status.create({ name, variant: "disabled", message: "Not deployed yet" });

export const useRename = (
  getItem: List.GetItem<arc.Key, arc.Arc>,
): { update: (params: Arc.RenameParams) => void } => {
  const dispatch = useDispatch();
  const confirm = Modals.useConfirm();
  const { update } = Arc.useRename({
    beforeUpdate: useCallback(
      async ({
        data,
        rollbacks,
        store,
        client,
      }: Flux.BeforeUpdateParams<Arc.RenameParams, false, Arc.FluxSubStore>) => {
        const { key, name } = data;
        const tsk = await Arc.retrieveTask({
          store,
          client,
          query: { arcKey: key },
        });
        const a = getItem(key);
        if (a == null) throw new UnexpectedError(`Arc with key ${key} not found`);
        const oldName = a.name;
        if (tsk?.status?.details.running === true) {
          const confirmed = await confirm({
            message: `Are you sure you want to rename ${a.name} to ${name}?`,
            description: `This will cause ${a.name} to stop and be reconfigured.`,
            cancel: { label: "Cancel" },
            confirm: { label: "Rename", variant: "error" },
          });
          if (!confirmed) return false;
        }
        dispatch(Layout.rename({ key, name }));
        rollbacks.push(() => dispatch(Layout.rename({ key, name: oldName })));
        return data;
      },
      [dispatch, getItem, confirm],
    ),
  });
  return { update };
};

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
