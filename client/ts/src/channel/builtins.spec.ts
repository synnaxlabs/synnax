// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, observe } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { channel } from "@/channel";
import Synnax from "@/client";
import { DisconnectedError } from "@/errors";
import {
  createCuttingProxy,
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
  holdResponses,
  recordResponses,
  rejectRequests,
  spyOnSend,
  testClientParams,
  waitForStreamLive,
} from "@/testutil";

const remote = createTestClient();

const TARGET = "/channel/retrieve-built-ins";

const PENDING = { keys: null, error: null };

const MESSAGE = "failed to fetch built-in channels";

// Builds the built-in keys over a fetch whose answers the spec gives by hand.
const createBuiltIns = () => {
  const epochs = new observe.Observer<number>();
  const fetches: Array<PromiseWithResolvers<channel.Key[]>> = [];
  const errors: Error[] = [];
  const builtIns = new channel.BuiltIns({
    fetch: async () => {
      const fetched = Promise.withResolvers<channel.Key[]>();
      fetches.push(fetched);
      return await fetched.promise;
    },
    onEpoch: (callback) => epochs.onChange(callback),
    onError: (error) => errors.push(error),
  });
  let epoch = 0;
  return {
    builtIns,
    connect: () => epochs.notify(++epoch),
    replace: () => {
      epoch = 0;
      epochs.notify(0);
    },
    fetches,
    errors,
  };
};

describe("BuiltIns", () => {
  it("should start no fetch on a read of the state or a connection", () => {
    const { builtIns, connect, fetches } = createBuiltIns();
    expect(builtIns.state()).toEqual(PENDING);
    connect();
    expect(builtIns.state()).toEqual(PENDING);
    expect(fetches).toHaveLength(0);
  });

  it("should fetch on the first awaited read with no connection", async () => {
    const { builtIns, fetches } = createBuiltIns();
    const retrieved = builtIns.retrieve();
    expect(fetches).toHaveLength(1);
    fetches[0].resolve([1, 2]);
    const state = await retrieved;
    expect(state).toEqual({ keys: new Set([1, 2]), error: null });
    expect(builtIns.state()).toBe(state);
  });

  it("should fetch again on each later connection and keep the last keys", async () => {
    const { builtIns, connect, fetches } = createBuiltIns();
    const changes: Array<Set<channel.Key> | null> = [];
    builtIns.onChange(({ keys }) => changes.push(keys));
    const retrieved = builtIns.retrieve();
    fetches[0].resolve([1]);
    const first = await retrieved;
    connect();
    expect(fetches).toHaveLength(2);
    expect(builtIns.state()).toBe(first);
    expect(await builtIns.retrieve()).toBe(first);
    fetches[1].resolve([1, 3]);
    await expect.poll(() => builtIns.state().keys).toEqual(new Set([1, 3]));
    expect(changes).toEqual([new Set([1]), new Set([1, 3])]);
  });

  it("should keep the state when a later connection fetches the same keys", async () => {
    const { builtIns, connect, fetches } = createBuiltIns();
    const retrieved = builtIns.retrieve();
    fetches[0].resolve([1, 2]);
    const first = await retrieved;
    const changes: channel.BuiltInsState[] = [];
    builtIns.onChange((state) => changes.push(state));
    connect();
    fetches[1].resolve([2, 1]);
    await fetches[1].promise;
    await builtIns.retrieve();
    expect(builtIns.state()).toBe(first);
    expect(changes).toEqual([]);
  });

  it("should hold a failed fetch as its error and report a run of failures once", async () => {
    const { builtIns, connect, fetches, errors } = createBuiltIns();
    const retrieved = builtIns.retrieve();
    const refused = new Error("refused");
    fetches[0].reject(refused);
    const failed = await retrieved;
    expect(failed.keys).toBeNull();
    expect(failed.error?.message).toBe(MESSAGE);
    expect(failed.error?.cause).toBe(refused);
    // An awaited read fetches again, and a second failure in one connection is quiet.
    const again = builtIns.retrieve();
    expect(fetches).toHaveLength(2);
    expect(builtIns.state()).toBe(failed);
    fetches[1].reject(new Error("refused again"));
    expect((await again).error?.message).toBe(MESSAGE);
    expect(errors).toHaveLength(1);
    // A new connection does not end the run; a fetch that passes does.
    connect();
    const third = builtIns.retrieve();
    fetches[2].reject(new Error("refused on the next connection"));
    expect((await third).error?.message).toBe(MESSAGE);
    expect(errors).toHaveLength(1);
    const next = builtIns.retrieve();
    fetches[3].resolve([4]);
    expect(await next).toEqual({ keys: new Set([4]), error: null });
    connect();
    fetches[4].reject(new Error("refused after a pass"));
    await expect.poll(() => errors.length).toBe(2);
  });

  it("should hold a connection error as a failed fetch", async () => {
    const { builtIns, fetches, errors } = createBuiltIns();
    const retrieved = builtIns.retrieve();
    const dropped = new DisconnectedError();
    fetches[0].reject(dropped);
    expect((await retrieved).error?.cause).toBe(dropped);
    expect(errors).toHaveLength(1);
  });

  it("should fail a fetch with no answer in 10 seconds", async () => {
    // Only the timers are fake: a fake clock would skew the shared client's heartbeat.
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
    onTestFinished(() => {
      vi.useRealTimers();
    });
    const { builtIns, fetches } = createBuiltIns();
    const retrieved = builtIns.retrieve();
    expect(fetches).toHaveLength(1);
    await vi.advanceTimersByTimeAsync(10_000);
    const state = await retrieved;
    expect(state.error?.message).toBe(MESSAGE);
    expect((state.error?.cause as Error).message).toBe("no answer in 10s");
  });

  it("should clear the keys when a new Core comes up behind the address", async () => {
    const { builtIns, connect, replace, fetches } = createBuiltIns();
    const retrieved = builtIns.retrieve();
    fetches[0].resolve([1]);
    await retrieved;
    replace();
    expect(builtIns.state()).toEqual(PENDING);
    expect(fetches).toHaveLength(1);
    connect();
    fetches[1].resolve([9]);
    await expect.poll(() => builtIns.state().keys).toEqual(new Set([9]));
  });

  it("should drop the answer of an earlier connection", async () => {
    const { builtIns, connect, fetches, errors } = createBuiltIns();
    const retrieved = builtIns.retrieve();
    connect();
    fetches[0].reject(new Error("late"));
    await expect.poll(() => fetches.length).toBe(2);
    fetches[1].resolve([6]);
    expect(await retrieved).toEqual({ keys: new Set([6]), error: null });
    expect(errors).toEqual([]);
  });
});

const PAGE_COUNT = 1010;

const createPagedChannels = async (): Promise<string> => {
  const prefix = `builtin_page_${id.create()}_`;
  await remote.channels.create(
    Array.from({ length: PAGE_COUNT }, (_, i) => ({
      name: `${prefix}${i}`,
      dataType: "float32",
      virtual: true,
    })),
  );
  return prefix;
};

// Pages through a watched list of non-internal channels, as a channel select does.
const pageToEnd = async (client: Synnax, prefix: string): Promise<number> => {
  const list = client.channels.watch({
    where: { internal: { eq: false }, name: { startsWith: prefix } },
    limit: 50,
  });
  const off = list.onChange(() => {});
  onTestFinished(off);
  let window = await list.retrieve();
  for (let i = 0; i < 40 && window.end != null; i++) window = await list.more(100);
  return window.records.length;
};

describe("Built-in channels on a live client", () => {
  it("should settle the keys after one response is cut while the stream stays live", async () => {
    const proxy = await createCuttingProxy();
    const internal: Error[] = [];
    const client = new Synnax({
      ...testClientParams(),
      host: "localhost",
      port: proxy.port,
      retry: FAST_RETRY,
      onRetry: () => {},
      onInternalError: (e) => internal.push(e),
    });
    onTestFinished(async () => await client.close());
    await client.connect();
    await waitForStreamLive(client.connection);
    const epoch = client.connection.status.details.epoch;
    proxy.cutNext(TARGET);
    const state = await client.channels.builtIns.retrieve();
    expect(proxy.cuts()).toBe(1);
    expect(state.error?.message).toBe(MESSAGE);
    expect(internal.map((e) => e.message)).toEqual([
      `Failed to fetch built-in channels. Cannot reach Core at localhost:${proxy.port}`,
    ]);
    expect(client.connection.status.details.epoch).toBe(epoch);
    // The link is healthy again, so the next awaited read gets the keys.
    expect((await client.channels.builtIns.retrieve()).keys).not.toBeNull();
  }, 30000);

  it("should page a non-internal list to the end when the keys already arrived", async () => {
    const prefix = await createPagedChannels();
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    await client.channels.builtIns.retrieve();
    expect(await pageToEnd(client, prefix)).toBe(PAGE_COUNT);
  }, 60000);

  it("should page a non-internal list to the end before any read of the keys", async () => {
    const prefix = await createPagedChannels();
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    expect(await pageToEnd(client, prefix)).toBe(PAGE_COUNT);
  }, 60000);

  it("should place new channels in an internal-line list without fetching it again", async () => {
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    const prefix = `builtin_lazy_${id.create()}_`;
    const list = client.channels.watch({
      where: { internal: { eq: false }, name: { startsWith: prefix } },
      limit: 50,
    });
    let length = 0;
    onTestFinished(list.onChange((w) => (length = w?.records.length ?? 0)));
    await list.retrieve();
    // The list fetched the keys; nothing else read them.
    await expect.poll(() => client.channels.builtIns.state().keys).not.toBeNull();
    const send = spyOnSend(client);
    for (let i = 0; i < 5; i++) {
      await remote.channels.create({
        name: `${prefix}${i}`,
        dataType: "float32",
        virtual: true,
      });
      await expect.poll(() => length).toBe(i + 1);
    }
    const refetches = send.mock.calls.filter(
      ([target]) => target === "/channel/retrieve",
    );
    expect(refetches).toHaveLength(0);
  }, 60000);

  it("should answer an awaited read on a client with no change stream", async () => {
    const client = createTestClient({ cache: false });
    await client.connect();
    const { keys } = await client.channels.builtIns.retrieve();
    expect(keys?.size).toBeGreaterThan(0);
  }, 15000);

  it("should keep the state across a reconnect that fetches the same keys", async () => {
    const { proxy, client } = await createProxiedTestClient({
      retry: FAST_RETRY,
      onRetry: () => {},
    });
    await client.connect();
    await waitForStreamLive(client.connection);
    const before = await client.channels.builtIns.retrieve();
    const changes: channel.BuiltInsState[] = [];
    client.channels.builtIns.onChange((s) => changes.push(s));
    const responses = recordResponses(client, TARGET);
    const epoch = client.connection.status.details.epoch;
    await proxy.sever();
    await proxy.restore();
    await expect
      .poll(() => client.connection.status.details.epoch, { timeout: 15000 })
      .toBeGreaterThan(epoch);
    await expect.poll(() => responses.length, { timeout: 8000 }).toBe(1);
    await responses[0];
    await client.channels.builtIns.retrieve();
    expect(client.channels.builtIns.state()).toBe(before);
    expect(changes).toEqual([]);
  }, 30000);
});

// How the fetch of the built-in keys behaves after the reconnect.
type AfterReconnect = "answers" | "answers-late" | "fails-once";

// Watches a list with an internal line across a reconnect, and checks that it admits
// a channel created after the reconnect.
const reconnectWithList = async (after: AfterReconnect): Promise<void> => {
  const internal: Error[] = [];
  const { proxy, client: local } = await createProxiedTestClient({
    retry: FAST_RETRY,
    onRetry: () => {},
    onInternalError: (e) => internal.push(e),
  });
  await local.connect();
  await waitForStreamLive(local.connection);
  await local.channels.builtIns.retrieve();
  const first = await remote.channels.create({
    name: `builtin_${id.create()}`,
    dataType: "float32",
    virtual: true,
  });
  const secondName = `builtin_${id.create()}`;
  const held = after === "answers-late" ? holdResponses(local, TARGET) : null;
  // A link that drops once can fail the first request after it returns.
  let failed = false;
  const rejected =
    after === "fails-once"
      ? rejectRequests(
          local,
          (target) => {
            if (target !== TARGET || failed) return false;
            failed = true;
            return true;
          },
          new DisconnectedError(),
        )
      : null;
  // The built-in keys are known, so the client matches this query itself.
  const list = local.channels.watch({
    where: { internal: { eq: false }, name: { in: [first.name, secondName] } },
  });
  const off = list.onChange(() => {});
  try {
    expect((await list.retrieve()).records.map((c) => c.key)).toEqual([first.key]);
    await proxy.sever();
    await proxy.restore();
    await waitForStreamLive(local.connection);
    const second = await remote.channels.create({
      name: secondName,
      dataType: "float32",
      virtual: true,
    });
    held?.release();
    const keys = () =>
      list
        .getCached()
        ?.records.map((c) => c.key)
        .sort();
    // A failed refetch keeps the last keys and reports once.
    const reported =
      after === "fails-once"
        ? [
            "Failed to fetch built-in channels. Operation failed because no Core is connected.",
          ]
        : [];
    await expect.poll(() => keys()?.length).toBe(2);
    expect(keys()).toEqual([first.key, second.key].sort());
    await expect.poll(() => internal.map((e) => e.message)).toEqual(reported);
  } finally {
    rejected?.mockRestore();
    off();
  }
};

describe("Built-in channels after a reconnect", () => {
  it("should keep a watched internal list live", async () => {
    await reconnectWithList("answers");
  }, 30000);

  it("should recover a watched internal list once late built-in keys arrive", async () => {
    await reconnectWithList("answers-late");
  }, 30000);

  it("should keep a watched internal list live when a refetch of the keys fails", async () => {
    await reconnectWithList("fails-once");
  }, 30000);
});
