// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { answerCommandsUntil, reportTaskRunning } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const createRunningTask = async () => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  const tsk = await rack.createTask({
    name: uniqueName("tsk"),
    type: "opc_read",
    config: {},
  });
  await reportTaskRunning(client, tsk);
  return tsk;
};

describe("whileStopped", () => {
  it("should run the action between the stop and the start of the tasks", async () => {
    const tsk = await createRunningTask();
    const driver = await answerCommandsUntil(client, [tsk], "start");
    const result = await Task.whileStopped({ tasks: [tsk] }, async () => {
      await tsk.executeCommandSync({ type: "action" });
      return "done";
    });
    expect(result).toBe("done");
    const commands = await driver.answered;
    expect(commands.map(({ type }) => type)).toEqual(["stop", "action", "start"]);
  });

  it("should start the tasks again and throw when the action fails", async () => {
    const tsk = await createRunningTask();
    const driver = await answerCommandsUntil(client, [tsk], "start");
    await expect(
      Task.whileStopped({ tasks: [tsk], keepStopped: true }, async () => {
        throw new Error("action failed");
      }),
    ).rejects.toThrow("action failed");
    const commands = await driver.answered;
    expect(commands.map(({ type }) => type)).toEqual(["stop", "start"]);
  });
});
