// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { createIdentifier, createSlaveDevice } from "@/feature/ethercat/testutil";
import { createRunningRack, renderWithDeviceForm } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { findDialogTrigger } from "@/testutil";

const client = createTestClient();

describe("EtherCAT.Device.SelectSlave", () => {
  it("should open the find view from the footer", async () => {
    await renderWithDeviceForm(
      <>
        <EtherCAT.Device.SelectSlave path="config.device" />
        <Modals.Stack />
      </>,
      { deviceKey: "", client },
    );
    fireEvent.click(await findDialogTrigger());
    fireEvent.click(await screen.findByText("Connect EtherCAT network"));
    expect(await screen.findByText("Not seeing your hardware?")).toBeTruthy();
  });

  it("should find only slaves on the network of the task's channels", async () => {
    const rack = await createRunningRack(client, EtherCAT.Device.HARDWARE.key);
    const create = async (network: string, configured: boolean) =>
      await createSlaveDevice(
        client,
        rack.key,
        { identifier: createIdentifier(), network },
        configured,
      );
    const inTask = await create("eth0", true);
    const sameNetwork = await create("eth0", false);
    const otherNetwork = await create("eth1", false);
    await renderWithDeviceForm(
      <>
        <EtherCAT.Device.SelectSlave path="config.device" />
        <Modals.Stack />
      </>,
      { deviceKey: "", channels: [{ device: inTask.key }], client },
    );
    fireEvent.click(await findDialogTrigger());
    fireEvent.click(await screen.findByText("Connect EtherCAT network"));
    await screen.findByText(sameNetwork.name);
    expect(screen.queryByText(otherNetwork.name)).toBeNull();
  });
});
