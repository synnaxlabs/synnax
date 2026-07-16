// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, NotFoundError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Flux } from "@/flux";
import { Log } from "@/log";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const entry = (
  channel: number,
  overrides: Partial<log.ChannelEntry> = {},
): log.ChannelEntry =>
  log.channelEntryZ.parse({ channel, color: color.ZERO, ...overrides });

describe("log queries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createLog = async (overrides: Partial<log.New> = {}): Promise<log.Log> => {
    const proj = await client.projects.create({
      name: `log_ws_${uuid.create()}`,
      layout: {},
    });
    return await client.logs.create(proj.key, { name: "test_log", ...overrides });
  };

  const loadAndSelect = async <T>(
    key: log.Key,
    hook: () => T,
  ): Promise<ReturnType<typeof renderHook<T, unknown>>> => {
    const retrieve = renderHook(() => Log.useRetrieve({ key }), { wrapper });
    await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
    return renderHook(hook, { wrapper });
  };

  const loadAndCount = async <T>(key: log.Key, hook: () => T) => {
    const retrieve = renderHook(() => Log.useRetrieve({ key }), { wrapper });
    await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
    let renderCount = 0;
    const { result } = renderHook(
      () => {
        renderCount++;
        return hook();
      },
      { wrapper },
    );
    return { result, renderCount: () => renderCount };
  };

  describe("selectors", () => {
    it("useSelectName reads the name", async () => {
      const created = await createLog({ name: "Sensors" });
      const { result } = await loadAndSelect(created.key, () =>
        Log.useSelectName({ key: created.key }),
      );
      expect(result.current).toBe("Sensors");
    });

    it("useSelectChannels reads the channel entries", async () => {
      const created = await createLog({ channels: [entry(1), entry(2)] });
      const { result } = await loadAndSelect(created.key, () =>
        Log.useSelectChannels({ key: created.key }),
      );
      expect(result.current.map((e) => e.channel)).toEqual([1, 2]);
    });

    it("useSelectChannelKeys reads the ordered keys", async () => {
      const created = await createLog({ channels: [entry(3), entry(7)] });
      const { result } = await loadAndSelect(created.key, () =>
        Log.useSelectChannelKeys({ key: created.key }),
      );
      expect(result.current).toEqual([3, 7]);
    });

    it("useSelectChannelEntry reads a single entry", async () => {
      const created = await createLog({
        channels: [entry(1), entry(2, { alias: "pressure" })],
      });
      const { result } = await loadAndSelect(created.key, () =>
        Log.useSelectChannelEntry({ key: created.key, channel: 2 }),
      );
      expect(result.current?.alias).toBe("pressure");
    });

    it("useSelectChannelEntry returns null when the channel is absent", async () => {
      const created = await createLog({ channels: [entry(1)] });
      const { result } = await loadAndSelect(created.key, () =>
        Log.useSelectChannelEntry({ key: created.key, channel: 99 }),
      );
      expect(result.current).toBeNull();
    });

    it("reads the display-config scalars", async () => {
      const created = await createLog({
        timestampPrecision: 3,
        hideChannelNames: true,
        hideReceiptTimestamp: true,
      });
      const { result } = await loadAndSelect(created.key, () => ({
        precision: Log.useSelectTimestampPrecision({ key: created.key }),
        names: Log.useSelectHideChannelNames({ key: created.key }),
        receipt: Log.useSelectHideReceiptTimestamp({ key: created.key }),
      }));
      expect(result.current.precision).toBe(3);
      expect(result.current.names).toBe(true);
      expect(result.current.receipt).toBe(true);
    });
  });

  describe("memoization / reference stability", () => {
    it("useSelectChannelKeys keeps a stable reference when only config changes", async () => {
      const created = await createLog({ channels: [entry(1), entry(2)] });
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        keys: Log.useSelectChannelKeys({ key: created.key }),
        dispatch: Log.useDispatch(),
      }));
      const first = result.current.keys;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [log.setChannelEntry({ entry: entry(1, { alias: "x" }) })],
        });
      });
      expect(result.current.keys).toBe(first);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectChannelKeys yields a new value when the key set changes", async () => {
      const created = await createLog({ channels: [entry(1), entry(2)] });
      const { result } = await loadAndSelect(created.key, () => ({
        keys: Log.useSelectChannelKeys({ key: created.key }),
        dispatch: Log.useDispatch(),
      }));
      const first = result.current.keys;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [log.addChannel({ channel: 3 })],
        });
      });
      await waitFor(() => {
        expect(result.current.keys).not.toBe(first);
        expect(result.current.keys).toEqual([1, 2, 3]);
      });
    });

    it("useSelectChannelEntry keeps a stable reference when its entry object is unchanged", async () => {
      const created = await createLog({
        channels: [entry(1), entry(2, { alias: "shared" })],
      });
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        entry: Log.useSelectChannelEntry({ key: created.key, channel: 2 }),
        dispatch: Log.useDispatch(),
      }));
      const first = result.current.entry;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [log.setChannelEntry({ entry: entry(1, { alias: "y" }) })],
        });
      });
      expect(result.current.entry).toBe(first);
      expect(renderCount()).toEqual(countBefore);
    });

    it("useSelectTimestampPrecision is stable across unrelated edits", async () => {
      const created = await createLog({ timestampPrecision: 2 });
      const { result, renderCount } = await loadAndCount(created.key, () => ({
        precision: Log.useSelectTimestampPrecision({ key: created.key }),
        dispatch: Log.useDispatch(),
      }));
      const first = result.current.precision;
      const countBefore = renderCount();
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [log.setHideChannelNames({ hideChannelNames: true })],
        });
      });
      expect(result.current.precision).toBe(first);
      expect(renderCount()).toEqual(countBefore);
    });
  });

  describe("useRetrieve", () => {
    it("should retrieve a log by key", async () => {
      const created = await createLog({ name: "retrieve_test" });
      const { result } = renderHook(() => Log.useRetrieve({ key: created.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.key).toEqual(created.key);
      expect(result.current.data?.name).toEqual("retrieve_test");
    });

    it("should cache retrieved logs", async () => {
      const created = await createLog({ name: "cached_log" });
      const { result: r1 } = renderHook(() => Log.useRetrieve({ key: created.key }), {
        wrapper,
      });
      await waitFor(() => expect(r1.current.variant).toEqual("success"));
      const { result: r2 } = renderHook(() => Log.useRetrieve({ key: created.key }), {
        wrapper,
      });
      await waitFor(() => expect(r2.current.variant).toEqual("success"));
      expect(r2.current.data).toEqual(r1.current.data);
    });
  });

  describe("useCreate", () => {
    it("should create a new log and store it in the flux store", async () => {
      const project = await client.projects.create({
        name: "create_project",
        layout: {},
      });
      const { result } = renderHook(
        () => ({ create: Log.useCreate(), store: Flux.useStore<Log.FluxSubStore>() }),
        { wrapper },
      );
      const key = uuid.create();
      await act(async () => {
        await result.current.create.updateAsync({
          key,
          project: project.key,
          name: "created_log",
        });
      });
      expect(result.current.create.variant).toEqual("success");
      expect((await client.logs.retrieve({ key })).name).toEqual("created_log");
      expect(result.current.store.logs.get(key)?.name).toEqual("created_log");
    });
  });

  describe("useRename", () => {
    it("should rename a log and update the cache", async () => {
      const created = await createLog({ name: "original_name" });
      const { result } = renderHook(
        () => ({
          retrieve: Log.useRetrieve({ key: created.key }),
          rename: Log.useRename(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      await act(async () => {
        await result.current.rename.updateAsync({ key: created.key, name: "renamed" });
      });
      expect((await client.logs.retrieve({ key: created.key })).name).toEqual(
        "renamed",
      );
      await waitFor(() =>
        expect(result.current.retrieve.data?.name).toEqual("renamed"),
      );
    });
  });

  describe("useDispatch", () => {
    it("should round-trip an action through the server", async () => {
      const created = await createLog({ name: "dispatch_test" });
      const { result } = renderHook(
        () => ({
          retrieve: Log.useRetrieve({ key: created.key }),
          dispatch: Log.useDispatch(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [log.addChannel({ channel: 1 })],
        });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.channels.map((e) => e.channel)).toContain(
          1,
        ),
      );
      const onServer = await client.logs.retrieve({ key: created.key });
      expect(onServer.channels.map((e) => e.channel)).toContain(1);
    });
  });

  describe("useDelete", () => {
    it("should delete a single log", async () => {
      const created = await createLog({ name: "delete_single" });
      const { result } = renderHook(() => Log.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(created.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(client.logs.retrieve({ key: created.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should delete multiple logs", async () => {
      const l1 = await createLog({ name: "delete_multi_1" });
      const l2 = await createLog({ name: "delete_multi_2" });
      const { result } = renderHook(() => Log.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync([l1.key, l2.key]);
      });
      expect(result.current.variant).toEqual("success");
      await expect(client.logs.retrieve({ key: l1.key })).rejects.toThrow(
        NotFoundError,
      );
      await expect(client.logs.retrieve({ key: l2.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });
});
