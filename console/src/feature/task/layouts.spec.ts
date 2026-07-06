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

describe("createLayout", () => {
  it("should build a layout carrying the task key from the vendor registry", async () => {
    const task = await createTask(NI.Task.ANALOG_READ_TYPE);
    const layout = Task.createLayout(task);
    expect(layout.key).toBe(task.key);
    expect(layout.name).toBe(task.name);
    expect(layout.type).toBe(NI.Task.ANALOG_READ_TYPE);
    expect(layout.args).toEqual({ taskKey: task.key });
  });

  it("should throw for a task type with no registered layout", async () => {
    const task = await createTask("unregistered_type");
    expect(() => Task.createLayout(task)).toThrow(
      "No layout configured for unregistered_type",
    );
  });
});

describe("retrieveAndPlaceLayout", () => {
  it("should throw a DisconnectedError when the client is null", async () => {
    await expect(Task.retrieveAndPlaceLayout(null, "1", vi.fn())).rejects.toSatisfy(
      (e) => DisconnectedError.matches(e),
    );
  });

  it("should mark the placed layout tab as non-editable for snapshots", async () => {
    const task = await createTask(NI.Task.ANALOG_READ_TYPE);
    const snapshot = await client.tasks.copy(task.key, `${task.name} (Snapshot)`, true);
    const placeLayout = vi.fn();
    await Task.retrieveAndPlaceLayout(client, snapshot.key, placeLayout);
    expect(placeLayout).toHaveBeenCalledTimes(1);
    expect(placeLayout.mock.calls[0][0].tab?.editable).toBe(false);
  });

  it("should place an editable layout for a live task", async () => {
    const task = await createTask(NI.Task.ANALOG_READ_TYPE);
    const placeLayout = vi.fn();
    await Task.retrieveAndPlaceLayout(client, task.key, placeLayout);
    expect(placeLayout).toHaveBeenCalledTimes(1);
    const placed = placeLayout.mock.calls[0][0];
    expect(placed.tab?.editable).toBeUndefined();
    expect(placed.name).toBe(task.name);
  });
});
