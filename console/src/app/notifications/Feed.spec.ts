// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, rack, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { TimeSpan } from "@synnaxlabs/x";
import { act, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderForward } from "@/app/notifications/testutil";
import { EtherCAT } from "@/feature/ethercat";
import { createTestDevice } from "@/platform/device/testutil";
import {
  close,
  createRackKey,
  settleWrites,
  writeStatus,
} from "@/platform/notifications/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const createScanner = async (deviceConfigured: boolean): Promise<task.Task> => {
  const r = await client.racks.create({ name: uniqueName("rack") });
  if (deviceConfigured)
    await createTestDevice(client, { make: EtherCAT.Device.MAKE, rack: r.key });
  return await r.createTask({
    name: uniqueName("scanner"),
    type: EtherCAT.Task.SCAN_TYPE,
    config: {},
    internal: true,
  });
};

describe("Notifications.Provider", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should notify a scanner error only with a configured device", async () => {
    const unused = await createScanner(false);
    const used = await createScanner(true);
    await renderForward(client);
    const unusedMessage = await writeStatus(
      client,
      task.statusKey(unused.key),
      "error",
    );
    const usedMessage = await writeStatus(client, task.statusKey(used.key), "error");
    await screen.findByText(usedMessage);
    expect(screen.queryByText(unusedMessage)).toBeNull();
  });

  it("should notify a device warning only after 5 s", async () => {
    vi.useFakeTimers({
      toFake: ["setTimeout", "clearTimeout"],
      shouldAdvanceTime: true,
    });
    const dev = await createTestDevice(client);
    // The Core opens a "Device status unknown" problem on each new device.
    await writeStatus(client, device.statusKey(dev.key), "success");
    await renderForward(client);
    const message = await writeStatus(client, device.statusKey(dev.key), "warning");
    await settleWrites(client);
    expect(screen.queryByText(message)).toBeNull();
    act(() => void vi.advanceTimersByTime(TimeSpan.seconds(5).milliseconds));
    expect(await screen.findByText(message)).toBeTruthy();
  });

  it("should not notify a task problem inside the problem of its rack", async () => {
    const r = await client.racks.create({ name: uniqueName("rack") });
    const createUserTask = async (): Promise<task.Task> =>
      await r.createTask({
        name: uniqueName("task"),
        type: "pagerduty_alert",
        config: {},
      });
    const [onDown, onUp] = await Promise.all([createUserTask(), createUserTask()]);
    const [down, up] = await Promise.all([
      createRackKey(client),
      createRackKey(client),
    ]);
    await writeStatus(client, rack.statusKey(down), "warning");
    await renderForward(client);
    const writeTaskError = async (tsk: task.Task, rackKey: rack.Key): Promise<string> =>
      await writeStatus(client, task.statusKey(tsk.key), "error", {
        task: tsk.key,
        running: false,
        rack: rackKey,
      });
    const covered = await writeTaskError(onDown, down);
    const shown = await writeTaskError(onUp, up);
    await screen.findByText(shown);
    expect(screen.queryByText(covered)).toBeNull();
  });

  it("should notify each failed command, even after a close", async () => {
    const r = await client.racks.create({ name: uniqueName("rack") });
    const tsk = await r.createTask({
      name: uniqueName("task"),
      type: "pagerduty_alert",
      config: {},
    });
    await renderForward(client);
    const failStart = async (cmd: string): Promise<string> =>
      await writeStatus(client, task.statusKey(tsk.key), "error", {
        task: tsk.key,
        running: false,
        cmd,
      });
    await close(await failStart("start-1"));
    // The second reply comes within 30 s, so it has the since of the first.
    expect(await screen.findByText(await failStart("start-2"))).toBeTruthy();
  });
});
