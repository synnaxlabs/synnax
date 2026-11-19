// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { DisconnectedError } from "@/errors";
import {
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
  waitForStreamLive,
} from "@/testutil";

const remote = createTestClient();

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
