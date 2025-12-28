// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, type rack, type Synnax, type task } from "@synnaxlabs/client";
import { Task } from "@synnaxlabs/pluto";
import { useCallback, useMemo } from "react";

/** The task type identifier for Arc tasks. */
export const TASK_TYPE = "arc";

/** Configuration schema for Arc tasks. */
export interface TaskConfig {
  arc_key: arc.Key;
  auto_start?: boolean;
}

const arcTaskFilter = (t: task.Task) => t.type === TASK_TYPE;

/**
 * Hook to find the task associated with an Arc program.
 * Since Arc programs have a 1:1 relationship with tasks, this returns
 * at most one task.
 */
export const useArcTask = (
  arcKey: arc.Key | null | undefined,
): task.Task | undefined => {
  const filter = useCallback(arcTaskFilter, []);
  const { data: tasks, getItem } = Task.useList({ filter });

  return useMemo(() => {
    if (arcKey == null || tasks == null) return undefined;
    const foundKey = tasks.find((key) => {
      const t = getItem(key);
      if (t == null) return false;
      try {
        const rawConfig = t.config as unknown;
        const config: TaskConfig =
          typeof rawConfig === "string"
            ? (JSON.parse(rawConfig) as TaskConfig)
            : (rawConfig as TaskConfig);
        return config.arc_key === arcKey;
      } catch {
        return false;
      }
    });
    return foundKey != null ? getItem(foundKey) : undefined;
  }, [arcKey, tasks, getItem]);
};

/**
 * Creates a new Arc task on the specified rack.
 *
 * @param client - Synnax client instance
 * @param arcKey - Key of the Arc program to execute
 * @param rackKey - Key of the rack to deploy to
 * @param name - Name for the task (defaults to Arc program name + " Task")
 * @returns The created task
 */
export const createArcTask = async (
  client: Synnax,
  arcKey: arc.Key,
  rackKey: rack.Key,
  name: string,
): Promise<task.Task> => {
  const rackInstance = await client.racks.retrieve({ key: rackKey });
  const config: TaskConfig = {
    arc_key: arcKey,
    auto_start: false,
  };
  return await rackInstance.createTask({
    name,
    type: TASK_TYPE,
    config,
  });
};

/**
 * Deletes an Arc task.
 *
 * @param client - Synnax client instance
 * @param taskKey - Key of the task to delete
 */
export const deleteArcTask = async (
  client: Synnax,
  taskKey: task.Key,
): Promise<void> => {
  await client.tasks.delete(taskKey);
};
