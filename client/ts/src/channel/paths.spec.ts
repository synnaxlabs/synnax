// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, onTestFinished } from "vitest";
import type { z } from "zod";

import { channel } from "@/channel";
import { group } from "@/group";
import type { Synnax } from "@/index";
import { createTestClient, createTestClientWithPolicy, spyOnSend } from "@/testutil";

const client = createTestClient();
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

/**
 * Connects a client that the change stream told of no folder made before, so it must
 * fetch the folders on each path.
 */
const connectLater = async (): Promise<Synnax> => {
  const later = createTestClient();
  onTestFinished(async () => await later.close());
  await later.connect();
  return later;
};

/** Watches the paths of the channels, in the order of the keys. */
const watchPaths = async (
  keys: channel.Key[],
  as: Synnax = client,
): Promise<() => Array<string | undefined>> => {
  const live = as.channels.watch({
    where: { key: { in: keys } },
    includePath: true,
  });
  let latest: channel.Channel[] = [];
  const off = live.onChange((w) => (latest = w?.records ?? []));
  onTestFinished(off);
  await live.retrieve();
  return () => keys.map((key) => latest.find((ch) => ch.key === key)?.path);
};

describe("channel paths", () => {
  // Changes made while the stream opens are lost, so open it first.
  beforeAll(async () => {
    await client.connect();
    await remote.connect();
  });

  describe("retrieve", () => {
    it("should fill the path of a channel at the top and of one in nested folders", async () => {
      const outer = await createFolder();
      const inner = await createFolder(outer);
      const [top, nested] = await Promise.all([createChannel(), createChannel(inner)]);
      const later = await connectLater();
      const res = await later.channels.retrieve({
        where: { key: { in: [top.key, nested.key] } },
        includePath: true,
      });
      expect(res.map((ch) => ch.path)).toEqual([
        top.name,
        `${outer.name}.${inner.name}.${nested.name}`,
      ]);
    });

    it("should leave the path absent on a retrieve that does not ask for it", async () => {
      const folder = await createFolder();
      const ch = await createChannel(folder);
      const where = { key: { in: [ch.key] } };
      await client.channels.retrieve({ where, includePath: true });
      const [bare] = await client.channels.retrieve({ where });
      expect(bare.path).toBeUndefined();
      expect((await client.channels.retrieve(ch.key)).path).toBeUndefined();
    });

    it("should fill the path of a channel retrieved by its key after a bare read", async () => {
      const folder = await createFolder();
      const ch = await createChannel(folder);
      const later = await connectLater();
      await later.channels.retrieve(ch.key);
      const res = await later.channels.retrieve({ key: ch.key, includePath: true });
      expect(res.path).toEqual(`${folder.name}.${ch.name}`);
    });

    it("should return the new path on a repeat retrieve after a folder rename", async () => {
      const folder = await createFolder();
      const ch = await createChannel(folder);
      const request = { where: { key: { in: [ch.key] } }, includePath: true };
      await client.channels.retrieve(request);
      const renamed = id.create();
      await remote.groups.rename(folder.key, renamed);
      await expect
        .poll(async () => (await client.channels.retrieve(request))[0].path)
        .toEqual(`${renamed}.${ch.name}`);
    });

    it("should leave the path absent when the caller cannot retrieve a folder on it", async () => {
      const folder = await createFolder();
      const ch = await createChannel(folder);
      const restricted = await createTestClientWithPolicy(remote, {
        name: "channel only",
        objects: [channel.resourceID(ch.key)],
        actions: ["retrieve"],
      });
      const res = await restricted.channels.retrieve({
        key: ch.key,
        includePath: true,
      });
      expect(res.name).toEqual(ch.name);
      expect(res.path).toBeUndefined();
    });
  });

  describe("requests", () => {
    it("should send no path in the create of a channel built from a retrieved one", async () => {
      const folder = await createFolder();
      const ch = await createChannel(folder);
      const [withPath] = await client.channels.retrieve({
        where: { key: { in: [ch.key] } },
        includePath: true,
      });
      expect(withPath.path).toEqual(`${folder.name}.${ch.name}`);
      const send = spyOnSend(client);
      onTestFinished(() => send.mockRestore());
      const copy = await client.channels.create({
        ...withPath.payload,
        key: undefined,
        name: `path_${id.create()}`,
        path: withPath.path,
      });
      onTestFinished(async () => await client.channels.delete(copy.key));
      // The request schema encodes the body, so the spec reads it as the wire does.
      const [[, body, schema]] = send.mock.calls.filter(
        ([target]) => target === "/channel/create",
      );
      const { channels } = (schema as z.ZodType<{ channels: object[] }>).parse(body);
      expect(channels).toHaveLength(1);
      expect(channels[0]).not.toHaveProperty("path");
      expect(withPath.payload).not.toHaveProperty("path");
    });
  });

  describe("getCached", () => {
    it("should hold a watched answer with a channel in a folder that the caller cannot retrieve", async () => {
      const folder = await createFolder();
      const ch = await createChannel(folder);
      const restricted = await createTestClientWithPolicy(remote, {
        name: "channel only",
        objects: [channel.resourceID(ch.key)],
        actions: ["retrieve"],
      });
      const live = restricted.channels.watch({
        where: { key: { in: [ch.key] } },
        includePath: true,
      });
      onTestFinished(live.onChange(() => {}));
      await live.retrieve();
      const cached = live.getCached();
      expect(cached?.records.map((c) => [c.key, c.path])).toEqual([
        [ch.key, undefined],
      ]);
    });
  });

  describe("watch", () => {
    it("should show the new path after another client renames a folder", async () => {
      const outer = await createFolder();
      const inner = await createFolder(outer);
      const ch = await createChannel(inner);
      const paths = await watchPaths([ch.key]);
      expect(paths()).toEqual([`${outer.name}.${inner.name}.${ch.name}`]);
      const renamed = id.create();
      await remote.groups.rename(outer.key, renamed);
      await expect.poll(paths).toEqual([`${renamed}.${inner.name}.${ch.name}`]);
    });

    it("should show the new path after another client moves a folder", async () => {
      const [from, to] = await Promise.all([createFolder(), createFolder()]);
      const inner = await createFolder(from);
      const ch = await createChannel(inner);
      const paths = await watchPaths([ch.key]);
      await remote.groups.setParent(inner.key, group.resourceID(to.key));
      await expect.poll(paths).toEqual([`${to.name}.${inner.name}.${ch.name}`]);
    });

    it("should show the new path after another client renames the channel", async () => {
      const folder = await createFolder();
      const ch = await createChannel(folder);
      const paths = await watchPaths([ch.key]);
      const renamed = `path_${id.create()}`;
      await remote.channels.rename(ch.key, renamed);
      await expect.poll(paths).toEqual([`${folder.name}.${renamed}`]);
    });

    it("should show the path after another client moves the channel into a folder this client has not read", async () => {
      const ch = await createChannel();
      const outer = await createFolder();
      const inner = await createFolder(outer);
      const paths = await watchPaths([ch.key], await connectLater());
      expect(paths()).toEqual([ch.name]);
      await remote.batch.send(remote.channels.setGroupRequest(ch.key, inner.key));
      await expect.poll(paths).toEqual([`${outer.name}.${inner.name}.${ch.name}`]);
    });

    it("should show the path after this client moves the channel into a folder it has not read", async () => {
      const ch = await createChannel();
      const folder = await createFolder();
      const later = await connectLater();
      const paths = await watchPaths([ch.key], later);
      await later.batch.send(later.channels.setGroupRequest(ch.key, folder.key));
      await expect.poll(paths).toEqual([`${folder.name}.${ch.name}`]);
    });

    it("should show the name alone after the channel moves to the top", async () => {
      const folder = await createFolder();
      const ch = await createChannel(folder);
      const paths = await watchPaths([ch.key]);
      await remote.batch.send(remote.channels.setGroupRequest(ch.key));
      await expect.poll(paths).toEqual([ch.name]);
    });

    it("should not notify a watched answer when a channel folder off its paths changes", async () => {
      const [folder, other] = await Promise.all([createFolder(), createFolder()]);
      const ch = await createChannel(folder);
      const live = client.channels.watch({
        where: { key: { in: [ch.key] } },
        includePath: true,
      });
      let notices = 0;
      onTestFinished(live.onChange(() => notices++));
      await live.retrieve();
      await expect.poll(() => client.groups.store.has(other.key)).toBe(true);
      const before = notices;
      await remote.groups.rename(other.key, id.create());
      await createFolder(other);
      await expect
        .poll(() => client.groups.store.get(other.key)?.name)
        .not.toEqual(other.name);
      await new Promise((resolve) => setTimeout(resolve, 300));
      expect(notices).toEqual(before);
    });

    it("should fetch again an answer that filters by path after a folder rename", async () => {
      const folder = await createFolder();
      const ch = await createChannel(folder);
      const live = client.channels.watch({
        where: { path: { eq: `${folder.name}.${ch.name}` } },
      });
      let keys: channel.Key[] = [];
      onTestFinished(live.onChange((w) => (keys = w?.records.map((c) => c.key) ?? [])));
      await live.retrieve();
      expect(keys).toEqual([ch.key]);
      await remote.groups.rename(folder.key, id.create());
      await expect.poll(() => keys).toEqual([]);
    });
  });
});
