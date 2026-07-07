// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Modbus } from "@/feature/modbus";
import { createModbusDevice } from "@/feature/modbus/testutil";
import { createDeviceResource, renderMenuItem } from "@/platform/device/testutil";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createTestStore, waitForPlacedLayout } from "@/testutil";

const client = createTestClient();

const renderItems = async () => {
  const dev = await createModbusDevice(client);
  const propsStore = await createTestStore();
  const resource = createDeviceResource({
    key: dev.key,
    name: dev.name,
    configured: true,
  });
  const handle = await renderMenuItem(
    <Modbus.Device.ContextMenuItems
      {...createBaseProps({ client, store: propsStore })}
      selection={createSelection({ ids: [resource.id] })}
      state={createState([resource])}
    />,
    { client },
  );
  return { ...handle, dev };
};

describe("Modbus.Device.ContextMenuItems", () => {
  it("should place the write task layout carrying the device key", async () => {
    const { store, dev } = await renderItems();
    fireEvent.click(await screen.findByText("Create write task"));
    const key = await waitForPlacedLayout(store, Modbus.Task.WRITE_TYPE);
    expect(Session.Layout.selectArgs(store.getState(), key)).toEqual({
      deviceKey: dev.key,
    });
  });

  it("should place the read task layout carrying the device key", async () => {
    const { store, dev } = await renderItems();
    fireEvent.click(await screen.findByText("Create read task"));
    const key = await waitForPlacedLayout(store, Modbus.Task.READ_TYPE);
    expect(Session.Layout.selectArgs(store.getState(), key)).toEqual({
      deviceKey: dev.key,
    });
  });

  it("should open the connect modal from the edit connection item", async () => {
    const { dev } = await renderItems();
    await screen.findByText("Create read task");
    fireEvent.click(screen.getByText("Edit connection"));
    await screen.findByText("Server");
    await waitFor(() => expect(screen.getByDisplayValue(dev.name)).toBeTruthy());
  });
});
