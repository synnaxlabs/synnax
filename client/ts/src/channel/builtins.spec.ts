// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, sleep, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";

import { DisconnectedError, type Synnax } from "@/index";
import {
  builtInsFetches,
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
  holdRequests,
  holdResponses,
  isBuiltInsFetch,
  rejectRequests,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";

const remote = createTestClient();

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

describe("Lists with an internal line", () => {
  it("should page to the end once the built-in keys arrived", async () => {
    const prefix = await createPagedChannels();
    const client = createTestClient();
    const send = spyOnSend(client);
    await client.connect();
    await waitForStreamLive(client.connection);
    await expect.poll(() => builtInsFetches(send).length).toBeGreaterThan(0);
    await builtInsFetches(send)[0];
    expect(await pageToEnd(client, prefix)).toBe(PAGE_COUNT);
  }, 60000);

  it("should page to the end before the built-in keys arrive", async () => {
    const prefix = await createPagedChannels();
    const client = createTestClient();
    holdResponses(client, isBuiltInsFetch);
    await client.connect();
    await waitForStreamLive(client.connection);
    expect(await pageToEnd(client, prefix)).toBe(PAGE_COUNT);
  }, 60000);
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
  // Only the fetches of a later connection answer late or fail.
  let later = false;
  const fetchesLater = (target: string, req: unknown): boolean =>
    later && isBuiltInsFetch(target, req);
  const held = after === "answers-late" ? holdResponses(local, fetchesLater) : null;
  let failed = false;
  // A link that drops once can fail the first request after it returns.
  const rejected =
    after === "fails-once"
      ? rejectRequests(
          local,
          (target, req) => {
            if (failed || !fetchesLater(target, req)) return false;
            failed = true;
            return true;
          },
          new DisconnectedError(),
        )
      : null;
  const send = spyOnSend(local);
  await local.connect();
  await waitForStreamLive(local.connection);
  await expect.poll(() => builtInsFetches(send).length).toBeGreaterThan(0);
  await builtInsFetches(send)[0];
  const first = await remote.channels.create({
    name: `builtin_${id.create()}`,
    dataType: "float32",
    virtual: true,
  });
  const secondName = `builtin_${id.create()}`;
  // The built-in keys are known, so the client matches this query itself.
  const list = local.channels.watch({
    where: { internal: { eq: false }, name: { in: [first.name, secondName] } },
  });
  const off = list.onChange(() => {});
  try {
    expect((await list.retrieve()).records.map((c) => c.key)).toEqual([first.key]);
    later = true;
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

describe("Losing the session with the built-in key fetch in flight", () => {
  // Holds the fetch until end has run, so the fetch reaches the Core with no session.
  const loseSession = async (end: (client: Synnax) => Promise<void>) => {
    const internal: Error[] = [];
    const client = createTestClient({ onInternalError: (e) => internal.push(e) });
    const held = holdRequests(client, isBuiltInsFetch);
    await client.connect();
    await expect.poll(() => held.sent()).toBeGreaterThan(0);
    await end(client);
    held.release();
    await sleep.sleep(TimeSpan.milliseconds(200));
    await client.close();
    return internal;
  };

  it("should not report the fetch when it fails after the close", async () => {
    expect(await loseSession(async (client) => await client.close())).toEqual([]);
  });

  it("should not report the fetch when it fails after a logout", async () => {
    expect(await loseSession(async (client) => await client.auth.logout())).toEqual([]);
  });
});
