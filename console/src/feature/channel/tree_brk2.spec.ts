// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connect, createServer, type Socket } from "node:net";

import { access, channel, DataType, query, Synnax, user } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
  FAST_RETRY,
  TEST_CLIENT_PARAMS,
  TEST_CORE_ADDRESS,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished } from "vitest";
import { z } from "zod";

import { Channel } from "@/feature/channel";
import { Tree } from "@/platform/tree";
import { createHostItems, HOST_ID, renderLiveTree } from "@/platform/tree/testutil";

const client = createTestClient();

const ROUTE = "POST /api/v1/channel/retrieve-built-ins";

// A TCP proxy in front of the test Core that, once armed, cuts the next response to the
// built-in keys after its headers. Every other request and the change stream pass.
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

const hostChannels = (...channels: { key: channel.Key }[]) =>
  createHostItems((c) => [
    Tree.createSource(
      c.channels,
      query.keysRequest(channels.map((ch) => ch.key)),
      (ch) => Tree.toResource("channel")(ch.payload),
    ),
  ]);

describe("channel/tree round 2 breaker", () => {
  it("should make a channel renamable after one cut response while the stream stays live", async () => {
    const ch = await client.channels.create({
      name: uniqueName("ch"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const proxy = await createCuttingProxy();
    const proxied = new Synnax({
      ...TEST_CLIENT_PARAMS,
      host: "localhost",
      port: proxy.port,
      retry: FAST_RETRY,
      onRetry: () => {},
    });
    onTestFinished(async () => await proxied.close());
    await proxied.connect();
    await waitForStreamLive(proxied.connection);
    const epoch = proxied.connection.status.details.epoch;
    proxy.fault.armed = true;
    await renderLiveTree({
      client: proxied,
      root: HOST_ID,
      items: { ...Channel.TREE_ITEMS, ...hostChannels(ch) },
    });
    await screen.findByText(ch.name);
    await waitFor(() => expect(proxy.fault.cuts).toBe(1));
    await new Promise((resolve) => setTimeout(resolve, 5000));
    expect(proxied.connection.status.details.streamLive).toBe(true);
    expect(proxied.connection.status.details.epoch).toBe(epoch);
    expect(screen.getByText(ch.name).className).toContain("lyra-text--editable");
  }, 30000);

  it("should make a channel renamable for a user whose stream the Core refuses", async () => {
    const ch = await client.channels.create({
      name: uniqueName("ch"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    // The user can read and rename the channel, but not read the change channels.
    const restricted = await createTestClientWithPolicy(client, [
      {
        name: uniqueName("policy"),
        objects: [
          user.TYPE_RESOURCE_ID,
          access.role.TYPE_RESOURCE_ID,
          access.policy.TYPE_RESOURCE_ID,
          channel.resourceID(ch.key),
        ],
        actions: ["retrieve"],
      },
      {
        name: uniqueName("policy"),
        objects: [channel.resourceID(ch.key)],
        actions: ["update"],
      },
    ]);
    const status = await restricted.connect();
    expect(status.details.streamDenied).toBe(true);
    // The Core answers this user's fetch of the built-in keys.
    const { keys } = await restricted.transport.unary.send(
      "/channel/retrieve-built-ins",
      {},
      z.object({}),
      z.object({ keys: z.array(z.number()) }),
    );
    expect(keys.length).toBeGreaterThan(0);
    await renderLiveTree({
      client: restricted,
      projectClient: client,
      root: HOST_ID,
      items: { ...Channel.TREE_ITEMS, ...hostChannels(ch) },
    });
    await screen.findByText(ch.name);
    await new Promise((resolve) => setTimeout(resolve, 3000));
    expect.soft(screen.getByText(ch.name).className).toContain("lyra-text--editable");
    expect
      .soft(restricted.channels.builtIns.state())
      .not.toEqual({ keys: null, error: null });
  }, 30000);
});
