// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type Synnax, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { Device as PDevice } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { createTestSink } from "@/platform/analytics/testutil";
import { Device } from "@/platform/device";
import {
  createTestDevice,
  type CreateTestDeviceOptions,
  renderMenuItem,
} from "@/platform/device/testutil";
import { createSelection } from "@/platform/tree/testutil";
import { awaitGranted, CaptureStatuses } from "@/testutil";

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

    /** @returns a free-looking identifier of the length, made of lowercase letters. */
    const createIdentifier = (length: number): string =>
      Array.from({ length }, () =>
        String.fromCharCode(97 + Math.floor(Math.random() * 26)),
      ).join("");

    /** @returns the devices with the keys, in the order of the keys. */
    const retrieveEach = async (keys: device.Key[]): Promise<device.Device[]> =>
      await Promise.all(
        keys.map(async (key) => await client.devices.retrieve({ key })),
      );

    interface SetupSeveralOptions {
      /** The devices to select, on one new rack unless one names its own rack. */
      devices: CreateTestDeviceOptions[];
      /** The identifiers that the item suggests for each device. */
      ideas?: string[];
    }

    const setupSeveral = async ({ devices, ideas }: SetupSeveralOptions) => {
      const rack = (await client.racks.create({ name: uniqueName("rack") })).key;
      const created: device.Device[] = [];
      for (const options of devices)
        created.push(await createTestDevice(client, { rack, ...options }));
      const multiItems = [
        Device.createTaskContextMenuItem({
          itemKey: "multi",
          label: "Create Multi Task",
          useCreate: () => multiCreate,
          configureMany: {
            zeroProperties: { rate: 10 },
            suggestIdentifiers: ({ name }) =>
              ideas ?? strings.createShortIdentifiers(name),
          },
        }),
        ...items,
      ];
      const onConfigure = vi.fn();
      const analytics = createTestSink();
      let statuses: Status.NotificationSpec[] = [];
      const { wrapper } = await renderMenuItem(
        <>
          <Device.TaskContextMenuItems
            client={client}
            onConfigure={onConfigure}
            selection={createSelection({
              ids: created.map((d) => device.resourceID(d.key)),
            })}
            items={multiItems}
          />
          <CaptureStatuses onStatuses={(s) => (statuses = s)} />
        </>,
        { client, analytics },
      );
      return {
        onConfigure,
        analytics,
        wrapper,
        statuses: () => statuses,
        keys: created.map((d) => d.key),
      };
    };

    beforeEach(() => multiCreate.mockClear());

    it("should offer only the items that take several devices", async () => {
      await setupSeveral({ devices: [{}, {}] });
      expect(await screen.findByText("Create Multi Task")).toBeTruthy();
      expect(screen.queryByText("Create Read Task")).toBeNull();
      expect(screen.queryByText("Create Write Task")).toBeNull();
    });

    it("should offer no item for devices on different racks", async () => {
      await awaitGranted(client, task.TYPE_RESOURCE_ID);
      const other = (await client.racks.create({ name: uniqueName("rack") })).key;
      const { keys, wrapper } = await setupSeveral({ devices: [{}, { rack: other }] });
      const { result } = renderHook(
        () => PDevice.useResultMultiple({ where: { key: { in: keys } } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.data).toHaveLength(2));
      await act(async () => {});
      expect(screen.queryByText("Create Multi Task")).toBeNull();
    });

    it("should configure the devices with different identifiers, then create one task", async () => {
      const idea = createIdentifier(8);
      const { onConfigure, analytics, keys } = await setupSeveral({
        devices: [{ configured: false }, { configured: false }],
        ideas: [idea],
      });
      fireEvent.click(await screen.findByText("Create Multi Task"));
      await waitFor(() =>
        expect(multiCreate).toHaveBeenCalledWith({ deviceKeys: keys }),
      );
      expect(onConfigure).not.toHaveBeenCalled();
      const saved = await retrieveEach(keys);
      expect(saved.map((d) => d.configured)).toEqual([true, true]);
      expect(saved.map((d) => d.properties.rate)).toEqual([10, 10]);
      expect(saved.map((d) => d.properties.identifier)).toEqual([idea, `${idea}_2`]);
      expect(analytics.capture.mock.calls).toEqual([
        ["device_configured", { make: "make" }],
        ["device_configured", { make: "make" }],
      ]);
    });

    it("should leave a configured device as it is", async () => {
      const kept = createIdentifier(8);
      const idea = createIdentifier(8);
      const { analytics, keys } = await setupSeveral({
        devices: [{ properties: { identifier: kept } }, { configured: false }],
        ideas: [idea],
      });
      fireEvent.click(await screen.findByText("Create Multi Task"));
      await waitFor(() =>
        expect(multiCreate).toHaveBeenCalledWith({ deviceKeys: keys }),
      );
      const [configured, found] = await retrieveEach(keys);
      expect(configured.properties.identifier).toBe(kept);
      expect(configured.properties.rate).toBeUndefined();
      expect(found.properties.identifier).toBe(idea);
      expect(analytics.capture.mock.calls).toEqual([
        ["device_configured", { make: "make" }],
      ]);
    });

    it("should create no task when every numbered idea is too long or taken", async () => {
      const idea = createIdentifier(12);
      await createTestDevice(client, { properties: { identifier: idea } });
      const { analytics, keys, statuses } = await setupSeveral({
        devices: [
          { name: "Zq", configured: false },
          { name: "Wv", configured: false },
        ],
        ideas: [idea],
      });
      fireEvent.click(await screen.findByText("Create Multi Task"));
      await waitFor(() =>
        expect(statuses()).toContainEqual(
          expect.objectContaining({
            description:
              "Synnax cannot suggest a free identifier for Zq. Configure the device " +
              "by hand.",
          }),
        ),
      );
      expect(multiCreate).not.toHaveBeenCalled();
      const saved = await retrieveEach(keys);
      expect(saved.map((d) => d.configured)).toEqual([false, false]);
      expect(analytics.capture).not.toHaveBeenCalled();
    });
  });
});
