// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { Flux, Task } from "@synnaxlabs/pluto";

export interface SetDataSavingParams {
  key: task.Key;
  dataSaving: boolean;
}

export const { useUpdate: useSetDataSaving } = Flux.createUpdate<
  SetDataSavingParams,
  Task.FluxSubStore
>({
  name: Task.RESOURCE_NAME,
  verbs: Flux.UPDATE_VERBS,
  update: async ({ client, data, store }) => {
    const { key, dataSaving } = data;
    const t = await Task.retrieveSingle({ client, store, query: { key } });
    const config = t.payload.config;
    // Only tasks with a dataSaving field in their config (primarily read tasks)
    // are eligible. Write tasks without this field are skipped.
    if (typeof config !== "object" || config == null || !("dataSaving" in config))
      return data;
    if ((config as Record<string, unknown>).dataSaving === dataSaving) return data;
    const wasRunning = t.status?.details.running === true;
    await client.tasks.create({ ...t.payload, config: { ...config, dataSaving } });
    if (wasRunning) await client.tasks.executeCommand({ task: key, type: "start" });
    return data;
  },
});
