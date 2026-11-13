// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type record } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Device } from "@/platform/device";
import {
  createTestDevice,
  setTestDeviceStatus,
  TEST_DEVICE_STATUS_MESSAGE,
} from "@/platform/device/testutil";
import { createAsyncSynnaxWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const ADAPTER = Device.createLegacyAdapter({
  make: "make",
  zeroProperties: { rate: 10, channels: {} },
});

describe("createLegacyAdapter", () => {
  let wrapper: React.FC<PropsWithChildren>;

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  // The cache receives each write from the Core's set signal.
  const awaitProperty = async (
    dev: device.Device,
    path: string,
    value: unknown,
  ): Promise<device.Device> => {
    await waitFor(async () =>
      expect((await client.devices.retrieve(dev.key)).properties[path]).toEqual(value),
    );
    return await client.devices.retrieve({ key: dev.key, includeStatus: true });
  };

  describe("useChangeIdentifier", () => {
    const changeIdentifier = async (
      dev: device.Device,
      identifier: string,
    ): Promise<device.Device> => {
      const { result } = renderHook(() => ADAPTER.useChangeIdentifier(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: dev.key, identifier });
      });
      return await awaitProperty(dev, "identifier", identifier);
    };

    it("should set an identifier on a device that has none", async () => {
      const dev = await createTestDevice(client, { properties: { rate: 50 } });
      const { properties } = await changeIdentifier(dev, "brand_new");
      expect(properties).toEqual({ rate: 50, identifier: "brand_new" });
    });

    it("should keep a property written after the device loaded", async () => {
      const dev = await createTestDevice(client, {
        properties: { identifier: "old_id", rate: 100 },
      });
      await client.devices.dispatch(
        dev.key,
        device.set({ path: ["properties", "rate"], value: 200 }),
      );
      const { properties } = await changeIdentifier(dev, "new_id");
      expect(properties).toEqual({ identifier: "new_id", rate: 200 });
    });

    it("should keep the name and the status that the Driver reported", async () => {
      const dev = await createTestDevice(client);
      await setTestDeviceStatus(client, dev);
      const updated = await changeIdentifier(dev, "new_id");
      expect(updated.name).toEqual(dev.name);
      expect(updated.status?.message).toEqual(TEST_DEVICE_STATUS_MESSAGE);
    });
  });

  describe("useSave", () => {
    const save = async (
      dev: device.Device,
      zeroProperties: record.Unknown = { rate: 10, channels: {} },
    ): Promise<device.Device> => {
      const adapter = Device.createLegacyAdapter({ make: "make", zeroProperties });
      const name = uniqueName("named");
      const { result } = renderHook(() => adapter.useSave(), { wrapper });
      await act(async () => {
        await result.current.updateAsync({ key: dev.key, name, identifier: "dev1" });
      });
      const saved = await awaitProperty(dev, "identifier", "dev1");
      expect(saved.name).toEqual(name);
      expect(saved.configured).toBe(true);
      return saved;
    };

    it("should fill the zero properties of an unconfigured device", async () => {
      const dev = await createTestDevice(client, { configured: false });
      const { properties } = await save(dev);
      expect(properties).toEqual({ rate: 10, channels: {}, identifier: "dev1" });
    });

    it("should keep the properties the device already has", async () => {
      const dev = await createTestDevice(client, {
        configured: false,
        properties: { rate: 99, nested: { port: { channel: 4 } } },
      });
      const { properties } = await save(dev);
      expect(properties).toEqual({
        rate: 99,
        channels: {},
        nested: { port: { channel: 4 } },
        identifier: "dev1",
      });
    });

    it("should only set the identifier with empty zero properties", async () => {
      const dev = await createTestDevice(client, {
        configured: false,
        properties: { rate: 5 },
      });
      const { properties } = await save(dev, {});
      expect(properties).toEqual({ rate: 5, identifier: "dev1" });
    });
  });
});
