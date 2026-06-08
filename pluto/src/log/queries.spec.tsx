// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, log, NotFoundError } from "@synnaxlabs/client";
import { color, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { aetherTest } from "@/aether/test";
import { Flux } from "@/flux";
import { flux } from "@/flux/aether";
import { Log } from "@/log";
import { Pluto } from "@/pluto";
import { status } from "@/status/aether";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

// Local store-seeding harness (no live core). Exposes the flux client so tests can seed
// the log store directly and exercise selectors / dispatch without a server round-trip.

const AetherProvider = aetherTest.createProvider({
  ...synnax.REGISTRY,
  ...status.REGISTRY,
  ...flux.createRegistry({ storeConfig: {} }),
});

const createSeededWrapper = (): {
  wrapper: FC<PropsWithChildren>;
  fluxClient: Flux.Client;
} => {
  const fluxClient = new Flux.Client({
    client: null,
    storeConfig: { ...Pluto.FLUX_STORE_CONFIG },
    handleError: status.createErrorHandler(console.error),
    handleAsyncError: status.createAsyncErrorHandler(console.error),
  });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <AetherProvider>
      <Status.Aggregator>
        <Synnax.TestProvider client={null}>
          <Flux.Provider client={fluxClient}>{children}</Flux.Provider>
        </Synnax.TestProvider>
      </Status.Aggregator>
    </AetherProvider>
  );
  return { wrapper, fluxClient };
};

const entry = (
  channel: number,
  overrides: Partial<log.ChannelEntry> = {},
): log.ChannelEntry => ({
  channel,
  color: color.ZERO,
  notation: "standard",
  precision: -1,
  alias: "",
  timestamp: { format: "preciseDate", tz: "local" },
  ...overrides,
});

const KEY = "00000000-0000-0000-0000-000000000001";

const logDoc = (overrides: Partial<log.Log> = {}): log.Log => ({
  key: KEY,
  name: "Test Log",
  channels: [],
  remoteCreated: true,
  timestampPrecision: 0,
  showChannelNames: true,
  showReceiptTimestamp: true,
  ...overrides,
});

const seedWith = (doc: log.Log) => {
  const { wrapper, fluxClient } = createSeededWrapper();
  const store = fluxClient.scopedStore<Log.FluxSubStore>("test-writer");
  store.logs.set(doc.key, doc);
  return { wrapper, store };
};

describe("log queries (seeded, no core)", () => {
  describe("selectors", () => {
    it("useSelectName reads the name", () => {
      const { wrapper } = seedWith(logDoc({ name: "Sensors" }));
      const { result } = renderHook(() => Log.useSelectName({ key: KEY }), { wrapper });
      expect(result.current).toBe("Sensors");
    });

    it("useSelectChannels reads the channel entries", () => {
      const { wrapper } = seedWith(logDoc({ channels: [entry(1), entry(2)] }));
      const { result } = renderHook(() => Log.useSelectChannels({ key: KEY }), {
        wrapper,
      });
      expect(result.current.map((e) => e.channel)).toEqual([1, 2]);
    });

    it("useSelectChannelKeys reads the ordered keys", () => {
      const { wrapper } = seedWith(logDoc({ channels: [entry(3), entry(7)] }));
      const { result } = renderHook(() => Log.useSelectChannelKeys({ key: KEY }), {
        wrapper,
      });
      expect(result.current).toEqual([3, 7]);
    });

    it("useSelectChannelEntry reads a single entry", () => {
      const { wrapper } = seedWith(
        logDoc({ channels: [entry(1), entry(2, { alias: "pressure" })] }),
      );
      const { result } = renderHook(
        () => Log.useSelectChannelEntry({ key: KEY, channel: 2 }),
        { wrapper },
      );
      expect(result.current?.alias).toBe("pressure");
    });

    it("useSelectChannelEntry returns null when the channel is absent", () => {
      const { wrapper } = seedWith(logDoc({ channels: [entry(1)] }));
      const { result } = renderHook(
        () => Log.useSelectChannelEntry({ key: KEY, channel: 99 }),
        { wrapper },
      );
      expect(result.current).toBeNull();
    });

    it("reads the display-config scalars", () => {
      const { wrapper } = seedWith(
        logDoc({
          timestampPrecision: 3,
          showChannelNames: false,
          showReceiptTimestamp: false,
        }),
      );
      const { result } = renderHook(
        () => ({
          precision: Log.useSelectTimestampPrecision({ key: KEY }),
          names: Log.useSelectShowChannelNames({ key: KEY }),
          receipt: Log.useSelectShowReceiptTimestamp({ key: KEY }),
        }),
        { wrapper },
      );
      expect(result.current.precision).toBe(3);
      expect(result.current.names).toBe(false);
      expect(result.current.receipt).toBe(false);
    });
  });

  describe("memoization / reference stability", () => {
    it("useSelectChannelKeys keeps a stable reference when only config changes", () => {
      const { wrapper, store } = seedWith(logDoc({ channels: [entry(1), entry(2)] }));
      const { result } = renderHook(() => Log.useSelectChannelKeys({ key: KEY }), {
        wrapper,
      });
      const first = result.current;
      act(() => {
        store.logs.set(KEY, logDoc({ channels: [entry(1, { alias: "x" }), entry(2)] }));
      });
      expect(result.current).toBe(first);
    });

    it("useSelectChannelKeys yields a new value when the key set changes", () => {
      const { wrapper, store } = seedWith(logDoc({ channels: [entry(1), entry(2)] }));
      const { result } = renderHook(() => Log.useSelectChannelKeys({ key: KEY }), {
        wrapper,
      });
      const first = result.current;
      act(() => {
        store.logs.set(KEY, logDoc({ channels: [entry(1), entry(2), entry(3)] }));
      });
      expect(result.current).not.toBe(first);
      expect(result.current).toEqual([1, 2, 3]);
    });

    it("useSelectChannelEntry keeps a stable reference when its entry object is unchanged", () => {
      const sharedEntry = entry(2, { alias: "shared" });
      const { wrapper, store } = seedWith(
        logDoc({ channels: [entry(1), sharedEntry] }),
      );
      const { result } = renderHook(
        () => Log.useSelectChannelEntry({ key: KEY, channel: 2 }),
        { wrapper },
      );
      const first = result.current;
      act(() => {
        store.logs.set(
          KEY,
          logDoc({ channels: [entry(1, { alias: "y" }), sharedEntry] }),
        );
      });
      expect(result.current).toBe(first);
    });

    it("useSelectTimestampPrecision is stable across unrelated edits", () => {
      const { wrapper, store } = seedWith(logDoc({ timestampPrecision: 2 }));
      const { result } = renderHook(
        () => Log.useSelectTimestampPrecision({ key: KEY }),
        { wrapper },
      );
      const first = result.current;
      act(() => {
        store.logs.set(KEY, logDoc({ timestampPrecision: 2, showChannelNames: false }));
      });
      expect(result.current).toBe(first);
    });
  });
});

// Integration tests against a live core (createTestClient → localhost:9090).

const client = createTestClient();

describe("log queries (live core)", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useRetrieve", () => {
    it("should retrieve a log by key", async () => {
      const workspace = await client.workspaces.create({
        name: "test_workspace",
        layout: {},
      });
      const created = await client.logs.create(workspace.key, {
        name: "retrieve_test",
      });
      const { result } = renderHook(() => Log.useRetrieve({ key: created.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data?.key).toEqual(created.key);
      expect(result.current.data?.name).toEqual("retrieve_test");
    });

    it("should cache retrieved logs", async () => {
      const workspace = await client.workspaces.create({
        name: "cache_workspace",
        layout: {},
      });
      const created = await client.logs.create(workspace.key, { name: "cached_log" });
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
      const workspace = await client.workspaces.create({
        name: "create_workspace",
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
          workspace: workspace.key,
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
      const workspace = await client.workspaces.create({
        name: "rename_workspace",
        layout: {},
      });
      const created = await client.logs.create(workspace.key, {
        name: "original_name",
      });
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
      const workspace = await client.workspaces.create({
        name: "dispatch_workspace",
        layout: {},
      });
      const created = await client.logs.create(workspace.key, {
        name: "dispatch_test",
      });
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
      const workspace = await client.workspaces.create({
        name: "delete_workspace",
        layout: {},
      });
      const created = await client.logs.create(workspace.key, {
        name: "delete_single",
      });
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
      const workspace = await client.workspaces.create({
        name: "delete_multi_workspace",
        layout: {},
      });
      const l1 = await client.logs.create(workspace.key, { name: "delete_multi_1" });
      const l2 = await client.logs.create(workspace.key, { name: "delete_multi_2" });
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
