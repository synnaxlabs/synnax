// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Middleware } from "@synnaxlabs/freighter";
import { DataType, id, sleep, TimeSpan } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, onTestFinished } from "vitest";

import type { channel } from "@/channel";
import { group } from "@/group";
import type { Synnax } from "@/index";
import { createTestClient } from "@/testutil";

const remote = createTestClient();

const createFolder = async (parent?: group.Group): Promise<group.Group> =>
  await remote.groups.create(
    parent == null
      ? { name: id.create(), scope: "channel" }
      : { name: id.create(), parent: group.resourceID(parent.key) },
  );

const createChannel = async (folder?: group.Group): Promise<channel.Channel> =>
  await remote.channels.create({
    name: `path_${id.create()}`,
    dataType: DataType.FLOAT32,
    virtual: true,
    group: folder?.key,
  });

/** Connects a client that the change stream told of no folder made before. */
const connectLater = async (): Promise<Synnax> => {
  const later = createTestClient();
  onTestFinished(async () => await later.close());
  await later.connect();
  return later;
};

const watchPath = async (
  key: channel.Key,
  as: Synnax,
): Promise<() => string | undefined> => {
  const live = as.channels.watch({ where: { key: { in: [key] } }, includePath: true });
  let latest: channel.Channel[] = [];
  onTestFinished(live.onChange((w) => (latest = w?.records ?? [])));
  await live.retrieve();
  return () => latest.find((ch) => ch.key === key)?.path;
};

describe("channel paths under a folder the client has not read", () => {
  beforeAll(async () => await remote.connect());

  it("should show the path after another client moves the channel's folder under it", async () => {
    const [outer, folder] = await Promise.all([createFolder(), createFolder()]);
    const ch = await createChannel(folder);
    const later = await connectLater();
    const path = await watchPath(ch.key, later);
    expect(path()).toEqual(`${folder.name}.${ch.name}`);
    await remote.groups.setParent(folder.key, group.resourceID(outer.key));
    const core = await connectLater();
    const [fresh] = await core.channels.retrieve({
      where: { key: { in: [ch.key] } },
      includePath: true,
    });
    expect(fresh.path).toEqual(`${outer.name}.${folder.name}.${ch.name}`);
    await expect.poll(path).toEqual(`${outer.name}.${folder.name}.${ch.name}`);
  });

  it("should show the path after this client moves the channel's folder under it", async () => {
    const [outer, folder] = await Promise.all([createFolder(), createFolder()]);
    const ch = await createChannel(folder);
    const later = await connectLater();
    const path = await watchPath(ch.key, later);
    await later.groups.setParent(folder.key, group.resourceID(outer.key));
    await expect.poll(path).toEqual(`${outer.name}.${folder.name}.${ch.name}`);
  });

  it("should show the path when this client's move commits during a read of the folders above it", async () => {
    const [outer, folder] = await Promise.all([createFolder(), createFolder()]);
    const ch = await createChannel(folder);
    // The read of the folders above starts before the move commits and ends after
    // the echo of the move.
    const order: Middleware = async (ctx, next) => {
      if (ctx.target.endsWith("/batch")) await sleep.sleep(TimeSpan.milliseconds(200));
      const res = await next(ctx);
      if (ctx.target.endsWith("/group/retrieve-ancestors"))
        await sleep.sleep(TimeSpan.milliseconds(800));
      return res;
    };
    const later = createTestClient({ middleware: [order] });
    await later.connect();
    const path = await watchPath(ch.key, later);
    await later.groups.setParent(folder.key, group.resourceID(outer.key));
    await expect.poll(path).toEqual(`${outer.name}.${folder.name}.${ch.name}`);
  });
});
