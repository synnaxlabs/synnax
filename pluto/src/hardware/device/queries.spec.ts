// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, device } from "@synnaxlabs/client";
import { id, status } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Device } from "@/hardware/device";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });
  describe("retrieve", () => {
    it("should return a device", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const dev = await client.hardware.devices.create({
        key: id.create(),
        name: "test",
        rack: rack.key,
        location: "test",
        make: "test",
        model: "test",
        properties: {},
      });
      const { result } = renderHook(() => Device.useRetrieve({ key: dev.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.key).toEqual(dev.key);
    });

    it("should update the query when the device is updated", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const dev = await client.hardware.devices.create({
        key: id.create(),
        name: "test",
        rack: rack.key,
        location: "test",
        make: "test",
        model: "test",
        properties: {},
      });
      const { result } = renderHook(() => Device.useRetrieve({ key: dev.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.key).toEqual(dev.key);
      await client.hardware.devices.create({
        ...dev,
        name: "test2",
      });
      await waitFor(() => {
        expect(result.current.data?.name).toEqual("test2");
      });
    });

    it("should update the query when the device status is updated", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const dev = await client.hardware.devices.create({
        key: id.create(),
        name: "test",
        rack: rack.key,
        location: "test",
        make: "test",
        model: "test",
        properties: {},
      });
      const { result } = renderHook(() => Device.useRetrieve({ key: dev.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.key).toEqual(dev.key);
      const devStatus: device.Status = status.create({
        key: id.create(),
        variant: "success",
        message: "Device is happy as a clam",
        details: {
          rack: rack.key,
          device: dev.key,
        },
      });
      const writer = await client.openWriter([device.STATUS_CHANNEL_NAME]);
      await writer.write(device.STATUS_CHANNEL_NAME, [devStatus]);
      await writer.close();
      await waitFor(() => {
        expect(result.current.data?.status?.variant).toEqual("success");
        expect(result.current.data?.status?.details.device).toEqual(dev.key);
        expect(result.current.data?.status?.message).toEqual(
          "Device is happy as a clam",
        );
      });
    });
  });

  describe("useList", () => {
    it("should return a list of device keys", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const dev1 = await client.hardware.devices.create({
        key: id.create(),
        name: "device1",
        rack: rack.key,
        location: "location1",
        make: "make1",
        model: "model1",
        properties: {},
      });
      const dev2 = await client.hardware.devices.create({
        key: id.create(),
        name: "device2",
        rack: rack.key,
        location: "location2",
        make: "make2",
        model: "model2",
        properties: {},
      });

      const { result } = renderHook(() => Device.useList(), {
        wrapper,
      });
      result.current.retrieve({});
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThan(2);
      expect(result.current.data).toContain(dev1.key);
      expect(result.current.data).toContain(dev2.key);
    });

    it("should get individual devices using getItem", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const dev = await client.hardware.devices.create({
        key: id.create(),
        name: "testDevice",
        rack: rack.key,
        location: "location",
        make: "make",
        model: "model",
        properties: {},
      });

      const { result } = renderHook(() => Device.useList(), {
        wrapper,
      });
      result.current.retrieve({});
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const device = result.current.getItem(dev.key);
      expect(device?.key).toEqual(dev.key);
      expect(device?.name).toEqual("testDevice");
    });

    it("should filter devices by search term", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      await client.hardware.devices.create({
        key: id.create(),
        name: "device1",
        rack: rack.key,
        location: "location1",
        make: "make1",
        model: "model1",
        properties: {},
      });
      await client.hardware.devices.create({
        key: id.create(),
        name: "special",
        rack: rack.key,
        location: "location2",
        make: "make2",
        model: "model2",
        properties: {},
      });

      const { result } = renderHook(() => Device.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ searchTerm: "special" });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(1);
      expect(
        result.current.data
          .map((d) => result.current.getItem(d)?.name)
          .includes("special"),
      ).toBe(true);
    });

    it("should filter devices by makes", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const targetMake = id.create();
      const dev1 = await client.hardware.devices.create({
        key: id.create(),
        name: "device1",
        rack: rack.key,
        location: "location1",
        make: targetMake,
        model: "model1",
        properties: {},
      });
      await client.hardware.devices.create({
        key: id.create(),
        name: "device2",
        rack: rack.key,
        location: "location2",
        make: "otherMake",
        model: "model2",
        properties: {},
      });

      const { result } = renderHook(() => Device.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ makes: [targetMake] });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toEqual(1);
      expect(result.current.data[0]).toEqual(dev1.key);
    });

    it("should handle pagination with limit and offset", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      for (let i = 0; i < 5; i++)
        await client.hardware.devices.create({
          key: id.create(),
          name: `device${i}`,
          rack: rack.key,
          location: `location${i}`,
          make: `make${i}`,
          model: `model${i}`,
          properties: {},
        });

      const { result } = renderHook(() => Device.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ limit: 2, offset: 1 });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should update the list when a device is created", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });

      const { result } = renderHook(() => Device.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialLength = result.current.data.length;

      const newDev = await client.hardware.devices.create({
        key: id.create(),
        name: "newDevice",
        rack: rack.key,
        location: "newLocation",
        make: "newMake",
        model: "newModel",
        properties: {},
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(initialLength + 1);
        expect(result.current.data).toContain(newDev.key);
      });
    });

    it("should update the list when a device is updated", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const dev = await client.hardware.devices.create({
        key: id.create(),
        name: "original",
        rack: rack.key,
        location: "location",
        make: "make",
        model: "model",
        properties: {},
      });

      const { result } = renderHook(() => Device.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.getItem(dev.key)?.name).toEqual("original");

      await client.hardware.devices.create({
        ...dev,
        name: "updated",
      });

      await waitFor(() => {
        expect(result.current.getItem(dev.key)?.name).toEqual("updated");
      });
    });

    it("should remove device from list when deleted", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const dev = await client.hardware.devices.create({
        key: id.create(),
        name: "toDelete",
        rack: rack.key,
        location: "location",
        make: "make",
        model: "model",
        properties: {},
      });

      const { result } = renderHook(() => Device.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(dev.key);

      await client.hardware.devices.delete(dev.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(dev.key);
      });
    });

    it("should update device status in the list", async () => {
      const rack = await client.hardware.racks.create({
        name: "test",
      });
      const dev = await client.hardware.devices.create({
        key: id.create(),
        name: "device",
        rack: rack.key,
        location: "location",
        make: "make",
        model: "model",
        properties: {},
      });

      const { result } = renderHook(() => Device.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const devStatus: device.Status = status.create({
        key: id.create(),
        variant: "error",
        message: "Device has issues",
        details: {
          rack: rack.key,
          device: dev.key,
        },
      });
      await act(async () => {
        const writer = await client.openWriter([device.STATUS_CHANNEL_NAME]);
        await writer.write(device.STATUS_CHANNEL_NAME, [devStatus]);
        await writer.close();
      });

      await waitFor(() => {
        const deviceInList = result.current.getItem(dev.key);
        expect(deviceInList?.status?.variant).toEqual("error");
        expect(deviceInList?.status?.message).toEqual("Device has issues");
      });
    });

    describe("retrieveCached", () => {
      it("should use cached data on initial mount", async () => {
        const rack = await client.hardware.racks.create({
          name: "test",
        });
        const dev = await client.hardware.devices.create({
          key: id.create(),
          name: "cached_device",
          rack: rack.key,
          location: "cached_location",
          make: "cached_make",
          model: "cached_model",
          properties: {},
        });

        const { result: firstResult, unmount } = renderHook(() => Device.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({});
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        expect(firstResult.current.data).toContain(dev.key);
        unmount();

        const { result: secondResult } = renderHook(() => Device.useList(), {
          wrapper,
        });
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev.key);
      });

      it("should filter cached data by makes", async () => {
        const rack = await client.hardware.racks.create({
          name: "test",
        });
        const targetMake = id.create();
        const dev1 = await client.hardware.devices.create({
          key: id.create(),
          name: "device_make1",
          rack: rack.key,
          location: "location",
          make: targetMake,
          model: "model",
          properties: {},
        });
        const dev2 = await client.hardware.devices.create({
          key: id.create(),
          name: "device_make2",
          rack: rack.key,
          location: "location",
          make: "other_make",
          model: "model",
          properties: {},
        });

        const { result: firstResult, unmount } = renderHook(() => Device.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({});
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        unmount();

        const { result: secondResult } = renderHook(
          () => Device.useList({ initialParams: { makes: [targetMake] } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev1.key);
        expect(secondResult.current.data).not.toContain(dev2.key);
      });

      it("should filter cached data by models", async () => {
        const rack = await client.hardware.racks.create({
          name: "test",
        });
        const targetModel = id.create();
        const dev1 = await client.hardware.devices.create({
          key: id.create(),
          name: "device_model1",
          rack: rack.key,
          location: "location",
          make: "make",
          model: targetModel,
          properties: {},
        });
        const dev2 = await client.hardware.devices.create({
          key: id.create(),
          name: "device_model2",
          rack: rack.key,
          location: "location",
          make: "make",
          model: "other_model",
          properties: {},
        });

        const { result: firstResult, unmount } = renderHook(() => Device.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({});
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        unmount();

        const { result: secondResult } = renderHook(
          () => Device.useList({ initialParams: { models: [targetModel] } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev1.key);
        expect(secondResult.current.data).not.toContain(dev2.key);
      });

      it("should filter cached data by racks", async () => {
        const rack1 = await client.hardware.racks.create({
          name: "rack1",
        });
        const rack2 = await client.hardware.racks.create({
          name: "rack2",
        });
        const dev1 = await client.hardware.devices.create({
          key: id.create(),
          name: "device_rack1",
          rack: rack1.key,
          location: "location",
          make: "make",
          model: "model",
          properties: {},
        });
        const dev2 = await client.hardware.devices.create({
          key: id.create(),
          name: "device_rack2",
          rack: rack2.key,
          location: "location",
          make: "make",
          model: "model",
          properties: {},
        });

        const { result: firstResult, unmount } = renderHook(() => Device.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({});
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        unmount();

        const { result: secondResult } = renderHook(
          () => Device.useList({ initialParams: { racks: [rack1.key] } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev1.key);
        expect(secondResult.current.data).not.toContain(dev2.key);
      });

      it("should filter cached data by locations", async () => {
        const rack = await client.hardware.racks.create({
          name: "test",
        });
        const targetLocation = id.create();
        const dev1 = await client.hardware.devices.create({
          key: id.create(),
          name: "device_loc1",
          rack: rack.key,
          location: targetLocation,
          make: "make",
          model: "model",
          properties: {},
        });
        const dev2 = await client.hardware.devices.create({
          key: id.create(),
          name: "device_loc2",
          rack: rack.key,
          location: "other_location",
          make: "make",
          model: "model",
          properties: {},
        });

        const { result: firstResult, unmount } = renderHook(() => Device.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({});
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        unmount();

        const { result: secondResult } = renderHook(
          () => Device.useList({ initialParams: { locations: [targetLocation] } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev1.key);
        expect(secondResult.current.data).not.toContain(dev2.key);
      });

      it("should filter cached data by names", async () => {
        const rack = await client.hardware.racks.create({
          name: "test",
        });
        const targetName = id.create();
        const dev1 = await client.hardware.devices.create({
          key: id.create(),
          name: targetName,
          rack: rack.key,
          location: "location",
          make: "make",
          model: "model",
          properties: {},
        });
        const dev2 = await client.hardware.devices.create({
          key: id.create(),
          name: "other_name",
          rack: rack.key,
          location: "location",
          make: "make",
          model: "model",
          properties: {},
        });

        const { result: firstResult, unmount } = renderHook(() => Device.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({});
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        unmount();

        const { result: secondResult } = renderHook(
          () => Device.useList({ initialParams: { names: [targetName] } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev1.key);
        expect(secondResult.current.data).not.toContain(dev2.key);
      });

      it("should handle combined filters", async () => {
        const rack1 = await client.hardware.racks.create({
          name: "test_rack",
        });
        const targetMake = id.create();
        const targetModel = id.create();
        const dev1 = await client.hardware.devices.create({
          key: id.create(),
          name: "device_combined1",
          rack: rack1.key,
          location: "location",
          make: targetMake,
          model: targetModel,
          properties: {},
        });
        const dev2 = await client.hardware.devices.create({
          key: id.create(),
          name: "device_combined2",
          rack: rack1.key,
          location: "location",
          make: targetMake,
          model: "other_model",
          properties: {},
        });
        const dev3 = await client.hardware.devices.create({
          key: id.create(),
          name: "device_combined3",
          rack: rack1.key,
          location: "location",
          make: "other_make",
          model: targetModel,
          properties: {},
        });

        const { result: firstResult, unmount } = renderHook(() => Device.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({});
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        unmount();

        const { result: secondResult } = renderHook(
          () =>
            Device.useList({
              initialParams: {
                makes: [targetMake],
                models: [targetModel],
                racks: [rack1.key],
              },
            }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev1.key);
        expect(secondResult.current.data).not.toContain(dev2.key);
        expect(secondResult.current.data).not.toContain(dev3.key);
      });
    });
  });
});
