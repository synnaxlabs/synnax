// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/client";
import {
  createProxiedTestClient,
  FAST_RETRY,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Channel } from "@/channel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("useResultBuiltIns across a reconnect", () => {
  it("should fetch the keys again after a reconnect drops a failed fetch", async () => {
    const { proxy, client } = await createProxiedTestClient({ retry: FAST_RETRY });
    const wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    const epoch = client.connection.status.details.epoch;
    await proxy.sever();
    const { result, rerender } = renderHook(() => Channel.useResultBuiltIns(), {
      wrapper,
    });
    await waitFor(() => expect(result.current.variant).toBe("error"), {
      timeout: 15000,
    });
    await proxy.restore();
    await waitFor(
      () => expect(client.connection.status.details.epoch).toBeGreaterThan(epoch),
      { timeout: 15000 },
    );
    // Any later render of a mounted reader, such as a tree item that re-renders.
    rerender();
    await waitFor(() => expect(result.current.variant).toBe("success"), {
      timeout: 8000,
    });
  }, 60000);

  it("should follow the keys the Core reports after a reconnect", async () => {
    const { proxy, client } = await createProxiedTestClient({ retry: FAST_RETRY });
    const wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    const { result, rerender } = renderHook(() => Channel.useResultBuiltIns(), {
      wrapper,
    });
    await waitFor(() => expect(result.current.variant).toBe("success"));
    // A channel that the restarted Core declares as a built-in, as an upgrade that
    // adopts a channel of a new built-in's name does.
    const adopted = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const before = await client.channels.retrieveBuiltIns();
    const answer = vi
      .spyOn(client.channels, "retrieveBuiltIns")
      .mockResolvedValue([...before, adopted.key]);
    const epoch = client.connection.status.details.epoch;
    await proxy.sever();
    await proxy.restore();
    await waitFor(
      () => expect(client.connection.status.details.epoch).toBeGreaterThan(epoch),
      { timeout: 15000 },
    );
    // The client asks again for its own internal line on the reconnect.
    await waitFor(() => expect(answer).toHaveBeenCalled());
    rerender();
    await waitFor(() => expect(result.current.data?.has(adopted.key)).toBe(true), {
      timeout: 8000,
    });
    expect(Channel.getCachedBuiltIns(client)?.has(adopted.key)).toBe(true);
  }, 60000);

  it("should settle a failed fetch that a reconnect overtook", async () => {
    const { proxy, client } = await createProxiedTestClient({
      retry: FAST_RETRY,
      onInternalError: () => {},
    });
    const wrapper = await createAsyncSynnaxWrapper({ client });
    await waitForStreamLive(client.connection);
    const { unary } = client.transport;
    const send = unary.send.bind(unary);
    const answered = Promise.withResolvers<void>();
    vi.spyOn(unary, "send").mockImplementation(async (...args) => {
      if (args[0] === "/channel/retrieve-built-ins") {
        await answered.promise;
        throw new Error("refused");
      }
      return await send(...args);
    });
    const { result, rerender } = renderHook(() => Channel.useResultBuiltIns(), {
      wrapper,
    });
    await waitFor(() => expect(result.current.variant).toBe("loading"));
    // The stream reconnects while the fetch runs.
    const epoch = client.connection.status.details.epoch;
    await proxy.sever();
    await proxy.restore();
    await waitFor(
      () => expect(client.connection.status.details.epoch).toBeGreaterThan(epoch),
      { timeout: 15000 },
    );
    rerender();
    answered.resolve();
    await waitFor(() => expect(result.current.variant).not.toBe("loading"), {
      timeout: 8000,
    });
  }, 60000);
});
