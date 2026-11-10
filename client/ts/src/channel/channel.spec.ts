// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  control,
  DataType,
  id,
  sleep,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, test, vi } from "vitest";

import { channel } from "@/channel";
import { NotFoundError, PathError, ValidationError } from "@/index";
import { query } from "@/query";
import { type ranger } from "@/ranger";
import {
  createProxiedTestClient,
  createTestClient,
  expectDeleted,
  expectLive,
  failRequests,
  FAST_RETRY,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";

const quiet = (): void => {};

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
      let chOne = new channel.Channel({
        name: chOneName,
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      chOne = await client.channels.create(chOne);
      let calculatedCH = new channel.Channel({
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

    test("create bool channel", async () => {
      const idx = await client.channels.create({
        name: id.create(),
        isIndex: true,
        dataType: DataType.TIMESTAMP,
      });
      const ch = await client.channels.create({
        name: id.create(),
        index: idx.key,
        dataType: DataType.BOOLEAN,
      });
      expect(ch.dataType.equals(DataType.BOOLEAN)).toBe(true);
      const retrieved = await client.channels.retrieve(ch.key);
      expect(retrieved.dataType.equals(DataType.BOOLEAN)).toBe(true);
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

      const sensorOne = new channel.Channel({
        name: id.create(),
        dataType: DataType.FLOAT32,
        index: timeIndexChannel.key,
      });

      const sensorTwo = new channel.Channel({
        name: id.create(),
        dataType: DataType.FLOAT32,
        index: timeIndexChannel.key,
      });

      const sensorThree = new channel.Channel({
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
      it("should not create a duplicate channel with the same name", async () => {
        const name = id.create();
        const props = {
          name,
          leaseholder: 1,
          virtual: true,
          dataType: DataType.FLOAT32,
        };
        const channel = await client.channels.create(props);
        await client.channels.create(props, { retrieveIfNameExists: true });
        const retrieved = await client.channels.retrieve({
          where: { name: { eq: name } },
        });
        expect(retrieved.map((c) => c.key)).toEqual([channel.key]);
      });
      it("should create one channel when creates of the same name run at once", async () => {
        const name = id.create();
        const props = {
          name,
          leaseholder: 1,
          virtual: true,
          dataType: DataType.FLOAT32,
        };
        const created = await Promise.all(
          Array.from({ length: 10 }, () =>
            client.channels.create(props, { retrieveIfNameExists: true }),
          ),
        );
        const keys = new Set(created.map((c) => c.key));
        expect(keys.size).toEqual(1);
        const retrieved = await client.channels.retrieve({
          where: { name: { eq: name } },
        });
        expect(retrieved.map((c) => c.key)).toEqual([...keys]);
      });
      it("should return the channels in the order they were passed", async () => {
        const name = id.create();
        const channel = await client.channels.create({
          name,
          leaseholder: 1,
          virtual: true,
          dataType: DataType.FLOAT32,
        });
        const newName = id.create();
        const created = await client.channels.create(
          [
            {
              name: newName,
              leaseholder: 1,
              virtual: true,
              dataType: DataType.FLOAT32,
            },
            { name, leaseholder: 1, virtual: true, dataType: DataType.FLOAT32 },
          ],
          { retrieveIfNameExists: true },
        );
        expect(created.map((c) => c.name)).toEqual([newName, name]);
        expect(created[1].key).toEqual(channel.key);
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
    test("reject a condition whose operator is undefined", async () => {
      const name: string | undefined = undefined;
      await expect(
        client.channels.retrieve({ where: { name: { eq: name } } }),
      ).rejects.toThrow(
        new PathError(
          ["where", "name"],
          new ValidationError("no operator set: validation error"),
        ),
      );
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
    test("retrieve by where", async () => {
      const prefix = `qry_${id.create()}`;
      const [a, , c] = await client.channels.create([
        { name: `${prefix}_a`, virtual: true, dataType: DataType.FLOAT32 },
        { name: `${prefix}_b`, virtual: true, dataType: DataType.FLOAT64 },
        { name: `${prefix}_c`, virtual: true, dataType: DataType.FLOAT32 },
      ]);
      const res = await client.channels.retrieve({
        where: { name: { regex: `^${prefix}` }, dataType: { eq: DataType.FLOAT32 } },
      });
      expect(res.map((ch) => ch.key).sort()).toEqual([a.key, c.key].sort());
    });
    test("retrieve by a flat filter", async () => {
      const prefix = `qry_${id.create()}`;
      const [a, , c] = await client.channels.create([
        { name: `${prefix}_a`, virtual: true, dataType: DataType.FLOAT32 },
        { name: `${prefix}_b`, virtual: true, dataType: DataType.FLOAT64 },
        { name: `${prefix}_c`, virtual: true, dataType: DataType.FLOAT32 },
      ]);
      const res = await client.channels.retrieve({
        name: { regex: `^${prefix}` },
        dataType: DataType.FLOAT32,
      });
      expect(res.map((ch) => ch.key).sort()).toEqual([a.key, c.key].sort());
    });
    test("match each name in a list exactly, never as a pattern", async () => {
      const prefix = `qry_${id.create()}`;
      const [a] = await client.channels.create([
        { name: `${prefix}_a`, virtual: true, dataType: DataType.FLOAT32 },
        { name: `${prefix}_b`, virtual: true, dataType: DataType.FLOAT32 },
      ]);
      const res = await client.channels.retrieve([a.name, `${prefix}_.*`]);
      expect(res.map((ch) => ch.key)).toEqual([a.key]);
    });
    test("retrieve virtual channels without calculated channels", async () => {
      const prefix = `qry_${id.create()}`;
      const virtual = await client.channels.create({
        name: `${prefix}_virtual`,
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      const where = {
        name: { in: [`${prefix}_virtual`, `${prefix}_calculated`] },
        virtual: true,
      };
      expect((await client.channels.retrieve({ where })).map((ch) => ch.key)).toEqual([
        virtual.key,
      ]);
      await client.channels.create({
        name: `${prefix}_calculated`,
        virtual: true,
        dataType: DataType.FLOAT32,
        expression: "return 1",
      });
      expect((await client.channels.retrieve({ where })).map((ch) => ch.key)).toEqual([
        virtual.key,
      ]);
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
    test("refuse a channel that a task config names", async () => {
      const channel = await client.channels.create({
        name: id.create(),
        leaseholder: 1,
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      const rack = await client.racks.create({ name: "test" });
      const device = await client.devices.create({
        key: id.create(),
        rack: rack.key,
        location: "127.0.0.1:502",
        name: "plc",
        make: "modbus",
        model: "modbus",
        properties: {},
      });
      const task = await rack.createTask({
        name: "read",
        type: "modbus_read",
        config: {
          device: device.key,
          channels: [{ type: "coil", key: "coil", channel: channel.key }],
        },
      });
      await expect(client.channels.delete(channel.key)).rejects.toThrow(
        new ValidationError(
          `cannot delete channel ${channel.name} because task ${task.name} uses it: ` +
            "validation error",
        ),
      );
      await client.tasks.delete(task.key);
      await client.channels.delete(channel.key);
    });
    test("refuse an index that a data channel uses", async () => {
      const index = await client.channels.create({
        name: id.create(),
        leaseholder: 1,
        isIndex: true,
        dataType: DataType.TIMESTAMP,
      });
      const data = await client.channels.create({
        name: id.create(),
        leaseholder: 1,
        index: index.key,
        dataType: DataType.FLOAT32,
      });
      await expect(client.channels.delete(index.key)).rejects.toThrow(
        new ValidationError(
          `cannot delete channel ${index.name} because channel ${data.name} uses ` +
            "it as its index: validation error",
        ),
      );
      await client.channels.delete(data.key);
      await client.channels.delete(index.key);
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

  describe("setGroupRequest", () => {
    it("should move channels into a group and back out", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        dataType: "float32",
        virtual: true,
      });
      const g = await client.groups.create({ name: id.create(), scope: "channel" });
      await client.batch.send(client.channels.setGroupRequest(ch.key, g.key));
      expect((await client.channels.retrieve(ch.key)).group).toEqual(g.key);
      expect(
        (await client.channels.retrieve({ groups: [g.key] })).map((c) => c.key),
      ).toEqual([ch.key]);
      await client.batch.send(client.channels.setGroupRequest(ch.key, null));
      expect(await client.channels.retrieve({ groups: [g.key] })).toEqual([]);
    });
  });
  describe("dispatch", () => {
    it("should apply the actions to the channel", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        dataType: "float32",
        virtual: true,
      });
      const g = await client.groups.create({ name: id.create(), scope: "channel" });
      const name = id.create();
      await client.channels.dispatch(ch.key, [
        channel.rename({ name }),
        channel.set({ path: ["group"], value: g.key }),
      ]);
      expect(await createTestClient().channels.retrieve(ch.key)).toMatchObject({
        name,
        group: g.key,
      });
    });

    it("should cache the channel as the Core holds it before it resolves", async () => {
      // A client with no change stream gets no echo of the dispatch.
      const unstreamed = createTestClient({ cache: false });
      const ch = await unstreamed.channels.create({
        name: id.create(),
        dataType: DataType.INT64,
        virtual: true,
        expression: "return 1",
      });
      await unstreamed.channels.dispatch(
        ch.key,
        channel.set({ path: ["expression"], value: "return 1.5" }),
      );
      const cached = unstreamed.channels.store.get(ch.key);
      expect(cached?.expression).toEqual("return 1.5");
      expect(cached?.dataType).toEqual(DataType.FLOAT64);
    });

    it("should resolve and report a failed fetch after the send", async () => {
      const onInternalError = vi.fn();
      const local = createTestClient({ onInternalError });
      const ch = await local.channels.create({
        name: id.create(),
        dataType: DataType.INT64,
        virtual: true,
        expression: "return 1",
      });
      const fetchError = new Error("fetch failed");
      failRequests(local, "/channel/retrieve", fetchError);
      await local.channels.dispatch(
        ch.key,
        channel.set({ path: ["expression"], value: "return 2" }),
      );
      expect(onInternalError).toHaveBeenCalledWith(
        expect.objectContaining({
          message: "failed to fetch a dispatched channel",
          cause: fetchError,
        }),
      );
    });

    it("should keep the cached channel when the Core rejects the actions", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        dataType: DataType.INT64,
        virtual: true,
        expression: "return 1",
      });
      const before = client.channels.store.get(ch.key);
      await expect(
        client.channels.dispatch(
          ch.key,
          channel.set({ path: ["expression"], value: "" }),
        ),
      ).rejects.toThrow("its expression cannot be empty");
      expect(client.channels.store.get(ch.key)).toBe(before);
    });
  });
  describe("update calculations", () => {
    test("update virtual channel expression", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        virtual: true,
        dataType: DataType.INT64,
        expression: "return 1",
      });
      await client.channels.dispatch(
        ch.key,
        channel.set({ path: ["expression"], value: "return 2" }),
      );
      const fresh = createTestClient();
      const channelsWithName = await fresh.channels.retrieve([ch.name]);
      expect(channelsWithName.length).toEqual(1);
      const retrieved = await fresh.channels.retrieve(ch.key);
      expect(retrieved.expression).toEqual("return 2");
    });

    test("update calculated channel name", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
        expression: "return 1",
      });
      const name = id.create();
      await client.channels.rename(ch.key, name);
      const retrieved = await client.channels.retrieve(ch.key);
      expect(retrieved.name).toEqual(name);
      expect(retrieved.expression).toEqual("return 1");
    });
  });

  describe("payload", () => {
    it("keeps shared concurrency through a round trip", () => {
      const ch = new channel.Channel({
        name: "shared",
        dataType: DataType.FLOAT32,
        virtual: true,
        concurrency: control.Concurrency.shared,
      });
      expect(ch.payload.concurrency).toEqual(control.Concurrency.shared);
      expect(new channel.Channel(ch.payload).concurrency).toEqual(
        control.Concurrency.shared,
      );
    });
  });

  describe("constructor normalization", () => {
    it("fills the defaults omitted from a crude operation", () => {
      const [op] = new channel.Channel({
        name: "op",
        dataType: DataType.FLOAT32,
        operations: [{ type: "avg" }],
      }).operations;
      expect(op.resetChannel).toEqual(0);
      expect(op.duration).toEqual(TimeSpan.ZERO);
    });

    it("parses a crude status time into a TimeStamp", () => {
      const date = new Date("2026-08-10T00:00:00.000Z");
      const ch = new channel.Channel({
        name: "stat",
        dataType: DataType.FLOAT32,
        status: { variant: "error", message: "boom", time: date },
      });
      expect(ch.status?.time).toBeInstanceOf(TimeStamp);
      expect(ch.status?.time).toEqual(new TimeStamp(date));
    });

    it("throws when the status is malformed", () => {
      expect(
        () =>
          new channel.Channel({
            name: "bad",
            dataType: DataType.FLOAT32,
            status: { variant: "not_a_variant" } as never,
          }),
      ).toThrow();
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

    // A plot panel resolves one channel per telemetry source in the same tick, so a
    // single retrieve that skips the record table costs one request per channel.
    it("coalesces concurrent single retrieves into one request", async () => {
      const created = [await createVirtual(), await createVirtual()];
      const local = createTestClient();
      await local.connect();
      const send = spyOnSend(local);
      const res = await Promise.all(
        created.map(async ({ key }) => await local.channels.retrieve(key)),
      );
      expect(res.map(({ key }) => key)).toEqual(created.map(({ key }) => key));
      expect(
        send.mock.calls.filter(([target]) => target === "/channel/retrieve"),
      ).toHaveLength(1);
    });

    it("coalesces the status lookups of concurrent calculated retrieves", async () => {
      const source = await createVirtual();
      const created = await Promise.all([
        client.channels.create({
          name: `qry_${id.create()}`,
          dataType: DataType.FLOAT32,
          virtual: true,
          expression: `return ${source.name} * 2`,
        }),
        client.channels.create({
          name: `qry_${id.create()}`,
          dataType: DataType.FLOAT32,
          virtual: true,
          expression: `return ${source.name} + 1`,
        }),
      ]);
      // One status request needs both statuses to exist: the calculation
      // engine only writes them once data flows, so create them directly.
      await Promise.all(
        created.map(
          async ({ key }) =>
            await client.statuses.set({
              owner: channel.resourceID(key),
              name: "Calculation Status",
              variant: "info",
              message: "calculating",
              time: TimeStamp.now(),
            }),
        ),
      );
      const local = createTestClient();
      await local.connect();
      const send = spyOnSend(local);
      const res = await Promise.all(
        created.map(async ({ key }) => await local.channels.retrieve(key)),
      );
      expect(res.map(({ key }) => key)).toEqual(created.map(({ key }) => key));
      const targets = send.mock.calls.map(([target]) => target);
      expect(targets.filter((t) => t === "/channel/retrieve")).toHaveLength(1);
      expect(targets.filter((t) => t === "/status/retrieve")).toHaveLength(1);
    });

    it("does not reject concurrent retrieves when a key in the window is missing", async () => {
      const ch = await createVirtual();
      const local = createTestClient();
      await local.connect();
      const [ok, missing] = await Promise.allSettled([
        local.channels.retrieve(ch.key),
        local.channels.retrieve(999999999),
      ]);
      expect(ok.status).toEqual("fulfilled");
      expect(missing.status).toEqual("rejected");
      if (missing.status === "rejected")
        expect(NotFoundError.matches(missing.reason)).toBe(true);
    });
  });

  describe("records written by create", () => {
    it("serves a key query straight from the record", async () => {
      const ch = await createVirtual();
      const cached = expectLive(client.channels.watch(ch.key).getCached());
      expect(cached.name).toEqual(ch.name);
    });

    it("answers a keys query without a fetch", async () => {
      const ch = await createVirtual();
      const send = spyOnSend(client);
      try {
        const res = await client.channels.retrieve({ key: [ch.key] });
        expect(res.map(({ key }) => key)).toEqual([ch.key]);
        expect(send).not.toHaveBeenCalled();
      } finally {
        send.mockRestore();
      }
    });
  });

  describe("name resolution", () => {
    it("serves literal names from the record store without reaching the cluster", async () => {
      const { proxy, client: local } = await createProxiedTestClient({
        retry: FAST_RETRY,
        onRetry: quiet,
      });
      const ch = await createVirtual(local);
      await proxy.sever();
      const res = await local.channels.retrieve([ch.name]);
      expect(res.map((c) => c.key)).toEqual([ch.key]);
    }, 30000);

    it("serves a single literal name from the record store", async () => {
      const { proxy, client: local } = await createProxiedTestClient({
        retry: FAST_RETRY,
        onRetry: quiet,
      });
      const ch = await createVirtual(local);
      await proxy.sever();
      expect((await local.channels.retrieve(ch.name)).key).toEqual(ch.key);
    }, 30000);

    it("fetches only the names the store cannot resolve", async () => {
      // Created before the local client exists, so neither its cache nor its change
      // stream ever carries it. Severing proves the fetch went out for this name and
      // not for the known one.
      const unknown = await createVirtual(remote);
      const { proxy, client: local } = await createProxiedTestClient({
        retry: FAST_RETRY,
        onRetry: quiet,
      });
      const known = await createVirtual(local);
      await proxy.sever();
      await expect(
        local.channels.retrieve([known.name, unknown.name]),
      ).rejects.toThrow();
      await proxy.restore();
      await expect
        .poll(async () =>
          (await local.channels.retrieve([known.name, unknown.name]))
            .map((c) => c.key)
            .sort(),
        )
        .toEqual([known.key, unknown.key].sort());
    }, 30000);

    it("goes to the cluster when a name matches two stored channels", async () => {
      const { proxy, client: local } = await createProxiedTestClient({
        retry: FAST_RETRY,
        onRetry: quiet,
      });
      const ch = await createVirtual(local);
      const other = await createVirtual(local);
      // A cluster running without name validation can hold duplicates. Forge
      // the collision: an ambiguous name must never resolve from the store.
      local.channels.store.set([
        new channel.Channel({ ...other.payload, name: ch.name }),
      ]);
      await proxy.sever();
      await expect(local.channels.retrieve([ch.name])).rejects.toThrow();
      // The guard falls through to the cluster rather than failing outright,
      // so the same request answers once the link returns.
      await proxy.restore();
      await expect
        .poll(async () => (await local.channels.retrieve([ch.name])).map((c) => c.key))
        .toEqual([ch.key]);
    }, 30000);

    it("goes to the cluster for a request narrowed beyond names", async () => {
      const { proxy, client: local } = await createProxiedTestClient({
        retry: FAST_RETRY,
        onRetry: quiet,
      });
      const ch = await createVirtual(local);
      await proxy.sever();
      // The store cannot prove it holds every virtual channel named this, so
      // any field narrowing the request beyond names disqualifies it.
      await expect(
        local.channels.retrieve({ name: [ch.name], virtual: true }),
      ).rejects.toThrow();
    }, 30000);

    it("resolves cached names without scanning the table", async () => {
      const ch = await createVirtual();
      const spy = vi.spyOn(client.channels.store, "get");
      const res = await client.channels.retrieve([ch.name]);
      expect(res).toHaveLength(1);
      expect(res[0].key).toEqual(ch.key);
      for (const [arg] of spy.mock.calls) expect(typeof arg).not.toBe("function");
      spy.mockRestore();
    });

    it("stops resolving a stale name after a rename", async () => {
      const ch = await createVirtual();
      const renamed = `qry_${id.create()}`;
      await client.channels.rename(ch.key, renamed);
      expect(await client.channels.retrieve([ch.name])).toHaveLength(0);
      const res = await client.channels.retrieve([renamed]);
      expect(res).toHaveLength(1);
      expect(res[0].key).toEqual(ch.key);
    });

    it("evicts a deleted name without scanning the table", async () => {
      const ch = await createVirtual();
      const spy = vi.spyOn(client.channels.store, "get");
      await client.channels.delete([ch.name]);
      for (const [arg] of spy.mock.calls) expect(typeof arg).not.toBe("function");
      spy.mockRestore();
      await expect(client.channels.retrieve(ch.key)).rejects.toThrow(NotFoundError);
    });
  });

  describe("onChange", () => {
    it("delivers a remote rename to a subscribed single query", async () => {
      const ch = await createVirtual();
      const handler = vi.fn();
      const off = client.channels.watch(ch.key).onChange(handler);
      try {
        await client.channels.retrieve(ch.key);
        const renamed = `qry_renamed_${id.create()}`;
        await remote.channels.rename(ch.key, renamed);
        await expect
          .poll(() => {
            const cached = client.channels.watch(ch.key).getCached();
            return query.isLive(cached) && cached.name === renamed;
          })
          .toBe(true);
        expect(handler).toHaveBeenCalled();
      } finally {
        off();
      }
    });

    it("delivers a remote delete as a deleted result carrying the corpse", async () => {
      const ch = await createVirtual();
      const results: Array<query.Cached<channel.Channel> | undefined> = [];
      const off = client.channels.watch(ch.key).onChange((c) => results.push(c));
      try {
        await client.channels.retrieve(ch.key);
        await remote.channels.delete(ch.key);
        await expect
          .poll(() => query.Deleted.matches(client.channels.watch(ch.key).getCached()))
          .toBe(true);
        const last = expectDeleted(results.at(-1));
        expect(last.corpse.name).toEqual(ch.name);
        await expect(client.channels.retrieve(ch.key)).rejects.toThrow("deleted");
      } finally {
        off();
      }
    });

    it("should admit a renamed channel to a name query locally", async () => {
      const ch = await createVirtual();
      const name = `qry_wanted_${id.create()}`;
      const list = client.channels.watch({ where: { name: { in: [name] } } });
      const off = list.onChange(vi.fn());
      const send = spyOnSend(client);
      try {
        expect((await list.retrieve()).records).toEqual([]);
        send.mockClear();
        await client.channels.rename(ch.key, name);
        expect(list.getCached()?.records.map((c) => c.key)).toEqual([ch.key]);
        await sleep.sleep(TimeSpan.milliseconds(250));
        expect(send.mock.calls.map(([target]) => target)).toEqual([
          "/channel/dispatch",
        ]);
      } finally {
        send.mockRestore();
        off();
      }
    });
  });

  describe("calculated channel cached without a status", () => {
    // Outlasts the query layer's refetch debounce.
    const REFETCH_WINDOW = 150;
    const countOf = (send: ReturnType<typeof spyOnSend>, target: string) =>
      send.mock.calls.filter(([t]) => t === target).length;

    const createCalculated = async () => {
      const source = await createVirtual();
      const calc = await client.channels.create({
        name: `qry_${id.create()}`,
        dataType: DataType.FLOAT32,
        virtual: true,
        expression: `return ${source.name} * 2`,
      });
      // The Core clears a new calculated channel's status: let that settle.
      await sleep.sleep(TimeSpan.milliseconds(500));
      return calc;
    };

    const setStatus = async (calc: channel.Channel, message: string) =>
      await client.statuses.set({
        owner: channel.resourceID(calc.key),
        name: calc.name,
        variant: "error",
        message,
        time: TimeStamp.now(),
      });

    const createCalculatedWithStatus = async () => {
      const calc = await createCalculated();
      await setStatus(calc, "invalid expression");
      return calc;
    };

    const createCalculatedWithoutStatus = async () => {
      const calc = await createCalculated();
      const held = await client.statuses.retrieve({
        owners: [channel.resourceID(calc.key)],
      });
      await client.statuses.delete(held.map((s) => s.key));
      return calc;
    };

    it("should fetch the status after a status-free fetch", async () => {
      const calc = await createCalculatedWithStatus();
      const local = createTestClient();
      await local.channels.retrieve([calc.key]);
      expect(local.channels.watch(calc.key).getCached()).toBeUndefined();
      const retrieved = await local.channels.retrieve(calc.key);
      expect(retrieved.status?.message).toBe("invalid expression");
      expect(
        expectLive(local.channels.watch(calc.key).getCached()).status?.message,
      ).toBe("invalid expression");
    });

    it("should fetch the status after a channel arrives on the stream", async () => {
      const calc = await createCalculatedWithStatus();
      const local = createTestClient();
      await local.connect();
      await waitForStreamLive(local.connection);
      const seen: (channel.Status | undefined)[] = [];
      const stop = local.channels.watch(calc.key).onChange((cached) => {
        if (query.isLive(cached)) seen.push(cached.status);
      });
      try {
        await client.channels.rename(calc.key, `qry_${id.create()}`);
        await expect.poll(() => seen.at(-1)?.message).toBe("invalid expression");
        expect(seen.every((st) => st != null)).toBe(true);
      } finally {
        stop();
      }
    });

    it("should fetch the status for a subscriber with no fetch", async () => {
      const calc = await createCalculatedWithStatus();
      const local = createTestClient();
      await local.channels.retrieve([calc.key]);
      const seen: (channel.Status | undefined)[] = [];
      const stop = local.channels.watch(calc.key).onChange((cached) => {
        if (query.isLive(cached)) seen.push(cached.status);
      });
      try {
        await expect.poll(() => seen.at(-1)?.message).toBe("invalid expression");
        expect(seen.every((st) => st != null)).toBe(true);
      } finally {
        stop();
      }
    });

    it("should hold the status through the grace window", async () => {
      const calc = await createCalculatedWithStatus();
      const local = createTestClient();
      await local.channels.retrieve([calc.key]);
      const stop = local.channels.watch(calc.key).onChange(() => {});
      stop();
      const retrieved = await local.channels.retrieve(calc.key);
      expect(retrieved.status?.message).toBe("invalid expression");
    });

    it("should fetch the status for a query under a range", async () => {
      const calc = await createCalculatedWithStatus();
      const rng = await createRange();
      const local = createTestClient();
      await local.channels.retrieve([calc.key]);
      const params = { key: calc.key, rangeKey: rng.key };
      expect(local.channels.watch(params).getCached()).toBeUndefined();
      const retrieved = await local.channels.retrieve(params);
      expect(retrieved.status?.message).toBe("invalid expression");
    });

    it("should cost one fetch for a query with no status row", async () => {
      const calc = await createCalculatedWithoutStatus();
      const local = createTestClient();
      await local.channels.retrieve([calc.key]);
      await waitForStreamLive(local.connection);
      const send = spyOnSend(local);
      const stop = local.channels.watch(calc.key).onChange(() => {});
      try {
        const retrieved = await local.channels.retrieve(calc.key);
        expect(retrieved.status).toBeUndefined();
        await local.channels.retrieve(calc.key);
        const cached = expectLive(local.channels.watch(calc.key).getCached());
        expect(cached.key).toBe(calc.key);
        expect(cached.status).toBeUndefined();
        await sleep.sleep(TimeSpan.milliseconds(REFETCH_WINDOW));
        expect(countOf(send, "/status/retrieve")).toBe(1);
        expect(countOf(send, "/channel/retrieve")).toBe(0);
      } finally {
        stop();
      }
    });

    it("should cost one fetch for a list", async () => {
      const calcs = await Promise.all([
        createCalculatedWithoutStatus(),
        createCalculatedWithoutStatus(),
        createCalculatedWithoutStatus(),
      ]);
      const keys = calcs.map(({ key }) => key);
      const local = createTestClient();
      await local.channels.retrieve(keys);
      await waitForStreamLive(local.connection);
      const send = spyOnSend(local);
      const stops = keys.map((key) => local.channels.watch(key).onChange(() => {}));
      try {
        await Promise.all(keys.map(async (key) => await local.channels.retrieve(key)));
        await sleep.sleep(TimeSpan.milliseconds(REFETCH_WINDOW));
        expect(countOf(send, "/status/retrieve")).toBe(1);
      } finally {
        stops.forEach((stop) => stop());
      }
    });

    it("should deliver a status written after a fetch found none", async () => {
      const calc = await createCalculatedWithoutStatus();
      const local = createTestClient();
      await local.channels.retrieve([calc.key]);
      const seen: (channel.Status | undefined)[] = [];
      const stop = local.channels.watch(calc.key).onChange((cached) => {
        if (query.isLive(cached)) seen.push(cached.status);
      });
      await waitForStreamLive(local.connection);
      try {
        await local.channels.retrieve(calc.key);
        await setStatus(calc, "broken expression");
        await expect.poll(() => seen.at(-1)?.message).toBe("broken expression");
      } finally {
        stop();
      }
    });

    it("should never fetch for a list of plain channels", async () => {
      const keys = (
        await Promise.all([createVirtual(), createVirtual(), createVirtual()])
      ).map(({ key }) => key);
      const local = createTestClient();
      await local.channels.retrieve(keys);
      await waitForStreamLive(local.connection);
      const send = spyOnSend(local);
      const stops = keys.map((key) => local.channels.watch(key).onChange(() => {}));
      try {
        for (const key of keys) {
          expect(expectLive(local.channels.watch(key).getCached()).key).toBe(key);
          await local.channels.retrieve(key);
        }
        await sleep.sleep(TimeSpan.milliseconds(REFETCH_WINDOW));
        expect(countOf(send, "/status/retrieve")).toBe(0);
        expect(countOf(send, "/channel/retrieve")).toBe(0);
      } finally {
        stops.forEach((stop) => stop());
      }
    });
  });

  describe("statuses", () => {
    const createCalculated = async (): Promise<channel.Channel> => {
      const source = await createVirtual();
      const ch = await client.channels.create({
        name: `qry_${id.create()}`,
        dataType: DataType.FLOAT32,
        virtual: true,
        expression: `return ${source.name} * 2`,
      });
      await setStatus(ch.key, "calculating");
      return ch;
    };

    const setStatus = async (key: number, message: string): Promise<void> => {
      await remote.statuses.set({
        owner: channel.resourceID(key),
        name: "Calculation Status",
        variant: "info",
        message,
        time: TimeStamp.now(),
      });
    };

    it("composes statuses only for a request that includes them", async () => {
      const ch = await createCalculated();
      const local = createTestClient();
      await local.connect();
      const where = { key: { in: [ch.key] } };
      const [bare] = await local.channels.retrieve({ where });
      expect(bare.status).toBeUndefined();
      const [withStatus] = await local.channels.retrieve({
        where,
        includeStatus: true,
      });
      expect(withStatus.status?.message).toEqual("calculating");
    });

    it("delivers a remote status change to a request that includes them", async () => {
      const ch = await createCalculated();
      const list = client.channels.watch({
        where: { key: { in: [ch.key] } },
        includeStatus: true,
      });
      const results: Array<channel.Channel[] | undefined> = [];
      const off = list.onChange((w) => results.push(w?.records));
      try {
        await list.retrieve();
        await setStatus(ch.key, "done");
        await expect.poll(() => results.at(-1)?.[0]?.status?.message).toEqual("done");
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

    it.each([
      {
        name: "single",
        retrieve: async (key: channel.Key, rangeKey: ranger.Key) =>
          await client.channels.retrieve(key, { rangeKey }),
      },
      {
        name: "where",
        retrieve: async (key: channel.Key, rangeKey: ranger.Key) => {
          const res = await client.channels.retrieve({ key: [key], rangeKey });
          return res[0];
        },
      },
    ])(
      "falls back to the parent alias on a $name query once the child's is deleted",
      async ({ retrieve }) => {
        const ch = await createVirtual();
        const parent = await createRange();
        const start = TimeStamp.now();
        const child = await client.ranges.create({
          name: `qry-${id.create()}`,
          timeRange: new TimeRange(start, start.add(TimeStamp.seconds(1))),
          parent: parent.key,
        });
        const inherited = `qry_alias_${id.create()}`;
        await client.ranges.setAlias(parent.key, ch.key, inherited);
        await client.ranges.setAlias(child.key, ch.key, `qry_alias_${id.create()}`);
        await client.ranges.deleteAlias(child.key, ch.key);
        // The delete's own echo can drop one read, and the next read restores it.
        await expect
          .poll(async () => (await retrieve(ch.key, child.key)).alias)
          .toEqual(inherited);
      },
    );

    it("delivers remote alias changes to a subscribed single query", async () => {
      const ch = await createVirtual();
      const rng = await createRange();
      const params = { key: ch.key, rangeKey: rng.key };
      const off = client.channels.watch(params).onChange(() => {});
      try {
        await client.channels.retrieve(ch.key, { rangeKey: rng.key });
        const alias = `qry_alias_${id.create()}`;
        await remote.ranges.setAlias(rng.key, ch.key, alias);
        await expect
          .poll(() => {
            const cached = client.channels.watch(params).getCached();
            return query.isLive(cached) ? cached.alias : undefined;
          })
          .toBe(alias);
        await remote.ranges.deleteAlias(rng.key, ch.key);
        await expect
          .poll(() => {
            const cached = client.channels.watch(params).getCached();
            return query.isLive(cached) ? cached.alias : undefined;
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
        key: [a.key, b.key],
        rangeKey: rng.key,
      });
      expect(res.find((c) => c.key === a.key)?.alias).toEqual(alias);
      expect(res.find((c) => c.key === b.key)?.alias).toBeUndefined();
    });

    it("puts the channels whose alias matches a search first", async () => {
      const term = id.create();
      const named = await client.channels.create({
        name: `${term}_named`,
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const aliased = await createVirtual();
      const excluded = await client.channels.create({
        name: `qry_${id.create()}`,
        dataType: DataType.FLOAT64,
        virtual: true,
      });
      const rng = await createRange();
      await client.ranges.setAlias(rng.key, aliased.key, `${term}_alias`);
      await client.ranges.setAlias(rng.key, excluded.key, `${term}_excluded`);
      await expect
        .poll(async () => {
          const res = await client.channels.retrieve({
            searchTerm: term,
            rangeKey: rng.key,
            dataType: DataType.FLOAT32,
          });
          return res.map((ch) => [ch.key, ch.alias]);
        })
        .toEqual([
          [aliased.key, `${term}_alias`],
          [named.key, undefined],
        ]);
    });

    it("does not serve an unfetched keys request under a range", async () => {
      const ch = await createVirtual();
      const rng = await createRange();
      await client.channels.retrieve(ch.key);
      expect(
        client.channels.watch({ key: [ch.key], rangeKey: rng.key }).getCached(),
      ).toBeUndefined();
    });

    describe("single query on a row cached without a range", () => {
      const createAliased = async () => {
        const ch = await createVirtual();
        const rng = await createRange();
        const alias = `qry_alias_${id.create()}`;
        await client.ranges.setAlias(rng.key, ch.key, alias);
        const local = createTestClient();
        await local.channels.retrieve(ch.key);
        return { local, params: { key: ch.key, rangeKey: rng.key }, alias };
      };

      it("should not answer getCached from the row", async () => {
        const { local, params, alias } = await createAliased();
        expect(local.channels.watch(params).getCached()).toBeUndefined();
        await local.channels.retrieve(params);
        expect(expectLive(local.channels.watch(params).getCached()).alias).toBe(alias);
      });

      it("should fetch the alias for a subscribed query", async () => {
        const { local, params, alias } = await createAliased();
        const stop = local.channels.watch(params).onChange(() => {});
        try {
          const res = await local.channels.retrieve(params);
          expect(res.alias).toBe(alias);
        } finally {
          stop();
        }
      });

      it("should fetch once for a channel with no alias", async () => {
        const ch = await createVirtual();
        const rng = await createRange();
        const local = createTestClient();
        await local.channels.retrieve(ch.key);
        await waitForStreamLive(local.connection);
        const send = spyOnSend(local);
        const params = { key: ch.key, rangeKey: rng.key };
        const stop = local.channels.watch(params).onChange(() => {});
        try {
          expect((await local.channels.retrieve(params)).alias).toBeUndefined();
          await local.channels.retrieve(params);
          expect(
            expectLive(local.channels.watch(params).getCached()).alias,
          ).toBeUndefined();
          expect(
            send.mock.calls.filter(([target]) => target === "/range/alias/retrieve"),
          ).toHaveLength(1);
        } finally {
          stop();
        }
      });
    });
  });
});
