// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, task } from "@synnaxlabs/client";
import { createTestClient, waitForStreamLive } from "@synnaxlabs/client/testutil";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Notifications } from "@/app/notifications";
import { EtherCAT } from "@/feature/ethercat";
import { HTTP } from "@/feature/http";
import { Kafka } from "@/feature/kafka";
import { MQTT } from "@/feature/mqtt";
import { Sift } from "@/feature/sift";
import { createTestDevice } from "@/platform/device/testutil";
import { createFileRack } from "@/platform/file-browser/testutil";
import {
  settleWrites,
  UncappedFeed,
  writeStatus,
} from "@/platform/notifications/testutil";
import { createTaskStatus } from "@/platform/task/testutil";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const Forward = (): ReactElement => {
  Notifications.useForward();
  return <UncappedFeed />;
};

const renderForward = async (): Promise<void> => {
  await waitForStreamLive(client.connection);
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Forward />, { wrapper });
};

const reply = async (tsk: task.Task): Promise<string> => {
  const { message } = await client.statuses.set(
    createTaskStatus({
      variant: "error",
      message: uniqueName("reply"),
      owner: task.resourceID(tsk.key),
      details: { cmd: uniqueName("cmd"), rack: tsk.rack },
    }),
  );
  return message;
};

const createScanner = async (deviceConfigured: boolean): Promise<task.Task> => {
  const r = await client.racks.create({ name: uniqueName("rack") });
  if (deviceConfigured)
    await createTestDevice(client, { make: EtherCAT.Device.MAKE, rack: r.key });
  return await r.createTask({
    name: uniqueName("scanner"),
    type: EtherCAT.Device.HARDWARE.scanType,
    config: {},
    internal: true,
  });
};

describe("Notifications.useForward", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should notify a scanner error only with a configured device", async () => {
    const unused = await createScanner(false);
    const used = await createScanner(true);
    await renderForward();
    const unusedMessage = await writeStatus(
      client,
      task.resourceID(unused.key),
      "error",
    );
    const usedMessage = await writeStatus(client, task.resourceID(used.key), "error");
    await screen.findByText(usedMessage);
    expect(screen.queryByText(unusedMessage)).toBeNull();
  });

  it("should not notify a reply of the file browser", async () => {
    const { rack, browser } = await createFileRack(client);
    const other = await rack.createTask({
      name: uniqueName("task"),
      type: EtherCAT.Device.HARDWARE.scanType,
      config: {},
    });
    await renderForward();
    const browserMessage = await reply(browser);
    await screen.findByText(await reply(other));
    expect(screen.queryByText(browserMessage)).toBeNull();
  });

  // Each dialog that sends a command to these scanners shows the reply in place.
  it.each([
    ["HTTP", HTTP.Device.MAKE, HTTP.Device.SCAN_TYPE],
    ["Kafka", Kafka.Device.MAKE, Kafka.Device.SCAN_TYPE],
    ["MQTT", MQTT.Device.MAKE, MQTT.Device.SCAN_TYPE],
  ])("should not notify a reply of the %s scanner", async (_, make, scanType) => {
    const r = await client.racks.create({ name: uniqueName("rack") });
    await createTestDevice(client, { make, rack: r.key });
    const create = async (type: string, internal: boolean): Promise<task.Task> =>
      await r.createTask({ name: uniqueName("task"), type, config: {}, internal });
    const scanner = await create(scanType, true);
    const other = await create("opc_read", false);
    await renderForward();
    const scannerMessage = await reply(scanner);
    await screen.findByText(await reply(other));
    expect(screen.queryByText(scannerMessage)).toBeNull();
  });

  it("should notify a device warning only after 5 s", async () => {
    vi.useFakeTimers({
      toFake: ["setTimeout", "clearTimeout"],
      shouldAdvanceTime: true,
    });
    const dev = await createTestDevice(client);
    await renderForward();
    const message = await writeStatus(client, device.resourceID(dev.key), "warning");
    await settleWrites(client);
    expect(screen.queryByText(message)).toBeNull();
    act(() => void vi.advanceTimersByTime(TimeSpan.seconds(5).milliseconds));
    expect(await screen.findByText(message)).toBeTruthy();
  });

  it("should notify the Sift stream progress only on an error", async () => {
    await renderForward();
    const key = `${Sift.Task.PROGRESS_KEY_PREFIX}${uniqueName("task")}`;
    const write = async (variant: "info" | "error"): Promise<string> => {
      const { message } = await client.statuses.set({
        key,
        name: "",
        variant,
        message: uniqueName("progress"),
        time: TimeStamp.now(),
      });
      return message;
    };
    const progress = await write("info");
    await settleWrites(client);
    expect(screen.queryByText(progress)).toBeNull();
    expect(await screen.findByText(await write("error"))).toBeTruthy();
  });

  it("should notify a Sift upload only on an error", async () => {
    await renderForward();
    const key = `${Sift.Range.KEY_PREFIX}${uniqueName("upload")}`;
    const write = async (variant: "success" | "error"): Promise<string> => {
      const { message } = await client.statuses.set({
        key,
        name: "",
        variant,
        message: uniqueName("upload"),
        time: TimeStamp.now(),
      });
      return message;
    };
    const success = await write("success");
    await settleWrites(client);
    expect(screen.queryByText(success)).toBeNull();
    expect(await screen.findByText(await write("error"))).toBeTruthy();
  });

  it("should notify a Sift retry warning only after 5 s", async () => {
    vi.useFakeTimers({
      toFake: ["setTimeout", "clearTimeout"],
      shouldAdvanceTime: true,
    });
    await renderForward();
    const { message } = await client.statuses.set({
      key: `${Sift.Task.PROGRESS_KEY_PREFIX}${uniqueName("task")}`,
      name: "",
      variant: "warning",
      message: uniqueName("retrying"),
      time: TimeStamp.now(),
    });
    await settleWrites(client);
    expect(screen.queryByText(message)).toBeNull();
    act(() => void vi.advanceTimersByTime(TimeSpan.seconds(5).milliseconds));
    expect(await screen.findByText(message)).toBeTruthy();
  });
});
