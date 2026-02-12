// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";

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
  const task = await client.tasks.retrieve({ key: keyToFetch });
  return {
    // type assertion okay for the moment because all of our current configs are object
    // types. We will want to get rid of this once we refactor to a more general
    // exporter and tighten up export / import logic more.
    data: JSON.stringify({ ...(task.config as record.Unknown), type: task.type }),
    name: task.name,
  };
};

export const useExport = () => Export.use(extract, "task");
