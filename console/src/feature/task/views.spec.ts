// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, DisconnectedError } from "@synnaxlabs/client";
import { describe, expect, it, vi } from "vitest";

import { NI } from "@/feature/ni";
import { Task } from "@/feature/task";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const createTask = async (type: string) => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await rack.createTask({ name: uniqueName("task"), type, config: {} });
};

describe("retrieveAndOpenTab", () => {
  it("should throw a DisconnectedError when the client is null", async () => {
    await expect(Task.retrieveAndOpenTab(null, "1", vi.fn())).rejects.toSatisfy((e) =>
      DisconnectedError.matches(e),
    );
  });

  it("should open a view tab carrying the retrieved task's type and key", async () => {
    const task = await createTask(NI.Task.ANALOG_READ_TYPE);
    const openTab = vi.fn();
    await Task.retrieveAndOpenTab(client, task.key, openTab);
    expect(openTab).toHaveBeenCalledTimes(1);
    expect(openTab.mock.calls[0][0]).toEqual({
      variant: "view",
      type: NI.Task.ANALOG_READ_TYPE,
      args: { taskKey: task.key },
    });
  });
});
