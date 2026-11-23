// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, DisconnectedError, type Synnax } from "@synnaxlabs/client";
import {
  answerRequests,
  createProxiedTestClient,
  FAST_RETRY,
  rejectRequests,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/channel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const TARGET = "/channel/retrieve-built-ins";

// Drops the connection of client and waits for the next one.
const reconnect = async (
  client: Synnax,
  proxy: { sever: () => Promise<void>; restore: () => Promise<void> },
): Promise<void> => {
  const epoch = client.connection.status.details.epoch;
  await proxy.sever();
  await proxy.restore();
  await waitFor(
    () => expect(client.connection.status.details.epoch).toBeGreaterThan(epoch),
    { timeout: 15000 },
  );
};

describe("useResultBuiltIns across a reconnect", () => {
  it("should hold a dropped fetch as failed and take the keys of the next connection", async () => {
    const internal: Error[] = [];
    const { proxy, client } = await createProxiedTestClient({
      retry: FAST_RETRY,
      onRetry: () => {},
      onInternalError: (e) => internal.push(e),
    });
    // The first fetch fails as a dropped link does.
    let dropped = false;
    const rejected = rejectRequests(
      client,
      (target) => {
        if (target !== TARGET || dropped) return false;
        dropped = true;
        return true;
      },
      new DisconnectedError(),
    );
    const wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    const { result } = renderHook(() => Channel.useResultBuiltIns(), { wrapper });
    await waitFor(() => expect(result.current.data?.error).not.toBeNull());
    expect(dropped).toBe(true);
    expect(result.current.data?.keys).toBeNull();
    expect(internal.map((e) => e.message)).toEqual([
      "Failed to fetch built-in channels. Operation failed because no Core is connected.",
    ]);
    await reconnect(client, proxy);
    await waitFor(() => expect(result.current.data?.keys).not.toBeNull(), {
      timeout: 8000,
    });
    rejected.mockRestore();
  }, 60000);

  it("should follow the keys the Core reports after a reconnect", async () => {
    const { proxy, client } = await createProxiedTestClient({
      retry: FAST_RETRY,
      onRetry: () => {},
    });
    const wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    const { result } = renderHook(() => Channel.useResultBuiltIns(), { wrapper });
    await waitFor(() => expect(result.current.data?.keys).not.toBeNull());
    // A channel that the restarted Core declares as a built-in, as an upgrade that
    // adopts a channel of a new built-in's name does.
    const adopted = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const before = (await client.channels.builtIns.retrieve()).keys ?? [];
    answerRequests(client, TARGET, async () => ({ keys: [...before, adopted.key] }));
    await reconnect(client, proxy);
    await waitFor(
      () => expect(result.current.data?.keys?.has(adopted.key)).toBe(true),
      { timeout: 8000 },
    );
    expect(client.channels.builtIns.state().keys?.has(adopted.key)).toBe(true);
  }, 60000);

  it("should settle a failed fetch that a reconnect overtook", async () => {
    const internal: Error[] = [];
    const { proxy, client } = await createProxiedTestClient({
      retry: FAST_RETRY,
      onRetry: () => {},
      onInternalError: (e) => internal.push(e),
    });
    const wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    const answered = Promise.withResolvers<void>();
    answerRequests(client, TARGET, async () => {
      await answered.promise;
      throw new Error("refused");
    });
    const { result } = renderHook(() => Channel.useResultBuiltIns(), { wrapper });
    expect(result.current.data).toEqual({ keys: null, error: null });
    // The stream reconnects while the fetch runs.
    await reconnect(client, proxy);
    answered.resolve();
    await waitFor(() => expect(result.current.data?.error).not.toBeNull(), {
      timeout: 8000,
    });
    // The answer of the first connection is stale, so only the second one reports.
    expect(internal.map((e) => e.message)).toEqual([
      "Failed to fetch built-in channels. refused",
    ]);
  }, 60000);
});
