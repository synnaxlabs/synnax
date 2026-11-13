// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { awaitCommand, renderInTaskFormWithClient } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();
const rack = await client.racks.create({ name: uniqueName("rack") });
const { key: FLOAT } = await client.channels.create({
  name: uniqueName("tare"),
  dataType: "float32",
  virtual: true,
});

const renderButton = async (taskKey: task.Key, running: boolean) =>
  await renderInTaskFormWithClient(<Task.TareButton channel={FLOAT} />, {
    client,
    values: { key: taskKey, status: { details: { running } } },
  });

describe("TareButton", () => {
  it("should send a tare command for the channel while the task runs", async () => {
    const tsk = await rack.createTask({
      name: uniqueName("tsk"),
      type: "opc_read",
      config: {},
    });
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    try {
      await renderButton(tsk.key, true);
      fireEvent.click(await screen.findByRole("button"));
      const cmd = await awaitCommand(streamer, tsk.key, "tare");
      expect(cmd.args).toEqual({ keys: [FLOAT] });
    } finally {
      streamer.close();
    }
  });

  it("should be disabled while the task is stopped", async () => {
    await renderButton("task-1", false);
    const button = await screen.findByRole("button");
    expect(button.getAttribute("aria-disabled")).toBe("true");
  });
});
