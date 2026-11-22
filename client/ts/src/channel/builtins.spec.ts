// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import type Synnax from "@/client";
import { DisconnectedError } from "@/errors";
import {
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
  holdRequests,
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
    const builtIns = vi.spyOn(client.channels, "retrieveBuiltIns");
    await client.connect();
    await waitForStreamLive(client.connection);
    await expect.poll(() => builtIns.mock.results.length).toBeGreaterThan(0);
    await builtIns.mock.results[0].value;
    expect(await pageToEnd(client, prefix)).toBe(PAGE_COUNT);
  }, 60000);

  it("should page to the end before the built-in keys arrive", async () => {
    const prefix = await createPagedChannels();
    const client = createTestClient();
    holdRequests(client, "/channel/retrieve-built-ins");
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
  const original = local.channels.retrieveBuiltIns.bind(local.channels);
  const builtIns = vi.spyOn(local.channels, "retrieveBuiltIns");
  const released = Promise.withResolvers<void>();
  await local.connect();
  await waitForStreamLive(local.connection);
  await expect.poll(() => builtIns.mock.results.length).toBeGreaterThan(0);
  await builtIns.mock.results[0].value;
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
    if (after === "answers-late")
      builtIns.mockImplementation(async () => {
        await released.promise;
        return await original();
      });
    // A link that drops once can fail the first request after it returns.
    if (after === "fails-once") builtIns.mockRejectedValueOnce(new DisconnectedError());
    await proxy.sever();
    await proxy.restore();
    await waitForStreamLive(local.connection);
    const second = await remote.channels.create({
      name: secondName,
      dataType: "float32",
      virtual: true,
    });
    released.resolve();
    const keys = () =>
      list
        .getCached()
        ?.records.map((c) => c.key)
        .sort();
    await expect.poll(() => internal.length > 0 || keys()?.length === 2).toBe(true);
    expect(internal.map((e) => e.message)).toEqual([]);
    expect(keys()).toEqual([first.key, second.key].sort());
  } finally {
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
