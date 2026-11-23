// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import type { channel } from "@/channel";
import { NotFoundError } from "@/index";
import { createTestClient, secondsLinspace, spyOnSend } from "@/testutil";

const client = createTestClient();
const remote = createTestClient();

const virtual = (name: string): channel.New => ({
  name,
  dataType: DataType.FLOAT64,
  virtual: true,
});

const digits = (): string => `${Date.now()}${Math.floor(Math.random() * 1e6)}`;

describe("Channel lookups", () => {
  describe("delete", () => {
    it("should refuse a delete of a path that names no channel, naming the path", async () => {
      const path = `${id.create()}.${id.create()}`;
      await expect(client.channels.delete([path])).rejects.toThrow(
        `channels not found matching ${JSON.stringify([path])}`,
      );
    });

    it("should delete a list of a key and a name in one request", async () => {
      const byKey = await client.channels.create(virtual(id.create()));
      const byName = await client.channels.create(virtual(id.create()));
      const send = spyOnSend(client);
      try {
        await client.channels.delete([byKey.key, byName.name]);
        const deletes = send.mock.calls.filter(
          ([target]) => target === "/channel/delete",
        );
        expect(deletes.map(([, req]) => req)).toEqual([
          { keys: [byKey.key], names: [byName.name] },
        ]);
      } finally {
        send.mockRestore();
      }
      const fresh = createTestClient();
      expect(
        await fresh.channels.retrieve({
          where: { key: { in: [byKey.key, byName.key] } },
        }),
      ).toEqual([]);
    });

    it("should delete the channel at a path", async () => {
      const g = await client.groups.create({ name: id.create(), scope: "channel" });
      const ch = await client.channels.create({
        ...virtual(id.create()),
        group: g.key,
      });
      const path = `${g.name}.${ch.name}`;
      expect((await client.channels.retrieve(path)).key).toEqual(ch.key);
      await client.channels.delete(path);
      const fresh = createTestClient();
      expect(await fresh.channels.retrieve({ where: { key: { eq: ch.key } } })).toEqual(
        [],
      );
    });

    it("should delete the channel named by digit text in a mixed list, not the channel with that key", async () => {
      const other = await client.channels.create(virtual(id.create()));
      const victim = await client.channels.create(virtual(id.create()));
      const named = await client.channels.create(virtual(String(victim.key)));
      await client.channels.delete([other.key, named.name]);
      expect((await client.channels.retrieve(victim.key)).key).toEqual(victim.key);
      await expect(client.channels.retrieve(named.key)).rejects.toThrow(NotFoundError);
    });

    it("should delete the data of the channel named by digit text in a mixed list", async () => {
      const index = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const victim = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const named = await client.channels.create({
        name: String(victim.key),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      for (const ch of [index, victim, named])
        await client.write(TimeStamp.seconds(1), ch.key, secondsLinspace(1, 3));
      await client.delete([index.key, named.name], TimeRange.MAX);
      expect((await client.read(TimeRange.MAX, victim.key)).length).toEqual(3);
      expect((await client.read(TimeRange.MAX, named.key)).length).toEqual(0);
    });
  });

  describe("list retrieve", () => {
    it("should retrieve a digit name with the store cold", async () => {
      const name = digits();
      const made = await remote.channels.create(virtual(name));
      const local = createTestClient();
      const got = await local.channels.retrieve([name]);
      expect(got.map((c) => c.key)).toEqual([made.key]);
    });

    it("should retrieve a list of one cached key with no channel retrieve", async () => {
      const local = createTestClient();
      await local.connect();
      const ch = await local.channels.create(virtual(id.create()));
      const send = spyOnSend(local);
      try {
        const got = await local.channels.retrieve([ch.key]);
        expect(got.map((c) => c.key)).toEqual([ch.key]);
        const mention = new RegExp(`\\b${ch.key}\\b`);
        expect(
          send.mock.calls
            .filter(([target]) => target === "/channel/retrieve")
            .map(([, req]) => JSON.stringify(req))
            .filter((req) => mention.test(req)),
        ).toEqual([]);
      } finally {
        send.mockRestore();
      }
    });

    it("should retrieve a digit name and the channel whose key it spells", async () => {
      const other = await client.channels.create(virtual(id.create()));
      const named = await client.channels.create(virtual(String(other.key)));
      const byName = await client.channels.retrieve([named.name]);
      expect(byName.map((c) => c.key)).toEqual([named.key]);
      const single = await client.channels.retrieve(named.name);
      expect(single.key).toEqual(named.key);
      const mixed = await client.channels.retrieve([other.key, named.name]);
      expect(mixed.map((c) => c.key).sort()).toEqual([other.key, named.key].sort());
    });

    it("should read leading zeros as a name", async () => {
      const name = `00${digits()}`;
      const made = await client.channels.create(virtual(name));
      expect((await client.channels.retrieve(name)).key).toEqual(made.key);
      expect((await client.channels.retrieve([name])).map((c) => c.key)).toEqual([
        made.key,
      ]);
    });
  });
});
