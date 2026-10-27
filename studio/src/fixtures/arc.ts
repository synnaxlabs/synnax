// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, type Synnax, type task, TimeSpan } from "@synnaxlabs/client";

import { connect, type ConnectionOptions } from "@/fixtures/client";

/** How long an Arc task has to report that it started. */
const START_TIMEOUT = TimeSpan.seconds(10);

/**
 * startArc binds an Arc to the Core's Arc rack and starts its task, returning the task.
 * Throws when the Core has no Arc rack or the task reports an error on start.
 */
export const startArc = async (client: Synnax, key: arc.Key): Promise<task.Task> => {
  const [rack] = await client.racks.retrieve({ integration: "arc" });
  if (rack == null) throw new Error("the Core has no rack that runs Arc");
  const tsk = await client.arcs.updateTask(key, { rack: rack.key });
  if (tsk == null)
    throw new Error(`binding Arc ${key} to rack ${rack.name} made no task`);
  const started = await tsk.executeCommandSync({
    type: "start",
    timeout: START_TIMEOUT,
  });
  if (started.variant === "error")
    throw new Error(`Arc ${key} failed to start: ${started.message}`);
  return tsk;
};

export interface ArcOptions extends ConnectionOptions {
  /** Name of the Arc, as the Console and control legends show it. */
  name: string;
  /** Arc source code. Every channel it names must already exist. */
  text: string;
}

export interface ArcFixture {
  key: arc.Key;
  /** Stops the Arc's task and closes the client. */
  stop: () => Promise<void>;
}

/**
 * runArc creates a text-mode Arc and runs it on the Core, so a shot opens on an
 * automation that is already running.
 */
export const runArc = async ({
  name,
  text,
  ...opts
}: ArcOptions): Promise<ArcFixture> => {
  const client = connect(opts);
  const { key } = await client.arcs.create({ name, mode: "text", text: { raw: text } });
  const tsk = await startArc(client, key);
  return {
    key,
    stop: async () => {
      await tsk.stop();
      await client.close();
    },
  };
};
