// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, observe } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { DisconnectedError } from "@/errors";
import {
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
  holdResponses,
  rejectRequests,
  waitForStreamLive,
} from "@/testutil";

const remote = createTestClient();

const TARGET = "/channel/retrieve-built-ins";

// Builds the built-in keys over a fetch whose answers the spec gives by hand.
const createBuiltIns = () => {
  const epochs = new observe.Observer<void>();
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
  return { builtIns, connect: () => epochs.notify(), fetches, errors };
};

describe("BuiltIns", () => {
  it("should fetch nothing until a reader asks", () => {
    const { builtIns, connect, fetches } = createBuiltIns();
    connect();
    expect(fetches).toHaveLength(0);
    expect(builtIns.get()).toBeNull();
    expect(fetches).toHaveLength(1);
  });

  it("should hold no keys until the fetch of a connection answers", async () => {
    const { builtIns, connect, fetches } = createBuiltIns();
    const retrieved = builtIns.retrieve();
    expect(fetches).toHaveLength(0);
    connect();
    fetches[0].resolve([1, 2]);
    expect(await retrieved).toEqual(new Set([1, 2]));
    expect(builtIns.get()).toBe(await retrieved);
  });

  it("should keep the last keys while a reconnect fetches them again", async () => {
    const { builtIns, connect, fetches } = createBuiltIns();
    const changes: Array<Set<channel.Key> | null> = [];
    builtIns.onChange(({ keys }) => changes.push(keys));
    connect();
    fetches[0].resolve([1]);
    const first = await builtIns.retrieve();
    connect();
    expect(builtIns.get()).toBe(first);
    expect(await builtIns.retrieve()).toBe(first);
    fetches[1].resolve([1, 3]);
    await expect.poll(() => builtIns.get()).toEqual(new Set([1, 3]));
    expect(changes).toEqual([new Set([1]), new Set([1, 3])]);
  });

  it("should report a failed fetch once and throw it to each reader", async () => {
    const { builtIns, connect, fetches, errors } = createBuiltIns();
    builtIns.get();
    connect();
    const refused = new Error("refused");
    fetches[0].reject(refused);
    const message = "failed to fetch built-in channels";
    await expect(builtIns.retrieve()).rejects.toThrow(message);
    await expect(builtIns.retrieve()).rejects.toThrow(message);
    expect(errors.map((e) => [e.message, e.cause])).toEqual([[message, refused]]);
    expect(fetches).toHaveLength(1);
    connect();
    const retrieved = builtIns.retrieve();
    fetches[1].resolve([4]);
    expect(await retrieved).toEqual(new Set([4]));
  });

  it("should hold a failed fetch in its state until the next connection", async () => {
    const { builtIns, connect, fetches } = createBuiltIns();
    const states: channel.BuiltInsState[] = [];
    builtIns.onChange((state) => states.push(state));
    expect(builtIns.state()).toEqual({ keys: null, error: null });
    connect();
    fetches[0].reject(new Error("refused"));
    await expect
      .poll(() => builtIns.state().error?.message)
      .toBe("failed to fetch built-in channels");
    const failed = builtIns.state();
    expect(builtIns.state()).toBe(failed);
    connect();
    expect(builtIns.state()).toEqual({ keys: null, error: null });
    fetches[1].resolve([8]);
    await expect.poll(() => builtIns.get()).toEqual(new Set([8]));
    expect(states.map(({ keys, error }) => [keys, error?.message ?? null])).toEqual([
      [null, "failed to fetch built-in channels"],
      [null, null],
      [new Set([8]), null],
    ]);
  });

  it("should leave a connection error unreported and wait for the next connection", async () => {
    const { builtIns, connect, fetches, errors } = createBuiltIns();
    const retrieved = builtIns.retrieve();
    connect();
    fetches[0].reject(new DisconnectedError());
    connect();
    fetches[1].resolve([5]);
    expect(await retrieved).toEqual(new Set([5]));
    expect(errors).toEqual([]);
  });

  it("should drop the answer of an earlier connection", async () => {
    const { builtIns, connect, fetches, errors } = createBuiltIns();
    const retrieved = builtIns.retrieve();
    connect();
    connect();
    fetches[1].resolve([6]);
    expect(await retrieved).toEqual(new Set([6]));
    fetches[0].reject(new Error("late"));
    fetches[0].resolve([7]);
    await expect.poll(() => builtIns.get()).toEqual(new Set([6]));
    expect(errors).toEqual([]);
  });
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
    await expect.poll(() => internal.length > 0 || keys()?.length === 2).toBe(true);
    expect(internal.map((e) => e.message)).toEqual([]);
    expect(keys()).toEqual([first.key, second.key].sort());
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
