// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import { createTestDevice, renderMenuItem } from "@/platform/device/testutil";
import { createSelection } from "@/platform/tree/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const readCreate = vi.fn();
const writeCreate = vi.fn();

const items = [
  Device.createTaskContextMenuItem({
    itemKey: "read",
    label: "Create Read Task",
    useCreate: () => readCreate,
  }),
  Device.createTaskContextMenuItem({
    itemKey: "write",
    label: "Create Write Task",
    useCreate: () => writeCreate,
  }),
];

const setup = async (configured: boolean, itemClient: Synnax | null = client) => {
  const onConfigure = vi.fn();
  const dev = await createTestDevice(client, { configured });
  await renderMenuItem(
    <Device.TaskContextMenuItems
      client={client}
      onConfigure={onConfigure}
      selection={createSelection({ ids: [device.resourceID(dev.key)] })}
      items={items}
    />,
    { client: itemClient },
  );
  return { onConfigure, key: dev.key };
};

describe("TaskContextMenuItems", () => {
  beforeEach(() => {
    readCreate.mockClear();
    writeCreate.mockClear();
  });

  it("should render nothing without task-create permission", async () => {
    await setup(true, null);
    expect(screen.queryByText("Create Read Task")).toBeNull();
    expect(screen.queryByText("Create Write Task")).toBeNull();
  });

  it("should configure an unconfigured device and create the task with its key", async () => {
    const { onConfigure, key } = await setup(false);
    await waitFor(() => expect(screen.getByText("Create Read Task")).toBeTruthy());
    fireEvent.click(screen.getByText("Create Read Task"));
    expect(onConfigure).toHaveBeenCalledWith(key);
    expect(readCreate).toHaveBeenCalledWith({ deviceKey: key });
    expect(writeCreate).not.toHaveBeenCalled();
  });

  it("should create from the clicked config without configuring an already-configured device", async () => {
    const { onConfigure, key } = await setup(true);
    await waitFor(() => expect(screen.getByText("Create Write Task")).toBeTruthy());
    fireEvent.click(screen.getByText("Create Write Task"));
    expect(onConfigure).not.toHaveBeenCalled();
    expect(writeCreate).toHaveBeenCalledWith({ deviceKey: key });
    expect(readCreate).not.toHaveBeenCalled();
  });

  describe("several selected devices", () => {
    const multiCreate = vi.fn();
    const multiItems = [
      Device.createTaskContextMenuItem({
        itemKey: "multi",
        label: "Create Multi Task",
        useCreate: () => multiCreate,
        multiDevice: { zeroProperties: { rate: 10 } },
      }),
      ...items,
    ];

    const setupSeveral = async (configured: boolean) => {
      const name = uniqueName("Zq Wv");
      const devices = [
        await createTestDevice(client, { configured, name }),
        await createTestDevice(client, { configured, name }),
      ];
      const onConfigure = vi.fn();
      await renderMenuItem(
        <Device.TaskContextMenuItems
          client={client}
          onConfigure={onConfigure}
          selection={createSelection({
            ids: devices.map((d) => device.resourceID(d.key)),
          })}
          items={multiItems}
        />,
        { client },
      );
      return { onConfigure, keys: devices.map((d) => d.key) };
    };

    beforeEach(() => multiCreate.mockClear());

    it("should offer only the items that take several devices", async () => {
      await setupSeveral(true);
      expect(await screen.findByText("Create Multi Task")).toBeTruthy();
      expect(screen.queryByText("Create Read Task")).toBeNull();
      expect(screen.queryByText("Create Write Task")).toBeNull();
    });

    it("should configure the devices without a dialog, then create one task", async () => {
      const { onConfigure, keys } = await setupSeveral(false);
      fireEvent.click(await screen.findByText("Create Multi Task"));
      await waitFor(() =>
        expect(multiCreate).toHaveBeenCalledWith({ deviceKeys: keys }),
      );
      expect(onConfigure).not.toHaveBeenCalled();
      const saved = await client.devices.retrieve({ key: keys });
      expect(saved.map((d) => d.configured)).toEqual([true, true]);
    });
  });
});
