// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, device, NotFoundError } from "@synnaxlabs/client";
import { id, type record, status } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Device } from "@/device";
import { Flux } from "@/flux";
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
        () => ({
          device: Device.useRetrieve({ key: dev.key }),
          store: Flux.useStore<Device.FluxSubStore>(),
        }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result1.current.device.variant).toBeDefined());
      expect(result1.current.device.data?.key).toEqual(dev.key);
      result1.current.store.statuses.set(
        status.create<typeof device.statusDetailsZ>({
          key: device.statusKey(dev.key),
          variant: "success",
          message: "Device is happy as a clam",
          details: { rack: rack.key, device: dev.key },
        }),
      );
      const { result: result2 } = renderHook(
        () => Device.useRetrieve({ key: dev.key }),
        { wrapper },
      );
      await waitFor(() => expect(result2.current.variant).toEqual("success"));
      expect(result2.current.data?.status?.variant).toEqual("success");
      expect(result2.current.data?.status?.message).toEqual(
        "Device is happy as a clam",
      );
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
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(1);
      expect(
        result.current.data
          .map((d) => result.current.getItem(d)?.name)
          .includes("special"),
      ).toBe(true);
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

        await waitFor(() => expect(result.current.form.value().rack).not.toEqual(0));
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
        const retrieved = await client.devices.retrieve({ key });
        expect(retrieved).toEqual({
          key,
          name: "Test Form Device",
          make: "TestMake",
          model: "TestModel",
          location: "Lab1",
          rack: rack.key,
          configured: true,
          properties: {},
          status: undefined,
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
        expect(nameField.status.message).toBe("Name is required");
        const makeField = result.current.form.get("make");
        expect(makeField.status.message).toBe("Make is required");
        const modelField = result.current.form.get("model");
        expect(modelField.status.message).toBe("Model is required");
        const locationField = result.current.form.get("location");
        expect(locationField.status.message).toBe("Location is required");
      });

      it("should support custom properties", async () => {
        interface CustomProperties extends record.Unknown {
          serialNumber: string;
          calibrationDate: string;
        }

        const rack = await client.racks.create({
          name: "test custom props rack",
        });
        const useForm = Device.createForm<CustomProperties>();
        const { result } = renderHook(() => useForm({ query: { key: "" } }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const customProps: CustomProperties = {
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
        expect(msg).toEqual("Name is required");
      });
    });
  });
});
