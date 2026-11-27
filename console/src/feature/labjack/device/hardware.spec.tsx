// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { labjack } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { LabJack } from "@/feature/labjack";
import { Device } from "@/platform/device";
import { renderMenuItem } from "@/platform/device/testutil";
import { createResource, createSelection } from "@/platform/tree/testutil";

const client = createTestClient();

const ConfigureMenuItem = Device.createConfigureMenuItem(LabJack.Device.TYPED_ADAPTER);
const ChangeIdentifierMenuItem = Device.createChangeIdentifierMenuItem({
  adapter: LabJack.Device.TYPED_ADAPTER,
  icon: "Logo.LabJack",
});

const createDevice = async (identifier = ""): Promise<labjack.device.Device> => {
  const rack = await client.racks.create({ name: uniqueName("labjack_rack") });
  return await client.labjack.devices.create({
    rack: rack.key,
    name: uniqueName("labjack_dev"),
    serial: Math.floor(Math.random() * 400_000) * 10_000 + 2,
    connection: "LJM_ctUSB",
    model: "t7",
    ports: { model: "t7" },
    identifier,
  });
};

const renderItems = async (identifier: string): Promise<void> => {
  const dev = await createDevice(identifier);
  const selection = createSelection({
    ids: [createResource(labjack.device.resourceID(dev.key), dev.name).id],
  });
  await renderMenuItem(
    <>
      <ConfigureMenuItem onConfigure={vi.fn()} selection={selection} />
      <ChangeIdentifierMenuItem selection={selection} handleError={() => {}} />
    </>,
    { client },
  );
};

describe("TYPED_ADAPTER", () => {
  it("should offer to configure a device without an identifier", async () => {
    await renderItems("");
    await screen.findByText("Configure");
    expect(screen.queryByText("Change identifier")).toBeNull();
  });

  it("should offer to change the identifier of a configured device", async () => {
    await renderItems("dev1");
    await screen.findByText("Change identifier");
    expect(screen.queryByText("Configure")).toBeNull();
  });

  it("should suggest identifiers from the model and serial", async () => {
    const dev = await createDevice();
    const Suggestions = () => (
      <span>
        {LabJack.Device.TYPED_ADAPTER.use(dev.key).suggestIdentifiers("").join(",")}
      </span>
    );
    await renderMenuItem(<Suggestions />, { client });
    await screen.findByText("t7,t7_0002");
  });

  it("should save the name and identifier and keep the ports", async () => {
    const dev = await createDevice();
    let save: ReturnType<typeof LabJack.Device.TYPED_ADAPTER.useSave> | undefined;
    const Save = () => {
      save = LabJack.Device.TYPED_ADAPTER.useSave();
      return null;
    };
    await renderMenuItem(<Save />, { client });
    await act(async () => {
      await save?.updateAsync({ key: dev.key, name: "Bench T7", identifier: "bench" });
    });
    await waitFor(async () => {
      const got = await client.labjack.devices.retrieve({ key: dev.key });
      expect(got.name).toBe("Bench T7");
      expect(got.identifier).toBe("bench");
      expect(got.ports).toEqual(dev.ports);
    });
  });
});
