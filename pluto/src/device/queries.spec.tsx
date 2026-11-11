// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, NotFoundError, query, type rack, status } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
  spyOnSend,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { caseconv, id, type record } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Device } from "@/device";
import { Errors } from "@/errors";
import { type Flux } from "@/flux";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const path = Device.formPath;

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("use", () => {
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
      const { result } = await renderHookSuspended(() => Device.use({ key: dev.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current?.key).toEqual(dev.key);
    });

    it("should pass an identity-stable query to the client on every render", async () => {
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
      // A suspended mount discards its memoized query, so the status is cached first.
      await client.devices.retrieve({ key: dev.key, includeStatus: true });
      const spy = vi.spyOn(client.devices, "watch");
      const { result, rerender } = await renderHookSuspended(
        () => Device.use({ key: dev.key }),
        { wrapper },
      );
      await waitFor(() => expect(result.current?.key).toEqual(dev.key));
      rerender();
      rerender();
      // One query object across every snapshot read is what keys the client's
      // route and hash memos; a fresh object per render re-parses each time.
      const queries = new Set(spy.mock.calls.map((call) => call[0]));
      expect(queries.size).toBe(1);
      spy.mockRestore();
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
      const { result } = await renderHookSuspended(() => Device.use({ key: dev.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current?.key).toEqual(dev.key);
      await act(async () => {
        await client.devices.rename(dev.key, "test2");
      });
      await waitFor(() => {
        expect(result.current?.name).toEqual("test2");
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
      const { result } = await renderHookSuspended(() => Device.use({ key: dev.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current?.key).toEqual(dev.key);
      const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
        owner: device.resourceID(dev.key),
        variant: "success",
        message: "Device is happy as a clam",
        details: {
          rack: rack.key,
          connected: true,
        },
      });
      await client.statuses.set(devStatus);
      await waitFor(() => {
        expect(result.current?.status?.variant).toEqual("success");
        expect(result.current?.status?.owner).toEqual(device.resourceID(dev.key));
        expect(result.current?.status?.message).toEqual("Device is happy as a clam");
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
      const { result } = await renderHookSuspended(() => Device.use({ key: dev.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current).not.toBeNull());
      const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
        owner: device.resourceID(dev.key),
        variant: "success",
        message: "Device is connected",
        details: {
          rack: rack.key,
          connected: true,
        },
      });
      await client.statuses.set(devStatus);
      await waitFor(() => {
        expect(result.current?.status?.variant).toEqual("success");
      });
      await act(async () => {
        await client.devices.rename(dev.key, "updated-name");
      });
      await waitFor(() => {
        expect(result.current?.name).toEqual("updated-name");
        expect(result.current?.status).not.toBeUndefined();
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
      const { result: result1 } = await renderHookSuspended(
        () => Device.use({ key: dev.key }),
        { wrapper },
      );
      await waitFor(() => expect(result1.current).not.toBeNull());
      expect(result1.current?.key).toEqual(dev.key);
      expect(result1.current?.status).toBeDefined();
      await act(async () => {
        await client.statuses.set(
          status.create<typeof device.statusDetailsZ>({
            owner: device.resourceID(dev.key),
            variant: "success",
            message: "Device is happy as a clam",
            details: { rack: rack.key, connected: true },
          }),
        );
      });
      const { result: result2 } = await renderHookSuspended(
        () => Device.use({ key: dev.key }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result2.current).not.toBeNull();
        expect(result2.current?.status?.variant).toEqual("success");
        expect(result2.current?.status?.message).toEqual("Device is happy as a clam");
      });
    });

    describe("device cached without a status", () => {
      const schemas = {
        properties: z.record(z.string(), z.unknown()),
        make: z.string(),
        model: z.string(),
      };

      const createDevice = async (name: string) => {
        const { key: rack } = await client.racks.create({ name });
        const dev: device.New = {
          key: id.create(),
          name,
          rack,
          location: "test",
          make: "test",
          model: "test",
          properties: {},
        };
        await client.devices.create(dev);
        return dev;
      };

      it("should fetch the status after a status-free list read", async () => {
        const dev = await createDevice("status-free-read");
        const local = createTestClient();
        // A filter on more than the key reads past the table, so no status comes.
        await local.devices.retrieve({
          where: { key: { in: [dev.key] }, rack: { eq: dev.rack } },
        });
        expect(
          local.devices.watch({ key: dev.key, includeStatus: true }).getCached(),
        ).toBeUndefined();
        const { result } = await renderHookSuspended(
          () => Device.use({ key: dev.key }),
          { wrapper: await createAsyncSynnaxWrapper({ client: local }) },
        );
        expect(result.current?.status?.owner).toEqual(device.resourceID(dev.key));
      });

      it("should fetch the status after a device arrives on the stream", async () => {
        const dev = await createDevice("stream-cached");
        const local = createTestClient();
        const stop = local.devices
          .watch({ where: { key: { in: [dev.key] }, rack: { eq: dev.rack } } })
          .onChange(() => {});
        await waitForStreamLive(local.connection);
        // An edit to properties writes no status and fetches nothing.
        await client.devices.dispatch(
          dev.key,
          device.set({ path: ["properties", "edited"], value: true }),
        );
        await waitFor(() => {
          const cached = local.devices.watch(dev.key).getCached();
          expect(query.isLive(cached) ? cached.properties.edited : undefined).toBe(
            true,
          );
        });
        expect(
          local.devices.watch({ key: dev.key, includeStatus: true }).getCached(),
        ).toBeUndefined();
        const { result } = await renderHookSuspended(
          () => Device.use({ key: dev.key }),
          { wrapper: await createAsyncSynnaxWrapper({ client: local }) },
        );
        expect(result.current?.status?.owner).toEqual(device.resourceID(dev.key));
        stop();
      });

      it("should fetch the status for a subscriber with no fetch", async () => {
        const dev = await createDevice("subscribed");
        const local = createTestClient();
        await local.devices.retrieve(dev.key);
        const { result } = renderHook(() => Device.useResult({ key: dev.key }), {
          wrapper: await createAsyncSynnaxWrapper({ client: local }),
        });
        await waitFor(() =>
          expect(result.current.data?.status?.owner).toEqual(
            device.resourceID(dev.key),
          ),
        );
      });

      it("should fetch the status for a hook with schemas", async () => {
        const dev = await createDevice("typed");
        const local = createTestClient();
        await local.devices.retrieve(dev.key);
        const { use } = Device.createRetrieve(schemas);
        const { result } = await renderHookSuspended(() => use({ key: dev.key }), {
          wrapper: await createAsyncSynnaxWrapper({ client: local }),
        });
        expect(result.current?.status?.owner).toEqual(device.resourceID(dev.key));
      });

      it("should fetch once when the device has no status row", async () => {
        const dev = await createDevice("no-status-row");
        const [owned] = await client.statuses.retrieve({
          owners: [device.resourceID(dev.key)],
        });
        await client.statuses.delete(owned.key);
        const local = createTestClient();
        await local.devices.retrieve(dev.key);
        const localWrapper = await createAsyncSynnaxWrapper({ client: local });
        const send = spyOnSend(local);
        const render = async () =>
          await renderHookSuspended(() => Device.use({ key: dev.key }), {
            wrapper: localWrapper,
          });
        const first = await render();
        expect(first.result.current?.status).toBeUndefined();
        first.unmount();
        const second = await render();
        expect(second.result.current?.key).toEqual(dev.key);
        const retrieves = send.mock.calls.filter(
          ([target]) => target === "/device/retrieve",
        );
        expect(retrieves).toHaveLength(1);
      });

      it("should settle a hook with schemas when the device has no status row", async () => {
        const dev = await createDevice("typed-no-status-row");
        const [owned] = await client.statuses.retrieve({
          owners: [device.resourceID(dev.key)],
        });
        await client.statuses.delete(owned.key);
        const local = createTestClient();
        await local.devices.retrieve(dev.key);
        const { use } = Device.createRetrieve(schemas);
        const { result } = await renderHookSuspended(() => use({ key: dev.key }), {
          wrapper: await createAsyncSynnaxWrapper({ client: local }),
        });
        expect(result.current?.key).toEqual(dev.key);
        await act(async () => {
          await client.devices.dispatch(
            dev.key,
            device.set({ path: ["location"], value: "updated" }),
          );
        });
        await waitFor(() => expect(result.current?.location).toEqual("updated"));
      });
    });
  });

  describe("useList", () => {
    const createDevice = async (): Promise<device.New> => {
      const rack = await client.racks.create({ name: "test" });
      const dev: device.New = {
        key: id.create(),
        name: id.create(),
        rack: rack.key,
        location: "location",
        make: "make",
        model: "model",
        properties: {},
      };
      await client.devices.create(dev);
      return dev;
    };

    const setStatus = async (dev: device.New, variant: status.Variant) =>
      await client.statuses.set(
        status.create<typeof device.statusDetailsZ>({
          owner: device.resourceID(dev.key),
          variant,
          message: variant,
          details: { rack: dev.rack, connected: variant === "success" },
        }),
      );

    it("should update the status of a device in the list", async () => {
      const dev = await createDevice();
      const { result } = renderHook(
        () => Device.useList({ query: { where: { name: { eq: dev.name } } } }),
        { wrapper },
      );
      act(() => result.current.more());
      await waitFor(() => expect(result.current.data).toEqual([dev.key]));
      await setStatus(dev, "error");
      await waitFor(() =>
        expect(result.current.getItem(dev.key)?.status?.variant).toEqual("error"),
      );
    });

    it("should keep the status of a device outside the list across a change", async () => {
      const dev = await createDevice();
      await setStatus(dev, "success");
      const missing = id.create();
      const { result } = renderHook(
        () => Device.useList({ query: { where: { name: { eq: missing } } } }),
        { wrapper },
      );
      const callback = vi.fn();
      const release = result.current.subscribe(callback, dev.key);
      await waitFor(() => expect(callback).toHaveBeenCalled());
      expect(result.current.getItem(dev.key)?.status?.variant).toEqual("success");
      await act(async () => {
        await client.devices.rename(dev.key, "updated");
      });
      await waitFor(() =>
        expect(result.current.getItem(dev.key)?.name).toEqual("updated"),
      );
      expect(result.current.getItem(dev.key)?.status?.variant).toEqual("success");
      release();
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
      const retrieved = await client.devices.retrieve(key);
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
      const retrieved = await client.devices.retrieve(dev.key);
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
      await expect(client.devices.retrieve(dev.key)).rejects.toThrow(NotFoundError);
    });
  });

  describe("useMultiple", () => {
    const createDevice = async (make: string) => {
      const rack = await client.racks.create({ name: "test" });
      return await client.devices.create({
        key: id.create(),
        name: "test",
        rack: rack.key,
        location: "test",
        make,
        model: "test",
        properties: {},
      });
    };

    it("should add a device of the make created after the query resolves", async () => {
      const make = id.create();
      const first = await createDevice(make);
      const { result } = await renderHookSuspended(
        () => Device.useMultiple({ where: { make: { in: [make] } } }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current?.map((d) => d.key)).toEqual([first.key]),
      );
      const second = await createDevice(make);
      await waitFor(() =>
        expect(result.current?.map((d) => d.key).sort()).toEqual(
          [first.key, second.key].sort(),
        ),
      );
    });

    it("should reflect a device configured after the query resolves", async () => {
      const make = id.create();
      const dev = await createDevice(make);
      const { result } = await renderHookSuspended(
        () => Device.useMultiple({ where: { make: { in: [make] } } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current?.[0]?.configured).toBe(false));
      await client.devices.dispatch(
        dev.key,
        device.set({ path: ["configured"], value: true }),
      );
      await waitFor(() => expect(result.current?.[0]?.configured).toBe(true));
    });
  });

  describe("useResultAncestors", () => {
    let rackKey: rack.Key;
    beforeAll(async () => {
      rackKey = (await client.racks.create({ name: "test" })).key;
    });

    const create = async (name: string, parent?: device.Key) =>
      await client.devices.create({
        key: id.create(),
        name,
        rack: rackKey,
        location: "test",
        make: "test",
        model: "test",
        properties: {},
        parent,
      });

    it("should keep the devices and the devices above them live", async () => {
      const top = await create("top");
      const mid = await create("mid", top.key);
      const leaf = await create("leaf", mid.key);
      const keysOf = (devices?: device.Device[]): device.Key[] =>
        (devices ?? []).map(({ key }) => key).sort();
      const { result } = renderHook(
        () => Device.useResultAncestors({ keys: [leaf.key] }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(keysOf(result.current.data)).toEqual(
          [leaf.key, mid.key, top.key].sort(),
        );
      });
      await act(async () => {
        await client.devices.rename(top.key, "renamed");
      });
      await waitFor(() =>
        expect(result.current.data).toContainEqual(
          expect.objectContaining({ key: top.key, name: "renamed" }),
        ),
      );
    });

    it("should fetch a device that joins the chain uncached", async () => {
      const chassis = await create("chassis");
      const leaf = await create("leaf", chassis.key);
      const holder = await create("holder");
      const reader = createTestClient();
      const { result } = renderHook(
        () => Device.useResultAncestors({ keys: [leaf.key] }),
        { wrapper: await createAsyncSynnaxWrapper({ client: reader }) },
      );
      await waitFor(() => expect(result.current.data).toHaveLength(2));
      await client.devices.dispatch(
        chassis.key,
        device.setRack({ rack: rackKey, parent: holder.key }),
      );
      await waitFor(() =>
        expect(result.current.data?.map(({ key }) => key).sort()).toEqual(
          [leaf.key, chassis.key, holder.key].sort(),
        ),
      );
    });
  });

  describe("result selectors", () => {
    const createDevice = async () => {
      const rack = await client.racks.create({ name: "test" });
      return await client.devices.create({
        key: id.create(),
        name: "test",
        rack: rack.key,
        location: "dev1",
        make: id.create(),
        model: "test",
        properties: {},
      });
    };

    it("should read the make of a device", async () => {
      const dev = await createDevice();
      const { result } = renderHook(() => Device.useResultMake({ key: dev.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.data).toEqual(dev.make));
    });

    it("should follow a change to the location of a device", async () => {
      const dev = await createDevice();
      const { result } = renderHook(() => Device.useResultLocation({ key: dev.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.data).toEqual("dev1"));
      await act(async () => {
        await client.devices.dispatch(
          dev.key,
          device.set({ path: ["location"], value: "dev2" }),
        );
      });
      await waitFor(() => expect(result.current.data).toEqual("dev2"));
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
        key: [dev1.key, dev2.key],
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

      await client.devices.retrieve(dev1.key);

      const devices = await client.devices.retrieve({
        key: [dev1.key, dev2.key],
        includeStatus: true,
      });

      expect(devices).toHaveLength(2);
      expect(devices.map((d) => d.key)).toContain(dev1.key);
      expect(devices.map((d) => d.key)).toContain(dev2.key);
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

      await client.devices.retrieve({ key: [dev1.key, dev2.key] });

      const devices = await client.devices.retrieve({
        key: [dev1.key, dev2.key],
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
        key: [dev.key],
        includeStatus: true,
      });

      expect(devices).toHaveLength(1);
      expect(devices[0].status).toBeDefined();
      expect(devices[0].status?.owner).toEqual(device.resourceID(dev.key));
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

      await client.devices.retrieve(dev.key);

      const devices = await client.devices.retrieve({
        key: [dev.key],
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
        const { result } = renderHook(() => Device.useForm({ query: null }), {
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
        const { result } = renderHook(() => useForm({ query: null }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        act(() => {
          result.current.form.set(path.rack, rack.key);
          result.current.form.set(path.name, "Test Form Device");
          result.current.form.set(path.make, "TestMake");
          result.current.form.set(path.model, "TestModel");
          result.current.form.set(path.location, "Lab1");
        });

        await act(async () => {
          result.current.save();
        });

        await waitFor(() => {
          expect(result.current.variant).toBe("success");
        });

        const key = result.current.form.get(path.key).value;
        // parent and status are cache enrichments whose presence depends on
        // which relationship queries have already run; don't pin them.
        const {
          parent: _parent,
          status: _status,
          ...retrieved
        } = await client.devices.retrieve(key);
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

      it("should create the device on a save after a failed create", async () => {
        const rack = await client.racks.create({ name: "test form rack" });
        const removed = await client.racks.create({ name: "removed form rack" });
        await client.racks.delete(removed.key);
        const useForm = Device.createForm();
        const { result } = renderHook(() => useForm({ query: null }), { wrapper });
        await waitFor(() => expect(result.current.variant).toBe("success"));
        act(() => {
          result.current.form.set(path.rack, removed.key);
          result.current.form.set(path.name, "Retried Device");
          result.current.form.set(path.make, "TestMake");
          result.current.form.set(path.model, "TestModel");
          result.current.form.set(path.location, "Lab1");
        });
        await act(async () => expect(await result.current.saveAsync()).toBe(false));
        expect(result.current.status.description).toEqual(
          `rack: rack ${removed.key} does not exist: validation error`,
        );
        act(() => result.current.form.set(path.rack, rack.key));
        await act(async () => expect(await result.current.saveAsync()).toBe(true));
        const key = result.current.form.get(path.key).value;
        const stored = await createTestClient().devices.retrieve(key);
        expect(stored).toMatchObject({ key, name: "Retried Device", rack: rack.key });
      });

      it("should validate required fields", async () => {
        const useForm = Device.createForm();
        const { result } = renderHook(() => useForm({ query: null }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        await act(async () => {
          result.current.save();
        });

        const nameField = result.current.form.get(path.name);
        expect(nameField.status.message).toBe("name is required");
        const makeField = result.current.form.get(path.make);
        expect(makeField.status.message).toBe("make is required");
        const modelField = result.current.form.get(path.model);
        expect(modelField.status.message).toBe("model is required");
        const locationField = result.current.form.get(path.location);
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
        const useForm = Device.createForm(
          { properties: customPropertiesZ, make: z.string(), model: z.string() },
          {
            key: "",
            rack: 0,
            name: "",
            make: "",
            model: "",
            location: "",
            configured: true,
            properties: { serialNumber: "", calibrationDate: "" },
          },
        );
        const { result } = renderHook(() => useForm({ query: null }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        const customProps: z.infer<typeof customPropertiesZ> = {
          serialNumber: "SN123456",
          calibrationDate: "2024-01-01",
        };

        act(() => {
          result.current.form.set(path.rack, rack.key);
          result.current.form.set(path.name, "Custom Device");
          result.current.form.set(path.make, "CustomMake");
          result.current.form.set(path.model, "CustomModel");
          result.current.form.set(path.location, "Lab2");
          result.current.form.set(path.properties, customProps);
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
        const { result } = await renderHookSuspended(
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

        const { result } = await renderHookSuspended(
          () => Device.useForm({ query: { key: testDevice.key } }),
          { wrapper },
        );

        await waitFor(() => {
          const formData = result.current.form.value();
          expect(formData.name).toBe("Device to Update");
        });

        act(() => {
          result.current.form.set(path.name, "Updated Device Name");
          result.current.form.set(path.location, "Lab4");
        });

        await act(async () => {
          result.current.save();
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        await waitFor(async () => {
          const updatedDevice = await client.devices.retrieve(testDevice.key);
          expect(updatedDevice.name).toBe("Updated Device Name");
          expect(updatedDevice.location).toBe("Lab4");
        });
      });

      it("should move an existing device to another rack", async () => {
        const rack = await client.racks.create({ name: "test move rack" });
        const target = await client.racks.create({ name: "test move target rack" });
        const testDevice = await client.devices.create({
          key: id.create(),
          rack: rack.key,
          name: "Device to Move",
          make: "make",
          model: "model",
          location: "Lab3",
          properties: { port: 1 },
        });
        const { result } = await renderHookSuspended(
          () => Device.useForm({ query: { key: testDevice.key } }),
          { wrapper },
        );
        await waitFor(() =>
          expect(result.current.form.value().name).toBe("Device to Move"),
        );
        act(() => result.current.form.set(path.rack, target.key));
        await act(async () => {
          await result.current.saveAsync();
        });
        expect(result.current.variant).toBe("success");
        await waitFor(async () => {
          const moved = await client.devices.retrieve(testDevice.key);
          expect(moved.rack).toBe(target.key);
          expect(moved.name).toBe("Device to Move");
          expect(moved.properties).toEqual({ port: 1 });
        });
      });

      it("should keep typed values and show the error on a refused save", async () => {
        const rack = await client.racks.create({ name: "test refused save rack" });
        const testDevice = await client.devices.create({
          key: id.create(),
          rack: rack.key,
          name: "Refused Device",
          make: "make",
          model: "model",
          location: "Lab6",
          properties: {},
        });
        const viewer = await createTestClientWithRole(client, "Viewer");
        const { result } = await renderHookSuspended(
          () => Device.useForm({ query: { key: testDevice.key } }),
          { wrapper: await createAsyncSynnaxWrapper({ client: viewer }) },
        );
        act(() => result.current.form.set(path.name, "Typed Name"));
        let saved = true;
        await act(async () => {
          saved = await result.current.saveAsync();
        });
        expect(saved).toBe(false);
        expect(result.current.variant).toBe("error");
        expect(result.current.form.value().name).toBe("Typed Name");
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

        const { result } = await renderHookSuspended(
          () => Device.useForm({ query: { key: testDevice.key } }),
          { wrapper },
        );

        await waitFor(() => {
          expect(result.current.form.value().name).toBe("Status Sync Device");
        });

        const devStatus: device.Status = status.create<typeof device.statusDetailsZ>({
          owner: device.resourceID(testDevice.key),
          variant: "warning",
          message: "Device is degraded",
          details: { rack: rack.key, connected: true },
        });
        await client.statuses.set(devStatus);

        await waitFor(() => {
          expect(result.current.form.value().status?.message).toBe(
            "Device is degraded",
          );
          expect(result.current.form.value().status?.variant).toBe("warning");
        });
      });

      const renderLoadedForm = async (
        beforeSave?: Flux.UseFormParams<
          Device.FormQuery,
          typeof Device.formSchema
        >["beforeSave"],
      ) => {
        const rack = await client.racks.create({ name: "test form status rack" });
        const testDevice = await client.devices.create({
          key: id.create(),
          rack: rack.key,
          name: "Form Status Device",
          make: "make",
          model: "model",
          location: "Lab6",
          properties: {},
        });
        await client.statuses.set(
          status.create<typeof device.statusDetailsZ>({
            owner: device.resourceID(testDevice.key),
            name: testDevice.name,
            message: "Device online",
            variant: "success",
            details: { rack: rack.key, connected: true },
          }),
        );
        const { result } = renderHook(
          () => Device.useForm({ query: { key: testDevice.key }, beforeSave }),
          { wrapper },
        );
        await waitFor(() =>
          expect(result.current.form.value().status?.message).toBe("Device online"),
        );
        return { result, testDevice };
      };

      it("should rename the stored status instead of writing back the one it read", async () => {
        const { result, testDevice } = await renderLoadedForm();
        act(() => result.current.form.set(path.name, "Renamed Status Device"));
        await act(async () => {
          await result.current.saveAsync();
        });
        expect(result.current.variant).toBe("success");
        await waitFor(async () => {
          const { name, status: stored } = await client.devices.retrieve({
            key: testDevice.key,
            includeStatus: true,
          });
          expect(name).toBe("Renamed Status Device");
          expect(stored).toMatchObject({
            name: "Renamed Status Device",
            message: "Device online",
          });
        });
      });

      it("should write a status that a beforeSave hook reports", async () => {
        const { result, testDevice } = await renderLoadedForm(async ({ get, set }) => {
          set(
            path.status,
            status.create<typeof device.statusDetailsZ>({
              name: get(path.name).value,
              message: "Device connected",
              variant: "success",
              details: {
                rack: get(path.rack).value,
                connected: true,
              },
            }),
          );
          return true;
        });
        act(() => result.current.form.set(path.name, "Connected Device"));
        await act(async () => {
          await result.current.saveAsync();
        });
        expect(result.current.variant).toBe("success");
        await waitFor(async () => {
          const { status: stored } = await client.devices.retrieve({
            key: testDevice.key,
            includeStatus: true,
          });
          expect(stored?.message).toBe("Device connected");
        });
      });
    });

    describe("validation", () => {
      it("should validate name field", async () => {
        const { result } = renderHook(() => Device.useForm({ query: null }), {
          wrapper,
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        act(() => {
          result.current.form.set(path.name, "");
        });

        const isValid = result.current.form.validate(path.name);
        expect(isValid).toBe(false);

        const msg = result.current.form.get(path.name).status.message;
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
    const SCHEMA_VALUES = {
      key: "",
      rack: 0,
      name: "",
      make: "custom_make" as const,
      model: "",
      location: "",
      configured: true,
      properties: { sampleRate: 0, channels: {} },
    };
    const schemas = {
      properties: propertiesSchema,
      make: makeSchema,
      model: modelSchema,
    };
    const setProperties = (properties: z.infer<typeof propertiesSchema>) =>
      device.set({
        path: ["properties"],
        value: caseconv.camelToSnake(properties, { schema: propertiesSchema }),
      });

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

        const { use } = Device.createRetrieve(schemas);
        const { result } = await renderHookSuspended(() => use({ key: dev.key }), {
          wrapper,
        });

        await waitFor(() => expect(result.current).not.toBeNull());
        expect(result.current?.properties.sampleRate).toBe(1000);
        expect(result.current?.properties.channels).toEqual({ ai0: 1, ai1: 2 });
        expect(result.current?.make).toBe("custom_make");
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

        const { use } = Device.createRetrieve(schemas);
        const { result } = await renderHookSuspended(() => use({ key: dev.key }), {
          wrapper,
        });

        await waitFor(() => expect(result.current).not.toBeNull());
        expect(result.current?.properties.sampleRate).toBe(100);

        await act(async () => {
          await client.devices.dispatch(
            dev.key,
            setProperties({ sampleRate: 500, channels: { ch1: 10 } }),
          );
        });

        await waitFor(() => {
          expect(result.current?.properties.sampleRate).toBe(500);
          expect(result.current?.properties.channels).toEqual({ ch1: 10 });
        });
      });
    });

    // Other consumers write untyped records into the client cache; a typed hook must
    // parse them, or a form reads a property group the vendor's defaults would fill.
    describe("untyped records in the cache", () => {
      const defaultedSchemas = {
        properties: z.object({
          sampleRate: z.number().default(100),
          channels: z.record(z.string(), z.number()).default(() => ({})),
        }),
        make: makeSchema,
        model: modelSchema,
      };
      const createUntyped = async () => {
        const rack = await client.racks.create({ name: "untyped-rack" });
        const dev = await client.devices.create({
          key: id.create(),
          name: "untyped-device",
          rack: rack.key,
          location: "test",
          make: "custom_make",
          model: "test",
          properties: {},
        });
        await client.devices.retrieve(dev.key);
        return dev;
      };

      it("should fill schema defaults on a record cached untyped", async () => {
        const dev = await createUntyped();
        const { use } = Device.createRetrieve(defaultedSchemas);
        const { result } = await renderHookSuspended(() => use({ key: dev.key }), {
          wrapper,
        });
        await waitFor(() => expect(result.current?.properties.sampleRate).toBe(100));
        expect(result.current?.properties.channels).toEqual({});
      });

      it("should keep schema defaults across an untyped streamed update", async () => {
        const dev = await createUntyped();
        const { use } = Device.createRetrieve(defaultedSchemas);
        const { result } = await renderHookSuspended(() => use({ key: dev.key }), {
          wrapper,
        });
        await waitFor(() => expect(result.current?.properties.sampleRate).toBe(100));
        await act(async () => {
          await client.devices.dispatch(
            dev.key,
            device.set({ path: ["name"], value: "renamed" }),
          );
        });
        await waitFor(() => expect(result.current?.name).toBe("renamed"));
        expect(result.current?.properties.sampleRate).toBe(100);
      });

      it("should fill schema defaults on records cached untyped by keys", async () => {
        const first = await createUntyped();
        const second = await createUntyped();
        const { use } = Device.createRetrieveMultiple(defaultedSchemas);
        const { result } = await renderHookSuspended(
          () => use({ where: { key: { in: [first.key, second.key] } } }),
          { wrapper },
        );
        await waitFor(() => expect(result.current).toHaveLength(2));
        expect(result.current.map(({ properties }) => properties.sampleRate)).toEqual([
          100, 100,
        ]);
      });
    });

    describe("createRetrieveMultiple", () => {
      const createDevice = async (rack: number, sampleRate: number) =>
        await client.devices.create(
          {
            key: id.create(),
            name: "schema-multi-device",
            rack,
            location: "test",
            make: "custom_make",
            model: "test",
            properties: { sampleRate, channels: {} },
          },
          schemas,
        );

      it("should retrieve the devices the keys name with typed properties", async () => {
        const rack = await client.racks.create({ name: "schema-multi-rack" });
        const first = await createDevice(rack.key, 100);
        const second = await createDevice(rack.key, 200);
        const { use } = Device.createRetrieveMultiple(schemas);
        const { result } = await renderHookSuspended(
          () => use({ where: { key: { in: [first.key, second.key] } } }),
          { wrapper },
        );
        await waitFor(() => expect(result.current).toHaveLength(2));
        const rates = result.current.map(({ properties }) => properties.sampleRate);
        expect(rates.sort()).toEqual([100, 200]);
      });

      it("should update when one of the devices changes", async () => {
        const rack = await client.racks.create({ name: "schema-multi-update-rack" });
        const first = await createDevice(rack.key, 100);
        const second = await createDevice(rack.key, 200);
        const { use } = Device.createRetrieveMultiple(schemas);
        const { result } = await renderHookSuspended(
          () => use({ where: { key: { in: [first.key, second.key] } } }),
          { wrapper },
        );
        await waitFor(() => expect(result.current).toHaveLength(2));
        await act(async () => {
          await client.devices.dispatch(
            second.key,
            setProperties({ sampleRate: 500, channels: {} }),
          );
        });
        await waitFor(() => {
          const updated = result.current.find(({ key }) => key === second.key);
          expect(updated?.properties.sampleRate).toBe(500);
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
        await client.devices.retrieve(dev.key);
        const retrieved = await client.devices.retrieve({
          key: dev.key,
          includeStatus: true,
          schemas,
        });
        expect(retrieved.properties.sampleRate).toEqual(100);
        const cached = await client.devices.retrieve(dev.key);
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

        const useForm = Device.createForm(schemas, SCHEMA_VALUES);
        const { result } = await renderHookSuspended(
          () => useForm({ query: { key: dev.key } }),
          {
            wrapper,
          },
        );

        await waitFor(() => {
          expect(result.current.form.value().name).toBe("schema-form-device");
        });

        expect(result.current.form.value().properties).toEqual({
          sampleRate: 300,
          channels: { a: 1 },
        });

        act(() => {
          result.current.form.set(path.name, "updated-schema-device");
        });

        await act(async () => {
          result.current.save();
        });

        await waitFor(() => expect(result.current.variant).toBe("success"));

        await waitFor(async () => {
          const retrieved = await client.devices.retrieve(dev.key);
          expect(retrieved.name).toBe("updated-schema-device");
        });
      });

      it("should save edited typed properties", async () => {
        const rack = await client.racks.create({ name: "schema-form-rack" });
        const dev = await client.devices.create(
          {
            key: id.create(),
            name: "schema-form-properties-device",
            rack: rack.key,
            location: "test",
            make: "custom_make",
            model: "test",
            properties: { sampleRate: 300, channels: { portA: 1 } },
          },
          schemas,
        );
        const useForm = Device.createForm(schemas, SCHEMA_VALUES);
        const { result } = await renderHookSuspended(
          () => useForm({ query: { key: dev.key } }),
          { wrapper },
        );
        await waitFor(() =>
          expect(result.current.form.value().name).toBe(
            "schema-form-properties-device",
          ),
        );
        act(() =>
          result.current.form.set(path.properties, {
            sampleRate: 500,
            channels: { portA: 1, portB: 2 },
          }),
        );
        await act(async () => {
          await result.current.saveAsync();
        });
        expect(result.current.variant).toBe("success");
        await waitFor(async () => {
          const retrieved = await client.devices.retrieve({ key: dev.key, schemas });
          expect(retrieved.properties).toEqual({
            sampleRate: 500,
            channels: { portA: 1, portB: 2 },
          });
        });
      });

      it("should apply schema defaults to a generically cached device", async () => {
        const defaultedSchemas = {
          properties: z.object({
            connection: z.object({ host: z.string() }).default({ host: "" }),
          }),
          make: z.string(),
          model: z.string(),
        };
        const rack = await client.racks.create({ name: "schema-form-rack" });
        const dev = await client.devices.create({
          key: id.create(),
          name: "generic_cached_form_device",
          rack: rack.key,
          location: "test",
          make: "custom_make",
          model: "test",
          properties: {},
        });

        const useForm = Device.createForm(defaultedSchemas, {
          ...SCHEMA_VALUES,
          properties: { connection: { host: "" } },
        });
        const { result } = await renderHookSuspended(
          () => useForm({ query: { key: dev.key } }),
          {
            wrapper,
          },
        );

        await waitFor(() => {
          expect(result.current.form.value().name).toBe("generic_cached_form_device");
        });
        expect(result.current.form.value().properties).toEqual({
          connection: { host: "" },
        });
      });

      it("should not seed the form from a cached device that fails the schemas", async () => {
        const rack = await client.racks.create({ name: "schema-form-rack" });
        const dev = await client.devices.create({
          key: id.create(),
          name: "generic_cached_invalid_device",
          rack: rack.key,
          location: "test",
          make: "custom_make",
          model: "test",
          properties: {},
        });

        const useForm = Device.createForm(schemas, SCHEMA_VALUES);
        const Display = (): ReactElement => {
          useForm({ query: { key: dev.key } });
          return <div data-testid="loaded" />;
        };
        let utils!: ReturnType<typeof render>;
        await act(async () => {
          utils = render(
            <Errors.SuspenseBoundary
              loading={<div>loading</div>}
              FallbackComponent={() => <div data-testid="error" />}
            >
              <Display />
            </Errors.SuspenseBoundary>,
            { wrapper, onCaughtError: () => {} },
          );
        });
        await waitFor(() => expect(utils.queryByTestId("error")).not.toBeNull());
        expect(utils.queryByTestId("loaded")).toBeNull();
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
        const useForm = Device.createForm(schemas, SCHEMA_VALUES);
        const { result } = await renderHookSuspended(
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
        const useForm = Device.createForm(schemas, SCHEMA_VALUES);
        const { result } = await renderHookSuspended(
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
          await client.devices.dispatch(
            dev.key,
            device.set({ path: ["properties"], value: {} }),
          );
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

        const useForm = Device.createForm(schemas, SCHEMA_VALUES);
        const { result } = await renderHookSuspended(
          () => useForm({ query: { key: dev.key } }),
          {
            wrapper,
          },
        );

        await waitFor(() => {
          expect(result.current.form.value().name).toBe("self_updating_device");
        });

        await act(async () => {
          await client.devices.dispatch(dev.key, [
            device.set({ path: ["name"], value: "renamed_device" }),
            setProperties({ sampleRate: 500, channels: {} }),
          ]);
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
