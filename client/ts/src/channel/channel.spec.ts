// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, test, vi } from "vitest";

import { Channel } from "@/channel/client";
import { NotFoundError } from "@/errors";
import { query } from "@/query";
import { createTestClient, expectDeleted, expectLive, isLive } from "@/testutil";

const client = createTestClient();
const remote = createTestClient();

describe("Channel", () => {
  describe("create", () => {
    test("create one", async () => {
      const name = id.create();
      const channel = await client.channels.create({
        name,
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      expect(channel.name).toEqual(name);
      expect(channel.leaseholder).toEqual(1);
      expect(channel.virtual).toBe(true);
      expect(channel.dataType).toEqual(DataType.FLOAT32);
    });

    test("create calculated", async () => {
      const chOneName = id.create();
      let chOne = new Channel({
        name: chOneName,
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      chOne = await client.channels.create(chOne);
      let calculatedCH = new Channel({
        name: id.create(),
        virtual: true,
        dataType: DataType.FLOAT32,
        expression: `return ${chOne.name} * 2`,
      });
      calculatedCH = await client.channels.create(calculatedCH);
      expect(calculatedCH.key).not.toEqual(0);
      expect(calculatedCH.virtual).toEqual(true);
      expect(calculatedCH.expression).toEqual(`return ${chOneName} * 2`);
    });

    test("create calculated channel with constant expression", async () => {
      // Calculated channels can now have constant expressions that don't
      // reference other channels
      const calculatedCH = await client.channels.create({
        name: id.create(),
        virtual: true,
        dataType: DataType.FLOAT32,
        expression: `return 2`,
      });
      expect(calculatedCH.virtual).toEqual(true);
      expect(calculatedCH.expression).toEqual(`return 2`);
    });

    test("create index and indexed pair", async () => {
      const one = await client.channels.create({
        name: id.create(),
        isIndex: true,
        dataType: DataType.TIMESTAMP,
      });
      expect(one.key).not.toEqual(0);
      const two = await client.channels.create({
        name: id.create(),
        index: one.key,
        dataType: DataType.FLOAT32,
      });
      expect(two.key).not.toEqual(0);
    });

    test("create many", async () => {
      const names = [id.create(), id.create()];
      const channels = await client.channels.create([
        { name: names[0], leaseholder: 1, virtual: true, dataType: DataType.FLOAT32 },
        { name: names[1], leaseholder: 1, virtual: true, dataType: DataType.FLOAT32 },
      ]);
      expect(channels.length).toEqual(2);
      expect(channels[0].name).toEqual(names[0]);
      expect(channels[1].name).toEqual(names[1]);
    });

    test("create instances of channels", async () => {
      const timeIndexChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

      const sensorOne = new Channel({
        name: id.create(),
        dataType: DataType.FLOAT32,
        index: timeIndexChannel.key,
      });

      const sensorTwo = new Channel({
        name: id.create(),
        dataType: DataType.FLOAT32,
        index: timeIndexChannel.key,
      });

      const sensorThree = new Channel({
        name: id.create(),
        dataType: DataType.FLOAT32,
        index: timeIndexChannel.key,
      });
      await client.channels.create([sensorOne, sensorTwo, sensorThree]);
    });

    describe("virtual", () => {
      it("should create a virtual channel", async () => {
        const channel = await client.channels.create({
          name: id.create(),
          dataType: DataType.JSON,
          virtual: true,
        });
        expect(channel.virtual).toEqual(true);
        const retrieved = await client.channels.retrieve(channel.key);
        expect(retrieved.virtual).toBe(true);
      });
    });

    describe("retrieveIfNameExists", () => {
      it("should retrieve the existing channel when it exists", async () => {
        const name = id.create();
        const channel = await client.channels.create({
          name,
          leaseholder: 1,
          virtual: true,
          dataType: DataType.FLOAT32,
        });
        const channelTwo = await client.channels.create(
          { name, leaseholder: 1, virtual: true, dataType: DataType.FLOAT32 },
          { retrieveIfNameExists: true },
        );
        expect(channelTwo.key).toEqual(channel.key);
      });
      it("should create a new channel when it does not exist", async () => {
        const channel = await client.channels.create({
          name: id.create(),
          leaseholder: 1,
          virtual: true,
          dataType: DataType.FLOAT32,
        });
        const channelTwo = await client.channels.create(
          {
            name: id.create(),
            leaseholder: 1,
            virtual: true,
            dataType: DataType.FLOAT32,
          },
          { retrieveIfNameExists: true },
        );
        expect(channelTwo.key).not.toEqual(channel.key);
      });
      it("should retrieve and create the correct channels when creating many", async () => {
        const name = id.create();
        const channel = await client.channels.create({
          name,
          leaseholder: 1,
          virtual: true,
          dataType: DataType.FLOAT32,
        });
        const channelTwo = await client.channels.create(
          [
            { name, leaseholder: 1, virtual: true, dataType: DataType.FLOAT32 },
            {
              name: id.create(),
              leaseholder: 1,
              virtual: true,
              dataType: DataType.FLOAT32,
            },
          ],
          { retrieveIfNameExists: true },
        );
        expect(channelTwo.length).toEqual(2);
        expect(channelTwo[0].key).toEqual(channel.key);
        expect(channelTwo[1].key).not.toEqual(channel.key);
      });
    });
  });

  describe("retrieve", () => {
    test("retrieve by key", async () => {
      const channel = await client.channels.create({
        name: id.create(),
        leaseholder: 1,
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      const retrieved = await client.channels.retrieve(channel.key);
      expect(retrieved.name).toEqual(channel.name);
      expect(retrieved.leaseholder).toEqual(1);
      expect(retrieved.virtual).toEqual(true);
      expect(retrieved.dataType).toEqual(DataType.FLOAT32);
    });
    test("retrieve by key - not found", async () => {
      await expect(
        async () => await client.channels.retrieve("1-1000"),
      ).rejects.toThrow(NotFoundError);
    });
    test("retrieve by name", async () => {
      const name = id.create();
      await client.channels.create({
        name,
        leaseholder: 1,
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      const retrieved = await client.channels.retrieve([name]);
      expect(retrieved.length).toBe(1);
      expect(retrieved[0].name).toEqual(name);
    });
    test("retrieve by key - not found", async () => {
      await expect(
        async () => await client.channels.retrieve("1-1000"),
      ).rejects.toThrow(NotFoundError);
    });
    test("retrieve by search term", async () => {
      const prefix = id.create();
      const names = [`${prefix}_1`, `${prefix}_2`];
      await client.channels.create(
        names.map((name) => ({ name, virtual: true, dataType: DataType.FLOAT32 })),
      );
      await expect
        .poll(async () => {
          const results = await client.channels.retrieve({ searchTerm: prefix });
          return results.map((c) => c.name).sort();
        })
        .toEqual(names);
    });
  });

  describe("delete", async () => {
    test("delete by key", async () => {
      const channel = await client.channels.create({
        name: id.create(),
        leaseholder: 1,
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      await client.channels.delete(channel.key);
      await expect(
        async () => await client.channels.retrieve(channel.key),
      ).rejects.toThrow(NotFoundError);
    });
    test("delete by name", async () => {
      const channel = await client.channels.create({
        name: id.create(),
        leaseholder: 1,
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      await client.channels.delete([channel.name]);
      await expect(
        async () => await client.channels.retrieve(channel.key),
      ).rejects.toThrow(NotFoundError);
    });
  });
  describe("rename", async () => {
    test("single rename", async () => {
      const channel = await client.channels.create({
        name: id.create(),
        leaseholder: 1,
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      const newName = id.create();
      await client.channels.rename(channel.key, newName);
      const renamed = await client.channels.retrieve(channel.key);
      expect(renamed.name).toEqual(newName);
    });
    test("multiple rename", async () => {
      const names = [id.create(), id.create()];
      const channels = await client.channels.create([
        { name: names[0], leaseholder: 1, virtual: true, dataType: DataType.FLOAT32 },
        { name: names[1], leaseholder: 1, virtual: true, dataType: DataType.FLOAT32 },
      ]);
      // Retrieve channels here to ensure we check for cache invalidation
      const initial = await client.channels.retrieve(channels.map((c) => c.key));
      expect(initial[0].name).toEqual(names[0]);
      expect(initial[1].name).toEqual(names[1]);
      const newNames = [id.create(), id.create()];
      await client.channels.rename(
        channels.map((c) => c.key),
        newNames,
      );
      const renamed = await client.channels.retrieve(channels.map((c) => c.key));
      expect(renamed[0].name).toEqual(newNames[0]);
      expect(renamed[1].name).toEqual(newNames[1]);
    });
  });

  describe("update calculations", () => {
    test("update virtual channel expression", async () => {
      const channel = await client.channels.create({
        name: id.create(),
        virtual: true,
        dataType: DataType.INT64,
        expression: "return 1",
      });

      const updated = await client.channels.create({
        key: channel.key,
        name: channel.name,
        dataType: channel.dataType,
        leaseholder: channel.leaseholder,
        virtual: true,
        expression: "return 2",
      });

      const channelsWithName = await client.channels.retrieve([channel.name]);
      expect(channelsWithName.length).toEqual(1);

      expect(updated.expression).toEqual("return 2");

      const retrieved = await client.channels.retrieve(channel.key);
      expect(retrieved.expression).toEqual("return 2");
    });

    test("update calculated channel name", async () => {
      const channel = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
        expression: "return 1",
      });

      const updated = await client.channels.create({
        key: channel.key,
        name: id.create(),
        dataType: channel.dataType,
        virtual: true,
        expression: channel.expression,
      });
      expect(updated.name).toEqual(updated.name);

      const retrieved = await client.channels.retrieve(channel.key);
      expect(retrieved.name).toEqual(updated.name);
    });
  });
});

const createVirtual = async (c = client) =>
  await c.channels.create({
    name: `qry_${id.create()}`,
    dataType: DataType.FLOAT32,
    virtual: true,
  });

const createRange = async (c = client) => {
  const start = TimeStamp.now();
  return await c.ranges.create({
    name: `qry-${id.create()}`,
    timeRange: new TimeRange(start, start.add(TimeStamp.seconds(1))),
  });
};

describe("cached reads", () => {
  // Changes made while the stream is still opening are lost (epoch
  // reconciliation repairs them in production); open it up front so remote
  // writes in these specs are always observed.
  beforeAll(async () => await client.connect());

  describe("retrieve", () => {
    it("reflects remote changes on an unsubscribed repeat retrieve", async () => {
      const ch = await createVirtual();
      const first = await client.channels.retrieve(ch.key);
      expect(first.name).toEqual(ch.name);
      const renamed = `qry_renamed_${id.create()}`;
      await remote.channels.rename(ch.key, renamed);
      // Unsubscribed queries hold no frozen answer: repeat retrieves refetch
      // and converge on the remote change once it streams in.
      await expect
        .poll(async () => (await client.channels.retrieve(ch.key)).name)
        .toBe(renamed);
    });

    it("preserves key order across cached and fetched entries", async () => {
      const a = await createVirtual();
      const b = await createVirtual();
      await client.channels.retrieve(b.key);
      const res = await client.channels.retrieve([a.key, b.key]);
      expect(res.map((c) => c.key)).toEqual([a.key, b.key]);
    });
  });

  describe("getCached", () => {
    it("serves a key query straight from the record written by create", async () => {
      const ch = await createVirtual();
      const cached = expectLive(client.channels.getCached(ch.key));
      expect(cached.name).toEqual(ch.name);
    });
  });

  describe("onChange", () => {
    it("delivers a remote rename to a subscribed single query", async () => {
      const ch = await createVirtual();
      const handler = vi.fn();
      const off = client.channels.onChange(ch.key, handler);
      try {
        await client.channels.retrieve(ch.key);
        const renamed = `qry_renamed_${id.create()}`;
        await remote.channels.rename(ch.key, renamed);
        await expect
          .poll(() => {
            const cached = client.channels.getCached(ch.key);
            return isLive(cached) && cached.name === renamed;
          })
          .toBe(true);
        expect(handler).toHaveBeenCalled();
      } finally {
        off();
      }
    });

    it("delivers a remote delete as a deleted result carrying the corpse", async () => {
      const ch = await createVirtual();
      const results: Array<query.Cached<Channel> | undefined> = [];
      const off = client.channels.onChange(ch.key, (c) => results.push(c));
      try {
        await client.channels.retrieve(ch.key);
        await remote.channels.delete(ch.key);
        await expect
          .poll(() => query.Deleted.matches(client.channels.getCached(ch.key)))
          .toBe(true);
        const last = expectDeleted(results.at(-1));
        expect(last.corpse.name).toEqual(ch.name);
        await expect(client.channels.retrieve(ch.key)).rejects.toThrow("deleted");
      } finally {
        off();
      }
    });
  });

  describe("aliases", () => {
    it("composes the alias for a single query under a range", async () => {
      const ch = await createVirtual();
      const rng = await createRange();
      const alias = `qry_alias_${id.create()}`;
      await client.ranges.setAlias(rng.key, ch.key, alias);
      const res = await client.channels.retrieve(ch.key, { rangeKey: rng.key });
      expect(res.alias).toEqual(alias);
      const bare = await client.channels.retrieve(ch.key);
      expect(bare.alias).toBeUndefined();
    });

    it("delivers remote alias changes to a subscribed single query", async () => {
      const ch = await createVirtual();
      const rng = await createRange();
      const query = { key: ch.key, rangeKey: rng.key };
      const off = client.channels.onChange(query, () => {});
      try {
        await client.channels.retrieve(ch.key, { rangeKey: rng.key });
        const alias = `qry_alias_${id.create()}`;
        await remote.ranges.setAlias(rng.key, ch.key, alias);
        await expect
          .poll(() => {
            const cached = client.channels.getCached(query);
            return isLive(cached) ? cached.alias : undefined;
          })
          .toBe(alias);
        await remote.ranges.deleteAlias(rng.key, ch.key);
        await expect
          .poll(() => {
            const cached = client.channels.getCached(query);
            return isLive(cached) ? cached.alias : undefined;
          })
          .toBeUndefined();
      } finally {
        off();
      }
    });

    it("composes aliases for a keys request under a range", async () => {
      const a = await createVirtual();
      const b = await createVirtual();
      const rng = await createRange();
      const alias = `qry_alias_${id.create()}`;
      await client.ranges.setAlias(rng.key, a.key, alias);
      const res = await client.channels.retrieve({
        keys: [a.key, b.key],
        rangeKey: rng.key,
      });
      expect(res.find((c) => c.key === a.key)?.alias).toEqual(alias);
      expect(res.find((c) => c.key === b.key)?.alias).toBeUndefined();
    });
  });
});
