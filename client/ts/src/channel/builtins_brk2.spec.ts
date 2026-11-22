// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connect, createServer, type Socket } from "node:net";

import { id } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { type channel } from "@/channel";
import Synnax from "@/client";
import {
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
  recordResponses,
  TEST_CLIENT_PARAMS,
  TEST_CORE_ADDRESS,
  waitForStreamLive,
} from "@/testutil";

const remote = createTestClient();

const ROUTE = "POST /api/v1/channel/retrieve-built-ins";

// A TCP proxy in front of the test Core that, once armed, cuts the next response to the
// built-in keys after its headers, as a link that drops mid-response does. Every other
// request and the change stream pass through.
const createCuttingProxy = async () => {
  const sockets = new Set<Socket>();
  const fault = { armed: false, cuts: 0 };
  const server = createServer((downstream) => {
    const upstream = connect(TEST_CORE_ADDRESS.port, TEST_CORE_ADDRESS.host);
    sockets.add(downstream);
    sockets.add(upstream);
    let cutNext = false;
    downstream.on("data", (chunk: Buffer) => {
      if (fault.armed && chunk.toString("latin1").includes(ROUTE)) {
        fault.armed = false;
        cutNext = true;
      }
      upstream.write(chunk);
    });
    upstream.on("data", (chunk: Buffer) => {
      if (!cutNext) return void downstream.write(chunk);
      const end = chunk.indexOf("\r\n\r\n");
      downstream.write(chunk.subarray(0, end + 4), () => {
        fault.cuts++;
        downstream.destroy();
        upstream.destroy();
      });
    });
    const destroy = () => {
      downstream.destroy();
      upstream.destroy();
    };
    for (const s of [downstream, upstream]) {
      s.on("error", destroy);
      s.on("close", destroy);
    }
  });
  await new Promise<void>((resolve) => server.listen(0, resolve));
  const address = server.address();
  if (address == null || typeof address === "string") throw new Error("no port");
  onTestFinished(async () => {
    sockets.forEach((s) => s.destroy());
    await new Promise<void>((resolve) => server.close(() => resolve()));
  });
  return { port: address.port, fault };
};

describe("BuiltIns round 2 breaker", () => {
  it("should settle the keys after one response is cut while the stream stays live", async () => {
    const proxy = await createCuttingProxy();
    const client = new Synnax({
      ...TEST_CLIENT_PARAMS,
      host: "localhost",
      port: proxy.port,
      retry: FAST_RETRY,
      onRetry: () => {},
    });
    onTestFinished(async () => await client.close());
    await client.connect();
    await waitForStreamLive(client.connection);
    const epoch = client.connection.status.details.epoch;
    proxy.fault.armed = true;
    expect(client.channels.builtIns.state()).toEqual({ keys: null, error: null });
    await expect.poll(() => proxy.fault.cuts).toBe(1);
    // The link is healthy again: the stream stays live and other requests answer.
    const ch = await remote.channels.create({
      name: `brk2_${id.create()}`,
      dataType: "float32",
      virtual: true,
    });
    expect((await client.channels.retrieve(ch.key)).key).toBe(ch.key);
    await new Promise((resolve) => setTimeout(resolve, 5000));
    expect(client.connection.status.details.streamLive).toBe(true);
    expect(client.connection.status.details.epoch).toBe(epoch);
    // The keys must not stay pending for the rest of the connection.
    const { keys, error } = client.channels.builtIns.state();
    expect(keys != null || error != null).toBe(true);
  }, 30000);

  const PAGE_COUNT = 1010;

  const createPagedChannels = async (): Promise<string> => {
    const prefix = `brk2_page_${id.create()}_`;
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

  it("should page a non-internal list to the end when the keys already arrived", async () => {
    const prefix = await createPagedChannels();
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    await client.channels.builtIns.retrieve();
    expect(await pageToEnd(client, prefix)).toBe(PAGE_COUNT);
  }, 60000);

  it("should page a non-internal list to the end when it is the first read of the keys", async () => {
    const prefix = await createPagedChannels();
    const client = createTestClient();
    await client.connect();
    await waitForStreamLive(client.connection);
    expect(await pageToEnd(client, prefix)).toBe(PAGE_COUNT);
  }, 60000);

  it("should answer retrieve on a client with no change stream", async () => {
    const client = createTestClient({ cache: false });
    await client.connect();
    const timeout = new Promise<"timeout">((resolve) =>
      setTimeout(() => resolve("timeout"), 5000),
    );
    const result = await Promise.race([client.channels.builtIns.retrieve(), timeout]);
    expect(result).toBeInstanceOf(Set);
  }, 15000);

  it("should keep the state across a reconnect that fetches the same keys", async () => {
    const { proxy, client } = await createProxiedTestClient({
      retry: FAST_RETRY,
      onRetry: () => {},
    });
    await client.connect();
    await waitForStreamLive(client.connection);
    await client.channels.builtIns.retrieve();
    const before = client.channels.builtIns.state();
    const changes: channel.BuiltInsState[] = [];
    client.channels.builtIns.onChange((s) => changes.push(s));
    const responses = recordResponses(client, "/channel/retrieve-built-ins");
    const epoch = client.connection.status.details.epoch;
    await proxy.sever();
    await proxy.restore();
    await expect
      .poll(() => client.connection.status.details.epoch, { timeout: 15000 })
      .toBeGreaterThan(epoch);
    await expect.poll(() => responses.length, { timeout: 8000 }).toBe(1);
    await responses[0];
    await new Promise((resolve) => setTimeout(resolve, 100));
    expect(client.channels.builtIns.get()).toEqual(before.keys);
    expect(changes).toEqual([]);
    expect(client.channels.builtIns.state()).toBe(before);
  }, 30000);

  // A TCP proxy whose target Core can change, as a new Core behind one address does.
  const createSwitchingProxy = async (port: number) => {
    const target = { port };
    const sockets = new Set<Socket>();
    const server = createServer((downstream) => {
      const upstream = connect(target.port, TEST_CORE_ADDRESS.host);
      sockets.add(downstream);
      sockets.add(upstream);
      downstream.pipe(upstream);
      upstream.pipe(downstream);
      const destroy = () => {
        downstream.destroy();
        upstream.destroy();
      };
      for (const s of [downstream, upstream]) {
        s.on("error", destroy);
        s.on("close", destroy);
      }
    });
    await new Promise<void>((resolve) => server.listen(0, resolve));
    const address = server.address();
    if (address == null || typeof address === "string") throw new Error("no port");
    onTestFinished(async () => {
      sockets.forEach((s) => s.destroy());
      await new Promise<void>((resolve) => server.close(() => resolve()));
    });
    return {
      port: address.port,
      switchTo: (next: number) => {
        target.port = next;
        sockets.forEach((s) => s.destroy());
        sockets.clear();
      },
    };
  };

  it("should drop the keys of a replaced Core before the new Core answers", async () => {
    const second = Number(process.env.BRK2_SECOND_CORE_PORT);
    if (!Number.isFinite(second)) throw new Error("BRK2_SECOND_CORE_PORT is not set");
    const proxy = await createSwitchingProxy(TEST_CORE_ADDRESS.port);
    const client = new Synnax({
      ...TEST_CLIENT_PARAMS,
      host: "localhost",
      port: proxy.port,
      retry: FAST_RETRY,
      onRetry: () => {},
      onInternalError: () => {},
    });
    onTestFinished(async () => await client.close());
    await client.connect();
    await waitForStreamLive(client.connection);
    const old = await client.channels.builtIns.retrieve();
    const oldCluster = client.connection.status.details.clusterKey;
    const epochs: number[] = [];
    client.connection.onChange(({ details }) => epochs.push(details.epoch));
    // The new Core's answer waits, so only the reset can clear the keys.
    const released = Promise.withResolvers<void>();
    const { unary } = client.transport;
    const send = unary.send.bind(unary);
    const spy = vi.spyOn(unary, "send").mockImplementation(async (...args) => {
      if (args[0] === "/channel/retrieve-built-ins") await released.promise;
      return await send(...args);
    });
    onTestFinished(() => {
      released.resolve();
      spy.mockRestore();
    });
    proxy.switchTo(second);
    await expect
      .poll(() => client.connection.status.details.clusterKey, { timeout: 20000 })
      .not.toBe(oldCluster);
    await waitForStreamLive(client.connection);
    // The client reset its cache for the new Core.
    expect(epochs).toContain(0);
    const now = client.channels.builtIns.get();
    expect(now === old ? "keys of the replaced Core" : now).toBeNull();
  }, 40000);
});
