// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type device } from "@synnaxlabs/client";
import { type Device, Flux } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  useCommonNetwork,
  useRetrieveObservable,
  useRetrieveSlave,
  useRetrieveSlaveStateful,
  useToggleEnabled,
} from "@/hardware/ethercat/device/queries";
import {
  MAKE,
  SLAVE_MODEL,
  type SlaveProperties,
  ZERO_SLAVE_PROPERTIES,
} from "@/hardware/ethercat/device/types";
import { type Channel } from "@/hardware/ethercat/task/types";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const createSlaveDevice = async (
  rackKey: number,
  properties: Partial<SlaveProperties> = {},
): Promise<device.Device<SlaveProperties>> => {
  const key = id.create();
  return await client.devices.create({
    key,
    name: properties.name ?? `EtherCAT Slave ${key}`,
    rack: rackKey,
    location: "test-location",
    make: MAKE,
    model: SLAVE_MODEL,
    properties: { ...ZERO_SLAVE_PROPERTIES, ...properties },
  });
};

describe("EtherCAT Device queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  let rack: { key: number };

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
    rack = await client.racks.create({ name: `test-ethercat-rack-${id.create()}` });
  });

  describe("useRetrieveSlave", () => {
    it("should retrieve a slave device by key", async () => {
      const dev = await createSlaveDevice(rack.key, {
        name: "Test Slave",
        network: "eth0",
        position: 1,
      });

      const { result } = renderHook(() => useRetrieveSlave({ key: dev.key }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.key).toEqual(dev.key);
      expect(result.current.data?.name).toEqual("Test Slave");
    });

    it("should update when device properties change", async () => {
      const dev = await createSlaveDevice(rack.key, {
        name: "Original Name",
        network: "eth0",
      });

      const { result } = renderHook(() => useRetrieveSlave({ key: dev.key }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.name).toEqual("Original Name");

      await act(async () => {
        await client.devices.create({
          ...dev,
          name: "Updated Name",
        });
      });

      await waitFor(() => {
        expect(result.current.data?.name).toEqual("Updated Name");
      });
    });

    it("should return proper SlaveProperties type with PDOs", async () => {
      const pdos = {
        inputs: [
          {
            name: "Status",
            index: 0x6000,
            subindex: 1,
            bitLength: 16,
            dataType: "uint16",
          },
        ],
        outputs: [
          {
            name: "Control",
            index: 0x7000,
            subindex: 1,
            bitLength: 16,
            dataType: "uint16",
          },
        ],
      };

      const dev = await createSlaveDevice(rack.key, {
        name: "PDO Device",
        network: "eth0",
        position: 2,
        pdos,
        enabled: true,
      });

      const { result } = renderHook(() => useRetrieveSlave({ key: dev.key }), {
        wrapper,
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.properties?.pdos?.inputs).toHaveLength(1);
      expect(result.current.data?.properties?.pdos?.outputs).toHaveLength(1);
      expect(result.current.data?.properties?.pdos?.inputs[0].name).toEqual("Status");
      expect(result.current.data?.properties?.enabled).toBe(true);
    });
  });

  describe("useRetrieveSlaveStateful", () => {
    it("should provide stateful retrieval with loading states", async () => {
      const dev = await createSlaveDevice(rack.key, { name: "Stateful Test" });

      const { result } = renderHook(() => useRetrieveSlaveStateful(), { wrapper });

      expect(result.current.variant).toEqual("loading");

      await act(async () => {
        result.current.retrieve({ key: dev.key });
      });

      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.key).toEqual(dev.key);
    });
  });

  describe("useRetrieveObservable", () => {
    it("should call onChange when device is retrieved", async () => {
      const dev = await createSlaveDevice(rack.key, {
        name: "Observable Test",
        network: "eth1",
      });

      const onChange = vi.fn();

      const { result } = renderHook(() => useRetrieveObservable({ onChange }), {
        wrapper,
      });

      await act(async () => {
        result.current.retrieve({ key: dev.key });
      });

      await waitFor(() => expect(onChange).toHaveBeenCalled());
      expect(onChange).toHaveBeenCalledWith(
        expect.objectContaining({
          data: expect.objectContaining({ key: dev.key }),
        }),
        expect.objectContaining({ key: dev.key }),
      );
    });

    it("should notify on subsequent device updates", async () => {
      const dev = await createSlaveDevice(rack.key, {
        name: "Observable Update Test",
        network: "eth2",
      });

      const onChange = vi.fn();

      const { result } = renderHook(() => useRetrieveObservable({ onChange }), {
        wrapper,
      });

      await act(async () => {
        result.current.retrieve({ key: dev.key });
      });

      await waitFor(() => expect(onChange).toHaveBeenCalledTimes(1));

      await act(async () => {
        await client.devices.create({
          ...dev,
          name: "Updated Observable Device",
        });
      });

      await waitFor(() => expect(onChange.mock.calls.length).toBeGreaterThanOrEqual(2));
    });
  });

  describe("useCommonNetwork", () => {
    it("should return network from first device in channels", async () => {
      const dev = await createSlaveDevice(rack.key, {
        name: "Network Test Device",
        network: "test-network",
      });

      const channels: Channel[] = [
        {
          type: "automatic",
          device: dev.key,
          pdo: "Status",
          channel: 1,
          enabled: true,
          key: id.create(),
          name: "Test Channel",
        },
      ];

      const { result } = renderHook(() => useCommonNetwork(channels), { wrapper });

      await waitFor(() => expect(result.current).toEqual("test-network"));
    });

    it("should return empty string when channels array is empty", async () => {
      const channels: Channel[] = [];

      const { result } = renderHook(() => useCommonNetwork(channels), { wrapper });

      expect(result.current).toEqual("");
    });

    it("should return empty string when no channels have devices", async () => {
      const channels: Channel[] = [
        {
          type: "automatic",
          device: "",
          pdo: "Status",
          channel: 1,
          enabled: true,
          key: id.create(),
          name: "No Device Channel",
        },
      ];

      const { result } = renderHook(() => useCommonNetwork(channels), { wrapper });

      expect(result.current).toEqual("");
    });

    it("should update when device's network changes", async () => {
      const dev = await createSlaveDevice(rack.key, {
        name: "Network Change Device",
        network: "original-network",
      });

      const channels: Channel[] = [
        {
          type: "automatic",
          device: dev.key,
          pdo: "Status",
          channel: 1,
          enabled: true,
          key: id.create(),
          name: "Test Channel",
        },
      ];

      const { result } = renderHook(() => useCommonNetwork(channels), { wrapper });

      await waitFor(() => expect(result.current).toEqual("original-network"));

      await act(async () => {
        await client.devices.create({
          ...dev,
          properties: { ...dev.properties, network: "updated-network" },
        });
      });

      await waitFor(() => expect(result.current).toEqual("updated-network"));
    });
  });

  describe("useToggleEnabled", () => {
    it("should toggle enabled from true to false", async () => {
      const dev = await createSlaveDevice(rack.key, {
        name: "Toggle Test Device",
        network: "eth0",
        enabled: true,
      });

      const { result } = renderHook(
        () => ({
          toggle: useToggleEnabled(),
          store: Flux.useStore<Device.FluxSubStore>(),
        }),
        { wrapper },
      );

      await act(async () => {
        await result.current.toggle.updateAsync({ keys: dev.key });
      });

      const updated = await client.devices.retrieve<SlaveProperties>({ key: dev.key });
      expect(updated.properties.enabled).toBe(false);
    });

    it("should toggle enabled from false to true", async () => {
      const dev = await createSlaveDevice(rack.key, {
        name: "Toggle False Device",
        network: "eth0",
        enabled: false,
      });

      const { result } = renderHook(
        () => ({
          toggle: useToggleEnabled(),
          store: Flux.useStore<Device.FluxSubStore>(),
        }),
        { wrapper },
      );

      await act(async () => {
        await result.current.toggle.updateAsync({ keys: dev.key });
      });

      const updated = await client.devices.retrieve<SlaveProperties>({ key: dev.key });
      expect(updated.properties.enabled).toBe(true);
    });

    it("should set explicit enabled value when provided", async () => {
      const dev = await createSlaveDevice(rack.key, {
        name: "Explicit Enable Device",
        network: "eth0",
        enabled: true,
      });

      const { result } = renderHook(
        () => ({
          toggle: useToggleEnabled(),
          store: Flux.useStore<Device.FluxSubStore>(),
        }),
        { wrapper },
      );

      await act(async () => {
        await result.current.toggle.updateAsync({ keys: dev.key, enabled: false });
      });

      const updated = await client.devices.retrieve<SlaveProperties>({ key: dev.key });
      expect(updated.properties.enabled).toBe(false);
    });

    it("should toggle multiple devices at once", async () => {
      const dev1 = await createSlaveDevice(rack.key, {
        name: "Multi Device 1",
        network: "eth0",
        enabled: true,
      });
      const dev2 = await createSlaveDevice(rack.key, {
        name: "Multi Device 2",
        network: "eth0",
        enabled: true,
      });

      const { result } = renderHook(
        () => ({
          toggle: useToggleEnabled(),
          store: Flux.useStore<Device.FluxSubStore>(),
        }),
        { wrapper },
      );

      await act(async () => {
        await result.current.toggle.updateAsync({ keys: [dev1.key, dev2.key] });
      });

      const updated1 = await client.devices.retrieve<SlaveProperties>({
        key: dev1.key,
      });
      const updated2 = await client.devices.retrieve<SlaveProperties>({
        key: dev2.key,
      });
      expect(updated1.properties.enabled).toBe(false);
      expect(updated2.properties.enabled).toBe(false);
    });

    it("should apply same enabled value to all devices", async () => {
      const dev1 = await createSlaveDevice(rack.key, {
        name: "Same Value Device 1",
        network: "eth0",
        enabled: true,
      });
      const dev2 = await createSlaveDevice(rack.key, {
        name: "Same Value Device 2",
        network: "eth0",
        enabled: false,
      });

      const { result } = renderHook(
        () => ({
          toggle: useToggleEnabled(),
          store: Flux.useStore<Device.FluxSubStore>(),
        }),
        { wrapper },
      );

      await act(async () => {
        await result.current.toggle.updateAsync({
          keys: [dev1.key, dev2.key],
          enabled: true,
        });
      });

      const updated1 = await client.devices.retrieve<SlaveProperties>({
        key: dev1.key,
      });
      const updated2 = await client.devices.retrieve<SlaveProperties>({
        key: dev2.key,
      });
      expect(updated1.properties.enabled).toBe(true);
      expect(updated2.properties.enabled).toBe(true);
    });
  });
});
