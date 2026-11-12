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

import { EtherCAT } from "@/feature/ethercat";
import { Device } from "@/platform/device";
import { createRunningRack } from "@/platform/device/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { answerNextCommand, createTaskStatus } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const BLOCKED = "No EtherCAT modules found on Ethernet 2.";

/**
 * Opens the find view on a rack whose EtherCAT scanner warns that the DewesoftX filter
 * blocks an adapter, with the data of the warning.
 */
const renderFind = async (data: unknown): Promise<task.Task> => {
  const hardware = { ...EtherCAT.Device.HARDWARE, key: uniqueName("integration") };
  const rack = await createRunningRack(client, hardware.key);
  const scanner = await rack.createTask({
    name: uniqueName("scanner"),
    type: EtherCAT.Task.SCAN_TYPE,
    config: {},
    internal: true,
  });
  await client.statuses.set(
    createTaskStatus({
      owner: task.resourceID(scanner.key),
      variant: "warning",
      message: "Failed to scan for devices",
      description: BLOCKED,
      details: { running: true, rack: rack.key, data },
    }),
  );
  await renderModalOpener(Device.useFindModal, [{ hardware }], { client });
  await screen.findByText(BLOCKED);
  return scanner;
};

describe("EtherCAT scan action", () => {
  it("should turn off the DewesoftX filter once the user confirms", async () => {
    const scanner = await renderFind({ filtered: ["\\Device\\NPF_{0}"] });
    const driver = await answerNextCommand(client, scanner);
    fireEvent.click(findButton("Turn off the filter"));
    fireEvent.click(await screen.findByRole("button", { name: "Turn off" }));
    expect((await driver.answered).type).toBe("disable_dewesoft_filter");
  });

  it("should not offer to turn off the filter when it blocks no adapter", async () => {
    await renderFind(undefined);
    expect(screen.queryByText("Turn off the filter")).toBeNull();
  });
});
