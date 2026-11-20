// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type Synnax, task } from "@synnaxlabs/client";
import { errors } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { onTestFinished } from "vitest";
import { z } from "zod";

import { createRunningRack } from "@/platform/device/testutil";
import type { Listing } from "@/platform/file-browser/list";
import { answerCommand } from "@/platform/task/testutil";

/** A running rack and its file browser task. */
export interface FileRack {
  rack: rack.Rack;
  browser: task.Task;
}

/** Creates a running rack with a file browser task, as a Driver with `file` has. */
export const createFileRack = async (client: Synnax): Promise<FileRack> => {
  const r = await createRunningRack(client, "file");
  const browser = await r.createTask({
    name: uniqueName("file_browser"),
    type: "file_browser",
    config: {},
  });
  return { rack: r, browser };
};

const listArgsZ = z.object({ path: z.string() });

/** Creates the listing of a folder, with defaults for the fields a spec ignores. */
export const createListing = (overrides: Partial<Listing> = {}): Listing => ({
  path: "",
  parent: "",
  secretsDir: "",
  secret: false,
  truncated: false,
  entries: [],
  ...overrides,
});

/**
 * Answers each list command to the browser as a Driver does, with the listing of its
 * path. A path with no listing gets the error of a missing folder. Answers until the
 * test ends.
 */
export const serveListings = async (
  client: Synnax,
  { browser }: FileRack,
  listings: Listing[],
): Promise<void> => {
  const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
  let finished = false;
  onTestFinished(() => {
    finished = true;
    streamer.close();
  });
  const respond = async (): Promise<void> => {
    for (;;) {
      const frame = await streamer.read();
      for (const sample of frame.get(task.COMMAND_CHANNEL_NAME)) {
        const cmd = task.commandZ.parse(sample);
        if (cmd.task !== browser.key || cmd.type !== "list") continue;
        const { path } = listArgsZ.parse(cmd.args);
        const listing = listings.find((l) => l.path === path);
        await answerCommand(
          client,
          browser,
          cmd,
          listing == null
            ? {
                variant: "error",
                message: "Failed to list files",
                description: `${path} does not exist.`,
              }
            : { data: listing },
        );
      }
    }
  };
  // The read fails once the streamer closes at the end of the test.
  respond().catch((e: unknown) => {
    if (!finished) throw errors.fromUnknown(e);
  });
};
