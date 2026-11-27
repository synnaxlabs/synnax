// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Middleware } from "@synnaxlabs/freighter";
import { DataType, id, TimeSpan } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, onTestFinished } from "vitest";

import type { channel } from "@/channel";
import type { group } from "@/group";
import type { Synnax } from "@/index";
import type { resource } from "@/resource";
import { createTestClient } from "@/testutil";

const remote = createTestClient();

const createFolder = async (scope: resource.Type = "channel"): Promise<group.Group> =>
  await remote.groups.create({ name: id.create(), scope });

const createChannel = async (folder?: group.Group): Promise<channel.Channel> =>
  await remote.channels.create({
    name: `path_${id.create()}`,
    dataType: DataType.FLOAT32,
    virtual: true,
    group: folder?.key,
  });

/** Longer than the debounce of a refetch and the trip to the Core. */
const SETTLE = TimeSpan.milliseconds(600);

const settle = async () =>
  await new Promise((resolve) => setTimeout(resolve, SETTLE.milliseconds));

/**
 * Connects a client that counts the channel retrieves it sends. The change stream
 * told it of no folder made before.
 */
const connectCounting = async (): Promise<{
  client: Synnax;
  retrieves: () => number;
}> => {
  let count = 0;
  const counter: Middleware = async (ctx, next) => {
    if (ctx.target.endsWith("/channel/retrieve")) count++;
    return await next(ctx);
  };
  const client = createTestClient({ middleware: [counter] });
  onTestFinished(async () => await client.close());
  await client.connect();
  return { client, retrieves: () => count };
};

describe("channel path fetches", () => {
  beforeAll(async () => await remote.connect());

  it("should not fetch a watched answer again when a channel outside it is created in a folder the client has not read", async () => {
    const folder = await createFolder();
    const watched = await createChannel();
    const { client, retrieves } = await connectCounting();
    const live = client.channels.watch({
      where: { key: { in: [watched.key] } },
      includePath: true,
    });
    onTestFinished(live.onChange(() => {}));
    await live.retrieve();
    await settle();
    const before = retrieves();
    const outside = await createChannel(folder);
    await expect.poll(() => client.channels.store.has(outside.key)).toBe(true);
    await settle();
    expect(retrieves() - before).toEqual(0);
  });

  it("should not fetch a watched answer again each time a channel outside it is renamed", async () => {
    const folder = await createFolder();
    const watched = await createChannel();
    const outside = await createChannel(folder);
    const { client, retrieves } = await connectCounting();
    await client.channels.retrieve(outside.key);
    const live = client.channels.watch({
      where: { key: { in: [watched.key] } },
      includePath: true,
    });
    onTestFinished(live.onChange(() => {}));
    await live.retrieve();
    await settle();
    const before = retrieves();
    for (let i = 0; i < 3; i++) {
      const name = `path_${id.create()}`;
      await remote.channels.rename(outside.key, name);
      await expect
        .poll(() => client.channels.store.get(outside.key)?.name)
        .toEqual(name);
      await settle();
    }
    expect(retrieves() - before).toEqual(0);
  });

  it("should send one channel retrieve when a watch subscribes after its first retrieve", async () => {
    const folder = await createFolder();
    const ch = await createChannel(folder);
    const { client, retrieves } = await connectCounting();
    await client.channels.retrieve(ch.key);
    const before = retrieves();
    const live = client.channels.watch({
      where: { key: { in: [ch.key] } },
      includePath: true,
    });
    await live.retrieve();
    onTestFinished(live.onChange(() => {}));
    await settle();
    expect(retrieves() - before).toEqual(1);
  });

  it("should not fetch an answer that filters by path again when a folder of another scope is deleted", async () => {
    const folder = await createFolder();
    const ch = await createChannel(folder);
    const other = await createFolder("schematic");
    const { client, retrieves } = await connectCounting();
    await client.groups.retrieve({ key: other.key });
    const live = client.channels.watch({
      where: { path: { eq: `${folder.name}.${ch.name}` } },
    });
    onTestFinished(live.onChange(() => {}));
    await live.retrieve();
    await settle();
    const before = retrieves();
    await remote.groups.delete(other.key);
    await expect.poll(() => client.groups.store.has(other.key)).toBe(false);
    await settle();
    expect(retrieves() - before).toEqual(0);
  });

  it("should fetch an answer that filters by path again when a channel folder new to the client appears", async () => {
    const folder = await createFolder();
    const ch = await createChannel(folder);
    const { client, retrieves } = await connectCounting();
    const live = client.channels.watch({
      where: { path: { eq: `${folder.name}.${ch.name}` } },
    });
    onTestFinished(live.onChange(() => {}));
    await live.retrieve();
    await settle();
    const before = retrieves();
    const created = await createFolder();
    await expect.poll(() => client.groups.store.has(created.key)).toBe(true);
    await settle();
    // A new folder and a rename of a folder that the client never read reach the group
    // table alike, and the rename can change what the path matches.
    expect(retrieves() - before).toEqual(1);
  });
});
