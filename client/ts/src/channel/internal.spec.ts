// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, observe } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { type channel } from "@/channel";
import {
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
  waitForStreamLive,
} from "@/testutil";

const remote = createTestClient();

const chainOf = (err: unknown): string => {
  const parts: string[] = [];
  let current: unknown = err;
  while (current instanceof Error) {
    parts.push(current.message);
    current = current.cause;
  }
  return parts.join(" <- ");
};

describe("Internal line across a reconnect", () => {
  it("should keep an internal-line list live while built-in keys reload", async () => {
    const internal: Error[] = [];
    const { proxy, client } = await createProxiedTestClient({
      retry: FAST_RETRY,
      onRetry: () => {},
      onInternalError: (error) => internal.push(error),
    });
    const original = client.channels.retrieveBuiltIns.bind(client.channels);
    const builtIns = vi.spyOn(client.channels, "retrieveBuiltIns");
    await client.connect();
    await waitForStreamLive(client.connection);
    await expect.poll(() => builtIns.mock.calls.length).toBeGreaterThanOrEqual(1);
    await builtIns.mock.results[0].value;
    // The reply of the next connection is slow.
    let release: () => void = () => {};
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    builtIns.mockImplementation(async (): Promise<channel.Key[]> => {
      await gate;
      return await original();
    });
    const prefix = `internal_${id.create()}`;
    const list = client.channels.watch({
      where: { internal: { eq: false }, name: { startsWith: prefix } },
    });
    const off = list.onChange(() => {});
    await list.retrieve();
    await proxy.sever();
    await observe.until(
      client.connection,
      () => client.connection.status,
      ({ details }) => !details.streamLive,
    );
    await proxy.restore();
    await waitForStreamLive(client.connection);
    await expect.poll(() => builtIns.mock.calls.length).toBe(2);
    const ch = await remote.channels.create({
      name: `${prefix}_a`,
      dataType: "float32",
      virtual: true,
    });
    try {
      await expect
        .poll(() => list.getCached()?.records.map((c) => c.key) ?? [], {
          timeout: 5000,
        })
        .toContain(ch.key)
        .catch((e: unknown) => {
          const errors = JSON.stringify(internal.map(chainOf));
          throw new Error(`${String(e)}; internal errors: ${errors}`);
        });
      expect(internal.map(chainOf)).toEqual([]);
    } finally {
      release();
      off();
    }
  }, 30000);
});
