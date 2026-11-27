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
import { uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import { Tree } from "@/platform/tree";
import {
  createHostItems,
  createResource,
  findTreeRow,
  HOST_ID,
  openTreeRowContextMenu,
  renderLiveTree,
} from "@/platform/tree/testutil";

const client = createTestClient();

describe("LabJack device tree item", async () => {
  const rack = await client.racks.create({ name: uniqueName("labjack_rack") });
  const renderDeviceTree = async (identifier = ""): Promise<string> => {
    const name = uniqueName("labjack_dev");
    await client.labjack.devices.create({
      rack: rack.key,
      name,
      serial: Math.floor(Math.random() * 4e9),
      connection: "LJM_ctUSB",
      model: "t7",
      ports: { model: "t7" },
      identifier,
    });
    await renderLiveTree({
      client,
      root: HOST_ID,
      items: {
        ...createHostItems((c) => [
          Tree.createSource(
            c.labjack.devices,
            { where: { rack: { eq: rack.key } } },
            Tree.toResource("labjack_device"),
          ),
        ]),
        ...LabJack.Device.TREE_ITEMS,
      },
    });
    return name;
  };

  it("should list the device on the rack", async () => {
    const name = await renderDeviceTree();
    expect(await findTreeRow(name)).not.toBeNull();
  });

  it("should show a rename from another client", async () => {
    const name = await renderDeviceTree();
    await findTreeRow(name);
    const [dev] = await client.labjack.devices.retrieve({
      where: { name: { eq: name } },
    });
    const renamed = uniqueName("renamed");
    await createTestClient().labjack.devices.dispatch(
      dev.key,
      labjack.device.set({ path: ["name"], value: renamed }),
    );
    await waitFor(async () => expect(await findTreeRow(renamed)).not.toBeNull());
  });

  it("should open the typed configure form from the context menu", async () => {
    const name = await renderDeviceTree();
    await openTreeRowContextMenu(name);
    expect(screen.queryByText("Change identifier")).toBeNull();
    fireEvent.click(await screen.findByText("Configure"));
    expect(await screen.findByDisplayValue(name)).toBeTruthy();
  });

  it("should offer to change the identifier of a configured device", async () => {
    const name = await renderDeviceTree("dev1");
    await openTreeRowContextMenu(name);
    await screen.findByText("Change identifier");
    expect(screen.queryByText("Configure")).toBeNull();
  });

  it("should preview a dragged device with its type name", () => {
    const key = uuid.create();
    render(
      LabJack.Device.TREE_ITEMS.labjack_device.haulPreview({
        resource: createResource(labjack.device.resourceID(key), "Device A"),
        client,
        count: 1,
        resolveColor: () => undefined,
      }),
    );
    expect(screen.getByText("Device A")).toBeTruthy();
    expect(screen.getByText("LabJack device")).toBeTruthy();
  });
});
