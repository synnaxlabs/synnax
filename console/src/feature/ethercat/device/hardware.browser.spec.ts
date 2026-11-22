// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { device } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { createSlaveDevice } from "@/feature/ethercat/testutil";
import { Device } from "@/platform/device";
import { createRunningRack } from "@/platform/device/testutil";
import { renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

describe("EtherCAT.Device.HARDWARE", () => {
  const find = async (properties: Partial<EtherCAT.Device.SlaveProperties>) => {
    const { HARDWARE } = EtherCAT.Device;
    const rack = await createRunningRack(client, HARDWARE.key);
    const slave = await createSlaveDevice(client, rack.key, properties, false);
    const filter = (d: device.Device) => d.key === slave.key;
    await renderModalOpener(Device.useFindModal, [{ hardware: HARDWARE, filter }], {
      client,
    });
    await screen.findByText(slave.name);
  };

  it("should show the network name and slot of a found slave", async () => {
    await find({ networkName: "Ethernet 2", position: 3 });
    expect(screen.getByText("Ethernet 2, slot 3")).toBeTruthy();
  });

  it("should show the location of a slave from a Driver that names no network", async () => {
    await find({ position: 3 });
    expect(screen.getByText("soem:eth0.Slot 3")).toBeTruthy();
  });
});
