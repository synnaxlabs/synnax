import { device, newTestClient } from "@synnaxlabs/client";
import { id, status } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/hardware/device";
import { newSynnaxWrapper } from "@/testutil/Synnax";

const client = newTestClient();

describe("queries", () => {
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
      const { result } = renderHook(
        () => Device.retrieve().useDirect({ params: { key: dev.key } }),
        { wrapper: newSynnaxWrapper(client) },
      );
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
      const { result } = renderHook(
        () => Device.retrieve().useDirect({ params: { key: dev.key } }),
        { wrapper: newSynnaxWrapper(client) },
      );
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
      const { result } = renderHook(
        () => Device.retrieve().useDirect({ params: { key: dev.key } }),
        { wrapper: newSynnaxWrapper(client) },
      );
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
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
      });
      act(() => {
        result.current.retrieve({ search: "special" });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(1);
      expect(
        result.current.data
          .map((d) => result.current.getItem(d)?.name)
          .includes("special"),
      ).toBeTruthy();
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
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
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
  });
});
