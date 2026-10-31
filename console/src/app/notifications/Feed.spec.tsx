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
import { TimeSpan } from "@synnaxlabs/x";
import { act, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Notifications } from "@/app/notifications";
import { EtherCAT } from "@/feature/ethercat";
import { createTestDevice } from "@/platform/device/testutil";
import {
  settleWrites,
  UncappedFeed,
  writeStatus,
} from "@/platform/notifications/testutil";
import { createConsoleWrapper, uniqueName } from "@/testutil";

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
    await renderForward();
    const message = await writeStatus(client, device.statusKey(dev.key), "warning");
    await settleWrites(client);
    expect(screen.queryByText(message)).toBeNull();
    act(() => void vi.advanceTimersByTime(TimeSpan.seconds(5).milliseconds));
    expect(await screen.findByText(message)).toBeTruthy();
  });
});
