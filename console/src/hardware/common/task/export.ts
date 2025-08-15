// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";

import { Export } from "@/export";
import { Layout } from "@/layout";

export const extract: Export.Extractor = async (key, { client, store }) => {
  if (client == null) throw new DisconnectedError();
  let keyToFetch = key;
  try {
    BigInt(key);
  } catch {
    const layoutState = Layout.select(store.getState(), key);
    if (layoutState == null)
      throw new Error(
        `Cannot export task with key ${key}. This is neither the key of a task nor the key of a task layout.`,
      );
    const args = layoutState.args;
    if (
      typeof args !== "object" ||
      args == null ||
      !("taskKey" in args) ||
      typeof args.taskKey !== "string"
    )
      throw new Error(
        `Cannot export task with key ${key}. You should configure the task before exporting it.`,
      );
    keyToFetch = args.taskKey;
  }
  const task = await client.hardware.tasks.retrieve({ key: keyToFetch });
  return { data: JSON.stringify(task.config), name: task.name };
};

export const useExport = () => Export.use(extract, "task");
