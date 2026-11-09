// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, task } from "@synnaxlabs/client";
import { Flux, Task } from "@synnaxlabs/pluto";
import { verbs } from "@synnaxlabs/x";

export interface SetDataSavingParams {
  key: task.Key;
  dataSaving: boolean;
}

export const { useUpdate: useSetDataSaving } = Flux.createUpdate<SetDataSavingParams>({
  name: Task.RESOURCE_NAME,
  verbs: verbs.UPDATE,
  update: async ({ client, data }) => {
    const { key, dataSaving } = data;
    const t = await client.tasks.retrieve({ key, includeStatus: true });
    const config = t.payload.config;
    // Only tasks with a dataSavingDisabled field in their config (primarily read
    // tasks) are eligible. Tasks without this field are skipped.
    if (
      typeof config !== "object" ||
      config == null ||
      !("dataSavingDisabled" in config)
    )
      return data;
    const dataSavingDisabled = !dataSaving;
    if ((config as Record<string, unknown>).dataSavingDisabled === dataSavingDisabled)
      return data;
    const wasRunning = t.status?.details.running === true;
    const set = task.set({
      path: ["config", "dataSavingDisabled"],
      value: dataSavingDisabled,
    });
    if (!(await client.tasks.dispatch(key, set)))
      throw new NotFoundError(`Task ${key} is not cached`);
    if (!wasRunning) return data;
    // The start sends the cached config hash, which changes only on the Core's echo.
    await client.tasks.settled(key);
    await client.tasks.executeCommand({ task: key, type: "start" });
    return data;
  },
});
