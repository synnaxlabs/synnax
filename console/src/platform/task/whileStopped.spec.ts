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

describe("whileStopped", () => {
  it("should run the action between the stop and the start of the tasks", async () => {
    const rack = await client.racks.create({ name: uniqueName("rack") });
    const tsk = await rack.createTask({
      name: uniqueName("tsk"),
      type: "opc_read",
      config: {},
    });
    await reportTaskRunning(client, tsk);
    const driver = await answerCommandsUntil(client, [tsk], "start");
    await Task.whileStopped({ tasks: [tsk] }, async () => {
      await tsk.executeCommandSync({ type: "action" });
    });
    const commands = await driver.answered;
    expect(commands.map(({ type }) => type)).toEqual(["stop", "action", "start"]);
  });
});
