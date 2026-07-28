// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, NotFoundError, status } from "@synnaxlabs/client";
import { createTestClient, isLive } from "@synnaxlabs/client/testutil";
import { id, type record } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";
import { z } from "zod";

import { Device } from "@/device";
import { Status } from "@/status";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useRetrieve", () => {
    it("should return a device", async () => {
      const rack = await client.racks.create({
        name: "test",
      });
      const dev = await client.devices.create({
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
      const rack = await client.racks.create({ name: "test" });
      const dev = await client.devices.create({
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
      await act(async () => {
        await client.devices.create({
          ...dev,
          name: "test2",
        });
      });
      await waitFor(() => {
        expect(result.current.data?.name).toEqual("test2");
      });
    });

    it("should update the query when the device status is updated", async () => {
      const rack = await client.racks.create({
        name: "test",
      });
      const dev = await client.devices.create({
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
      const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
        key: device.statusKey(dev.key),
        variant: "success",
        message: "Device is happy as a clam",
        details: {
          rack: rack.key,
          device: dev.key,
        },
      });
      await client.statuses.set(devStatus);
      await waitFor(() => {
        expect(result.current.data?.status?.variant).toEqual("success");
        expect(result.current.data?.status?.details.device).toEqual(dev.key);
        expect(result.current.data?.status?.message).toEqual(
          "Device is happy as a clam",
        );
      });
    });

    it("should not set status to undefined when the device is updated", async () => {
      const rack = await client.racks.create({ name: "test" });
      const dev = await client.devices.create({
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
      const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
        key: device.statusKey(dev.key),
        variant: "success",
        message: "Device is connected",
        details: {
          rack: rack.key,
          device: dev.key,
        },
      });
      await client.statuses.set(devStatus);
      await waitFor(() => {
        expect(result.current.data?.status?.variant).toEqual("success");
      });
      await act(async () => {
        await client.devices.create({
          ...dev,
          name: "updated-name",
        });
      });
      await waitFor(() => {
        expect(result.current.data?.name).toEqual("updated-name");
        expect(result.current.data?.status).not.toBeUndefined();
      });
    });

    it("should correctly retrieve the devices status even when the query is cached", async () => {
      const rack = await client.racks.create({ name: "test" });
      const dev = await client.devices.create({
        key: id.create(),
        name: "test",
        rack: rack.key,
        location: "test",
        make: "test",
        model: "test",
        properties: {},
      });
      const { result: result1 } = renderHook(
        () => Device.useRetrieve({ key: dev.key }),
        { wrapper },
      );
      await waitFor(() => expect(result1.current.variant).toEqual("success"));
      expect(result1.current.data?.key).toEqual(dev.key);
      expect(result1.current.data?.status).toBeDefined();
      await act(async () => {
        await client.statuses.set(
          status.create<typeof device.statusDetailsZ>({
            key: device.statusKey(dev.key),
            variant: "success",
            message: "Device is happy as a clam",
            details: { rack: rack.key, device: dev.key },
          }),
        );
      });
      const { result: result2 } = renderHook(
        () => Device.useRetrieve({ key: dev.key }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result2.current.variant).toEqual("success");
        expect(result2.current.data?.status?.variant).toEqual("success");
        expect(result2.current.data?.status?.message).toEqual(
          "Device is happy as a clam",
        );
      });
    });
  });

  describe("useList", () => {
    it("should return a list of device keys", async () => {
      const rack = await client.racks.create({
        name: "test",
      });
      const dev1 = await client.devices.create({
        key: id.create(),
        name: "device1",
        rack: rack.key,
        location: "location1",
        make: "make1",
        model: "model1",
        properties: {},
      });
      const dev2 = await client.devices.create({
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
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(dev1.key);
      expect(result.current.data).toContain(dev2.key);
    });

    it("should get individual devices using getItem", async () => {
      const rack = await client.racks.create({
        name: "test",
      });
      const dev = await client.devices.create({
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
      const rack = await client.racks.create({
        name: "test",
      });
      await client.devices.create({
        key: id.create(),
        name: "device1",
        rack: rack.key,
        location: "location1",
        make: "make1",
        model: "model1",
        properties: {},
      });
      await client.devices.create({
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
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data.length).toBeGreaterThanOrEqual(1);
        expect(
          result.current.data
            .map((d) => result.current.getItem(d)?.name)
            .includes("special"),
        ).toBe(true);
      });
    });

    it("should filter devices by makes", async () => {
      const rack = await client.racks.create({
        name: "test",
      });
      const targetMake = id.create();
      const dev1 = await client.devices.create({
        key: id.create(),
        name: "device1",
        rack: rack.key,
        location: "location1",
        make: targetMake,
        model: "model1",
        properties: {},
      });
      await client.devices.create({
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
      const rack = await client.racks.create({
        name: "test",
      });
      for (let i = 0; i < 5; i++)
        await client.devices.create({
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
      const rack = await client.racks.create({
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

      const newDev = await client.devices.create({
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
      const rack = await client.racks.create({
        name: "test",
      });
      const dev = await client.devices.create({
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

      await client.devices.create({
        ...dev,
        name: "updated",
      });

      await waitFor(() => {
        expect(result.current.getItem(dev.key)?.name).toEqual("updated");
      });
    });

    it("should remove device from list when deleted", async () => {
      const rack = await client.racks.create({
        name: "test",
      });
      const dev = await client.devices.create({
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

      await client.devices.delete(dev.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(dev.key);
      });
    });

    it("should update device status in the list", async () => {
      const rack = await client.racks.create({
        name: "test",
      });
      const dev = await client.devices.create({
        key: id.create(),
        name: "device",
        rack: rack.key,
        location: "location",
        make: "make",
        model: "model",
        properties: {},
      });

      const { result } = renderHook(
        () => ({
          list: Device.useList(),
          status: Status.useSet(),
        }),
        {
          wrapper,
        },
      );
      result.current.list.retrieve({});
      await waitFor(() => expect(result.current.list.variant).toEqual("success"));

      const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
        key: device.statusKey(dev.key),
        variant: "error",
        message: "Device has issues",
        details: { rack: rack.key, device: dev.key },
      });
      await act(async () => {
        await result.current.status.updateAsync({ statuses: devStatus });
      });

      await waitFor(() => {
        const deviceInList = result.current.list.getItem(dev.key);
        expect(deviceInList?.status?.variant).toEqual("error");
        expect(deviceInList?.status?.message).toEqual("Device has issues");
      });
    });

    describe("retrieveCached", () => {
      it("should use cached data on initial mount", async () => {
        const rack = await client.racks.create({
          name: "test",
        });
        const dev = await client.devices.create({
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
        const rack = await client.racks.create({
          name: "test",
        });
        const targetMake = id.create();
        const dev1 = await client.devices.create({
          key: id.create(),
          name: "device_make1",
          rack: rack.key,
          location: "location",
          make: targetMake,
          model: "model",
          properties: {},
        });
        const dev2 = await client.devices.create({
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
          () => Device.useList({ initialQuery: { makes: [targetMake] } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev1.key);
        expect(secondResult.current.data).not.toContain(dev2.key);
      });

      it("should filter cached data by models", async () => {
        const rack = await client.racks.create({
          name: "test",
        });
        const targetModel = id.create();
        const dev1 = await client.devices.create({
          key: id.create(),
          name: "device_model1",
          rack: rack.key,
          location: "location",
          make: "make",
          model: targetModel,
          properties: {},
        });
        const dev2 = await client.devices.create({
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
          () => Device.useList({ initialQuery: { models: [targetModel] } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev1.key);
        expect(secondResult.current.data).not.toContain(dev2.key);
      });

      it("should filter cached data by racks", async () => {
        const rack1 = await client.racks.create({
          name: "rack1",
        });
        const rack2 = await client.racks.create({
          name: "rack2",
        });
        const dev1 = await client.devices.create({
          key: id.create(),
          name: "device_rack1",
          rack: rack1.key,
          location: "location",
          make: "make",
          model: "model",
          properties: {},
        });
        const dev2 = await client.devices.create({
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
          () => Device.useList({ initialQuery: { racks: [rack1.key] } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev1.key);
        expect(secondResult.current.data).not.toContain(dev2.key);
      });

      it("should filter cached data by locations", async () => {
        const rack = await client.racks.create({
          name: "test",
        });
        const targetLocation = id.create();
        const dev1 = await client.devices.create({
          key: id.create(),
          name: "device_loc1",
          rack: rack.key,
          location: targetLocation,
          make: "make",
          model: "model",
          properties: {},
        });
        const dev2 = await client.devices.create({
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
          () => Device.useList({ initialQuery: { locations: [targetLocation] } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev1.key);
        expect(secondResult.current.data).not.toContain(dev2.key);
      });

      it("should filter cached data by names", async () => {
        const rack = await client.racks.create({
          name: "test",
        });
        const targetName = id.create();
        const dev1 = await client.devices.create({
          key: id.create(),
          name: targetName,
          rack: rack.key,
          location: "location",
          make: "make",
          model: "model",
          properties: {},
        });
        const dev2 = await client.devices.create({
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
          () => Device.useList({ initialQuery: { names: [targetName] } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(dev1.key);
        expect(secondResult.current.data).not.toContain(dev2.key);
      });

      it("should handle combined filters", async () => {
        const rack1 = await client.racks.create({
          name: "test_rack",
        });
        const targetMake = id.create();
        const targetModel = id.create();
        const dev1 = await client.devices.create({
          key: id.create(),
          name: "device_combined1",
          rack: rack1.key,
          location: "location",
          make: targetMake,
          model: targetModel,
          properties: {},
        });
        const dev2 = await client.devices.create({
          key: id.create(),
          name: "device_combined2",
          rack: rack1.key,
          location: "location",
          make: targetMake,
          model: "other_model",
          properties: {},
        });
        const dev3 = await client.devices.create({
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
              initialQuery: {
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

  describe("useCreate", () => {
    it("should create a device", async () => {
      const rack = await client.racks.create({
        name: "test",
      });
      const { result } = renderHook(() => Device.useCreate(), {
        wrapper,
      });
      const key = id.create();
      const dev: device.Device = {
        key,
        rack: rack.key,
        location: "location",
        name: "test",
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
        configured: true,
      };
      await act(async () => {
        await result.current.updateAsync(dev);
      });
      expect(result.current.variant).toEqual("success");
      const retrieved = await client.devices.retrieve({ key });
      expect(retrieved.key).toEqual(key);
      expect(retrieved.name).toEqual("test");
      expect(retrieved.make).toEqual("ni");
      expect(retrieved.model).toEqual("dog");
      expect(retrieved.properties).toEqual({ cat: "dog" });
    });
  });

  describe("useRename", () => {
    it("should rename a device", async () => {
      const rack = await client.racks.create({
        name: "test",
      });
      const dev = await client.devices.create({
        key: id.create(),
        name: "test",
        rack: rack.key,
        location: "location",
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
      });
      const { result } = renderHook(() => Device.useRename(), {
        wrapper,
      });
      await act(async () => {
        await result.current.updateAsync({ key: dev.key, name: "new-name" });
      });
      expect(result.current.variant).toEqual("success");
      const retrieved = await client.devices.retrieve({ key: dev.key });
      expect(retrieved.name).toEqual("new-name");
    });
  });

  describe("useDelete", () => {
    it("should delete a device", async () => {
      const rack = await client.racks.create({
        name: "test",
      });
      const dev = await client.devices.create({
        key: id.create(),
        name: "test",
        rack: rack.key,
        location: "location",
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
      });
      const { result } = renderHook(() => Device.useDelete(), {
        wrapper,
      });
      await act(async () => {
        await result.current.updateAsync(dev.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(client.devices.retrieve({ key: dev.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("useRetrieveGroupID", () => {
    it("should retrieve the group ID", async () => {
      const { result } = renderHook(() => Device.useRetrieveGroupID({}), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.type).toEqual("group");
      expect(result.current.data?.key).not.toBeFalsy();
    });
  });

  describe("keyed multi-retrieve", () => {
    it("should retrieve multiple devices from the server when none are cached", async () => {
      const rack = await client.racks.create({ name: "test" });
      const dev1 = await client.devices.create({
        key: id.create(),
        name: "device1",
        rack: rack.key,
        location: "location1",
        make: "make1",
        model: "model1",
        properties: {},
      });
      const dev2 = await client.devices.create({
        key: id.create(),
        name: "device2",
        rack: rack.key,
        location: "location2",
        make: "make2",
        model: "model2",
        properties: {},
      });

      const devices = await client.devices.retrieve({
        keys: [dev1.key, dev2.key],
        includeStatus: true,
      });

      expect(devices).toHaveLength(2);
      expect(devices.map((d) => d.key)).toContain(dev1.key);
      expect(devices.map((d) => d.key)).toContain(dev2.key);
    });

    it("should use cached devices and only fetch missing ones", async () => {
      const rack = await client.racks.create({ name: "test" });
      const dev1 = await client.devices.create({
        key: id.create(),
        name: "cached_device",
        rack: rack.key,
        location: "location1",
        make: "make1",
        model: "model1",
        properties: {},
      });
      const dev2 = await client.devices.create({
        key: id.create(),
        name: "uncached_device",
        rack: rack.key,
        location: "location2",
        make: "make2",
        model: "model2",
        properties: {},
      });

      await client.devices.retrieve({ key: dev1.key });

      const devices = await client.devices.retrieve({
        keys: [dev1.key, dev2.key],
        includeStatus: true,
      });

      expect(devices).toHaveLength(2);
      expect(devices.map((d) => d.key)).toContain(dev1.key);
      expect(devices.map((d) => d.key)).toContain(dev2.key);
      expect(isLive(client.devices.getCached({ keys: [dev1.key] }))).toBe(true);
      expect(isLive(client.devices.getCached({ keys: [dev2.key] }))).toBe(true);
    });

    it("should return all cached devices when all are in the store", async () => {
      const rack = await client.racks.create({ name: "test" });
      const dev1 = await client.devices.create({
        key: id.create(),
        name: "cached1",
        rack: rack.key,
        location: "location1",
        make: "make1",
        model: "model1",
        properties: {},
      });
      const dev2 = await client.devices.create({
        key: id.create(),
        name: "cached2",
        rack: rack.key,
        location: "location2",
        make: "make2",
        model: "model2",
        properties: {},
      });

      await client.devices.retrieve({ keys: [dev1.key, dev2.key] });

      const devices = await client.devices.retrieve({
        keys: [dev1.key, dev2.key],
        includeStatus: true,
      });

      expect(devices).toHaveLength(2);
      expect(devices.map((d) => d.key)).toContain(dev1.key);
      expect(devices.map((d) => d.key)).toContain(dev2.key);
    });

    it("should include statuses on fetched devices", async () => {
      const rack = await client.racks.create({ name: "test" });
      const dev = await client.devices.create({
        key: id.create(),
        name: "device_with_status",
        rack: rack.key,
        location: "location",
        make: "make",
        model: "model",
        properties: {},
      });

      const devices = await client.devices.retrieve({
        keys: [dev.key],
        includeStatus: true,
      });

      expect(devices).toHaveLength(1);
      expect(devices[0].status).toBeDefined();
      expect(devices[0].status?.key).toEqual(device.statusKey(dev.key));
    });

    it("should fetch statuses for cached devices", async () => {
      const rack = await client.racks.create({ name: "test" });
      const dev = await client.devices.create({
        key: id.create(),
        name: "cached_device",
        rack: rack.key,
        location: "location",
        make: "make",
        model: "model",
        properties: {},
      });

      await client.devices.retrieve({ key: dev.key });

      const devices = await client.devices.retrieve({
        keys: [dev.key],
        includeStatus: true,
      });

      expect(devices).toHaveLength(1);
      expect(devices[0].status).toBeDefined();
    });
  });

  describe("single retrieve", () => {
    it("should return an undefined status when a cached device has no status", async () => {
      // A cold client: cached answers carry status whenever one is already
      // in the cache, so absence is only deterministic with nothing cached.
      const cold = createTestClient();
      const rack = await cold.racks.create({ name: "test" });
      const key = id.create();
      await cold.devices.create({
        key,
        rack: rack.key,
        name: "cached_no_status",
        location: "location",
        make: "make",
        model: "model",
        configured: true,
        properties: {},
      });
      const dev = await cold.devices.retrieve({ key, includeStatus: false });
      expect(dev.key).toEqual(key);
      expect(dev.status).toBeUndefined();
    });
  });

  describe("useForm", () => {
    describe("create mode", () => {
      it("should initialize with default values for new device", async () => {
        const { result } = renderHook(() => Device.useForm({ query: { key: "" } }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.form.value()).toBeDefined());

        const formData = result.current.form.value();
        expect(formData.name).toBe("");
        expect(formData.make).toBe("");
        expect(formData.model).toBe("");
        expect(formData.location).toBe("");
        expect(formData.properties).toEqual({});
      });

      it("should create a new device on save", async () => {
        const rack = await client.racks.create({
          name: "test form rack",
        });
        const useForm = Device.createForm();
        const { result } = renderHook(() => useForm({ query: { key: "" } }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        act(() => {
          result.current.form.set("rack", rack.key);
          result.current.form.set("name", "Test Form Device");
          result.current.form.set("make", "TestMake");
          result.current.form.set("model", "TestModel");
          result.current.form.set("location", "Lab1");
        });

        await act(async () => {
          result.current.save();
        });

        await waitFor(() => {
          expect(result.current.variant).toBe("success");
        });

        const key = result.current.form.get<device.Key>("key").value;
        // parent and status are cache enrichments whose presence depends on
        // which relationship queries have already run; don't pin them.
        const {
          parent: _parent,
          status: _status,
          ...retrieved
        } = await client.devices.retrieve({ key });
        expect(retrieved).toEqual({
          key,
          name: "Test Form Device",
          make: "TestMake",
          model: "TestModel",
          location: "Lab1",
          rack: rack.key,
          configured: true,
          properties: {},
        });
      });

      it("should validate required fields", async () => {
        const useForm = Device.createForm();
        const { result } = renderHook(() => useForm({ query: { key: "" } }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        await act(async () => {
          result.current.save();
        });

        const nameField = result.current.form.get("name");
        expect(nameField.status.message).toBe("name is required");
        const makeField = result.current.form.get("make");
        expect(makeField.status.message).toBe("make is required");
        const modelField = result.current.form.get("model");
        expect(modelField.status.message).toBe("model is required");
        const locationField = result.current.form.get("location");
        expect(locationField.status.message).toBe("location is required");
      });

      it("should support custom properties", async () => {
        const customPropertiesZ = z.object({
          serialNumber: z.string(),
          calibrationDate: z.string(),
        });

        const rack = await client.racks.create({
          name: "test custom props rack",
        });
        const useForm = Device.createForm({
          properties: customPropertiesZ,
          make: z.string(),
          model: z.string(),
        });
        const { result } = renderHook(() => useForm({ query: { key: "" } }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const customProps: z.infer<typeof customPropertiesZ> = {
          serialNumber: "SN123456",
          calibrationDate: "2024-01-01",
        };

        act(() => {
          result.current.form.set("rack", rack.key);
          result.current.form.set("name", "Custom Device");
          result.current.form.set("make", "CustomMake");
          result.current.form.set("model", "CustomModel");
          result.current.form.set("location", "Lab2");
          result.current.form.set("properties", customProps);
        });

        await act(async () => {
          result.current.save();
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const formData = result.current.form.value();
        expect(formData.properties).toEqual(customProps);
      });
    });

    describe("update mode", () => {
      it("should load existing device data", async () => {
        const rack = await client.racks.create({
          name: "test update rack",
        });
        const testDevice = await client.devices.create({
          key: id.create(),
          rack: rack.key,
          name: "Existing Device",
          make: "ExistingMake",
          model: "ExistingModel",
          location: "Lab3",
          properties: { testProp: "value" },
        });

        const useForm = Device.createForm();
        const { result } = renderHook(
          () =>
            useForm({
              query: { key: testDevice.key },
            }),
          { wrapper },
        );

        await waitFor(() => {
          const formData = result.current.form.value();
          expect(formData.name).toBe("Existing Device");
        });

        const formData = result.current.form.value();
        expect(formData.key).toBe(testDevice.key);
        expect(formData.rack).toBe(rack.key);
        expect(formData.make).toBe("ExistingMake");
        expect(formData.model).toBe("ExistingModel");
        expect(formData.location).toBe("Lab3");
        expect(formData.properties).toEqual({ testProp: "value" });
      });

      it("should update existing device", async () => {
        const rack = await client.racks.create({
          name: "test update rack 2",
        });
        const testDevice = await client.devices.create({
          key: id.create(),
          rack: rack.key,
          name: "Device to Update",
          make: "OriginalMake",
          model: "OriginalModel",
          location: "Lab3",
          properties: {},
        });

        const { result } = renderHook(
          () => Device.useForm({ query: { key: testDevice.key } }),
          { wrapper },
        );

        await waitFor(() => {
          const formData = result.current.form.value();
          expect(formData.name).toBe("Device to Update");
        });

        act(() => {
          result.current.form.set("name", "Updated Device Name");
          result.current.form.set("location", "Lab4");
        });

        await act(async () => {
          result.current.save();
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const updatedDevice = await client.devices.retrieve({
          key: testDevice.key,
        });
        expect(updatedDevice.name).toBe("Updated Device Name");
        expect(updatedDevice.location).toBe("Lab4");
      });

      it("should sync streamed status updates into the form", async () => {
        const rack = await client.racks.create({ name: "test status sync rack" });
        const testDevice = await client.devices.create({
          key: id.create(),
          rack: rack.key,
          name: "Status Sync Device",
          make: "make",
          model: "model",
          location: "Lab5",
          properties: {},
        });

        const { result } = renderHook(
          () => Device.useForm({ query: { key: testDevice.key } }),
          { wrapper },
        );

        await waitFor(() => {
          expect(result.current.form.value().name).toBe("Status Sync Device");
        });

        const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
          key: device.statusKey(testDevice.key),
          variant: "warning",
          message: "Device is degraded",
          details: { rack: rack.key, device: testDevice.key },
        });
        await client.statuses.set(devStatus);

        await waitFor(() => {
          expect(result.current.form.value().status?.message).toBe(
            "Device is degraded",
          );
          expect(result.current.form.value().status?.variant).toBe("warning");
        });
      });
    });

    describe("validation", () => {
      it("should validate name field", async () => {
        const { result } = renderHook(() => Device.useForm({ query: { key: "" } }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        act(() => {
          result.current.form.set("name", "");
        });

        const isValid = result.current.form.validate("name");
        expect(isValid).toBe(false);

        const msg = result.current.form.get("name").status.message;
        expect(msg).toEqual("name is required");
      });
    });
  });

  describe("with schemas", () => {
    const propertiesSchema = z.object({
      sampleRate: z.number(),
      channels: z.record(z.string(), z.number()),
    });
    const makeSchema = z.literal("custom_make");
    const modelSchema = z.string();
    const schemas = {
      properties: propertiesSchema,
      make: makeSchema,
      model: modelSchema,
    };

    describe("createRetrieve", () => {
      it("should retrieve a device with typed properties", async () => {
        const rack = await client.racks.create({ name: "schema-test-rack" });
        const dev = await client.devices.create(
          {
            key: id.create(),
            name: "schema-test-device",
            rack: rack.key,
            location: "test",
            make: "custom_make",
            model: "test",
            properties: { sampleRate: 1000, channels: { ai0: 1, ai1: 2 } },
          },
          schemas,
        );

        const { useRetrieve } = Device.createRetrieve(schemas);
        const { result } = renderHook(() => useRetrieve({ key: dev.key }), { wrapper });

        await waitFor(() => expect(result.current.variant).toEqual("success"));
        expect(result.current.data?.properties.sampleRate).toBe(1000);
        expect(result.current.data?.properties.channels).toEqual({ ai0: 1, ai1: 2 });
        expect(result.current.data?.make).toBe("custom_make");
      });

      it("should update typed device when properties change", async () => {
        const rack = await client.racks.create({ name: "schema-update-rack" });
        const dev = await client.devices.create(
          {
            key: id.create(),
            name: "schema-update-device",
            rack: rack.key,
            location: "test",
            make: "custom_make",
            model: "test",
            properties: { sampleRate: 100, channels: {} },
          },
          schemas,
        );

        const { useRetrieve } = Device.createRetrieve(schemas);
        const { result } = renderHook(() => useRetrieve({ key: dev.key }), { wrapper });

        await waitFor(() => expect(result.current.variant).toEqual("success"));
        expect(result.current.data?.properties.sampleRate).toBe(100);

        await act(async () => {
          await client.devices.create(
            {
              ...dev,
              properties: { sampleRate: 500, channels: { ch1: 10 } },
            },
            schemas,
          );
        });

        await waitFor(() => {
          expect(result.current.data?.properties.sampleRate).toBe(500);
          expect(result.current.data?.properties.channels).toEqual({ ch1: 10 });
        });
      });
    });

    describe("createCreate", () => {
      it("should create a device with typed properties", async () => {
        const rack = await client.racks.create({ name: "schema-create-rack" });
        const { useUpdate } = Device.createCreate(schemas);
        const { result } = renderHook(() => useUpdate(), { wrapper });

        const key = id.create();
        await act(async () => {
          await result.current.updateAsync({
            key,
            rack: rack.key,
            location: "test",
            name: "created-with-schema",
            make: "custom_make",
            model: "test",
            properties: { sampleRate: 2000, channels: { x: 5 } },
            configured: true,
          });
        });

        expect(result.current.variant).toEqual("success");

        const retrieved = await client.devices.retrieve({
          key,
          schemas,
        });
        expect(retrieved.properties.sampleRate).toBe(2000);
        expect(retrieved.properties.channels).toEqual({ x: 5 });
      });
    });

    describe("schema-typed retrieve with a generically cached device", () => {
      it("should apply schema defaults to a cached device stored without vendor parsing", async () => {
        const defaultedSchemas = {
          properties: z.object({
            connection: z.object({ host: z.string() }).default({ host: "" }),
          }),
          make: z.string(),
          model: z.string(),
        };
        const rack = await client.racks.create({ name: "test" });
        const key = id.create();
        await client.devices.create({
          key,
          rack: rack.key,
          name: "generic_cached",
          location: "location",
          make: "some_make",
          model: "model",
          configured: true,
          properties: {},
        });
        const dev = await client.devices.retrieve({
          key,
          includeStatus: true,
          schemas: defaultedSchemas,
        });
        expect(dev.properties.connection).toEqual({ host: "" });
      });

      it("should fetch schema-parsed properties even when a generic copy is cached", async () => {
        const rack = await client.racks.create({ name: "test" });
        const dev = await client.devices.create(
          {
            key: id.create(),
            name: "generic_cached_device",
            rack: rack.key,
            location: "test",
            make: "custom_make",
            model: "test",
            properties: { sampleRate: 100, channels: {} },
          },
          schemas,
        );
        await client.devices.retrieve({ key: dev.key });
        const retrieved = await client.devices.retrieve({
          key: dev.key,
          includeStatus: true,
          schemas,
        });
        expect(retrieved.properties.sampleRate).toEqual(100);
        const cached = await client.devices.retrieve({ key: dev.key });
        expect(cached.properties).toEqual({ sampleRate: 100, channels: {} });
      });
    });

    describe("createForm", () => {
      it("should load and save device with typed properties", async () => {
        const rack = await client.racks.create({ name: "schema-form-rack" });
        const dev = await client.devices.create(
          {
            key: id.create(),
            name: "schema-form-device",
            rack: rack.key,
            location: "test",
            make: "custom_make",
            model: "test",
            properties: { sampleRate: 300, channels: { a: 1 } },
          },
          schemas,
        );

        const useForm = Device.createForm(schemas);
        const { result } = renderHook(() => useForm({ query: { key: dev.key } }), {
          wrapper,
        });

        await waitFor(() => {
          expect(result.current.form.value().name).toBe("schema-form-device");
        });

        expect(result.current.form.value().properties).toEqual({
          sampleRate: 300,
          channels: { a: 1 },
        });

        act(() => {
          result.current.form.set("name", "updated-schema-device");
        });

        await act(async () => {
          result.current.save();
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const retrieved = await client.devices.retrieve({ key: dev.key });
        expect(retrieved.name).toBe("updated-schema-device");
      });

      it("should not reset the form when a different device is set", async () => {
        const rack = await client.racks.create({ name: "schema-form-rack" });
        const dev = await client.devices.create(
          {
            key: id.create(),
            name: "target_form_device",
            rack: rack.key,
            location: "test",
            make: "custom_make",
            model: "test",
            properties: { sampleRate: 300, channels: { a: 1 } },
          },
          schemas,
        );

        const foreignKey = id.create();
        const foreignSeen = { current: false };
        const useForm = Device.createForm(schemas);
        const { result } = renderHook(
          () => {
            Device.useSetSynchronizer((changed) => {
              if (changed.key === foreignKey) foreignSeen.current = true;
            });
            return useForm({ query: { key: dev.key } });
          },
          { wrapper },
        );

        await waitFor(() => {
          expect(result.current.form.value().name).toBe("target_form_device");
        });

        await client.devices.create({
          key: foreignKey,
          name: "foreign_device",
          rack: rack.key,
          location: "elsewhere",
          make: "other_make",
          model: "other_model",
          properties: {},
        });
        await waitFor(() => {
          expect(foreignSeen.current).toBe(true);
        });

        expect(result.current.form.value().key).toBe(dev.key);
        expect(result.current.form.value().name).toBe("target_form_device");
        expect(result.current.form.value().properties).toEqual({
          sampleRate: 300,
          channels: { a: 1 },
        });
      });

      it("should keep current values when the device is set with schema-invalid properties", async () => {
        const rack = await client.racks.create({ name: "schema-form-rack" });
        const dev = await client.devices.create(
          {
            key: id.create(),
            name: "shape_guarded_device",
            rack: rack.key,
            location: "test",
            make: "custom_make",
            model: "test",
            properties: { sampleRate: 300, channels: { a: 1 } },
          },
          schemas,
        );

        const latestProperties = { current: undefined as record.Unknown | undefined };
        const useForm = Device.createForm(schemas);
        const { result } = renderHook(
          () => {
            Device.useSetSynchronizer((changed) => {
              if (changed.key === dev.key)
                latestProperties.current = changed.properties;
            });
            return useForm({ query: { key: dev.key } });
          },
          { wrapper },
        );

        await waitFor(() => {
          expect(result.current.form.value().name).toBe("shape_guarded_device");
        });

        await act(async () => {
          await client.devices.create({ ...dev, status: undefined, properties: {} });
        });
        await waitFor(() => {
          expect(latestProperties.current).toEqual({});
        });

        expect(result.current.form.value().properties).toEqual({
          sampleRate: 300,
          channels: { a: 1 },
        });
      });

      it("should reset the form when the device itself is updated", async () => {
        const rack = await client.racks.create({ name: "schema-form-rack" });
        const dev = await client.devices.create(
          {
            key: id.create(),
            name: "self_updating_device",
            rack: rack.key,
            location: "test",
            make: "custom_make",
            model: "test",
            properties: { sampleRate: 300, channels: { a: 1 } },
          },
          schemas,
        );

        const useForm = Device.createForm(schemas);
        const { result } = renderHook(() => useForm({ query: { key: dev.key } }), {
          wrapper,
        });

        await waitFor(() => {
          expect(result.current.form.value().name).toBe("self_updating_device");
        });

        await act(async () => {
          await client.devices.create(
            {
              ...dev,
              status: undefined,
              name: "renamed_device",
              properties: { sampleRate: 500, channels: {} },
            },
            schemas,
          );
        });

        await waitFor(() => {
          expect(result.current.form.value().name).toBe("renamed_device");
          expect(result.current.form.value().properties).toEqual({
            sampleRate: 500,
            channels: {},
          });
        });
      });
    });
  });
});
