// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { control, DataType, id, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, test, vi } from "vitest";

import { channel } from "@/channel";
import { NotFoundError, PathError, ValidationError } from "@/index";
import { query } from "@/query";
import {
  createProxiedTestClient,
  createTestClient,
  expectDeleted,
  expectLive,
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
        new PathError(["where", "name"], new ValidationError("no operator set")),
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
    test("retrieve no channels for an empty array", async () => {
      await client.channels.create({
        name: id.create(),
        virtual: true,
        dataType: DataType.FLOAT32,
      });
      expect(await client.channels.retrieve([])).toEqual([]);
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

  describe("countLimited", () => {
    it("should return the count from the Core", async () => {
      // Other spec files create and delete channels on the same Core at once, so the
      // exact count is pinned in the Core's specs.
      await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      expect(await client.channels.countLimited()).toBeGreaterThanOrEqual(1);
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
          `cannot delete channel ${channel.name} because task ${task.name} uses it`,
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
            "it as its index",
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

    it("should keep the cached channel when the Core rejects the actions", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        dataType: DataType.INT64,
        virtual: true,
      });
      const before = client.channels.store.get(ch.key);
      await expect(
        client.channels.dispatch(ch.key, channel.rename({ name: "bad name" })),
      ).rejects.toThrow("contains invalid characters");
      expect(client.channels.store.get(ch.key)).toBe(before);
    });
  });
  describe("update calculations", () => {
    test("update calculated channel name", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
        expression: "return 1",
      });
      const name = id.create();
      await client.channels.rename(ch.key, name);
      expect((await client.channels.retrieve(ch.key)).name).toEqual(name);
      const [calc] = await client.calculations.retrieve({ output: ch.key });
      expect(calc.expression).toEqual("return 1");
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
      await waitForStreamLive(local.connection);
      const send = spyOnSend(local);
      const res = await Promise.all(
        created.map(async ({ key }) => await local.channels.retrieve(key)),
      );
      expect(res.map(({ key }) => key)).toEqual(created.map(({ key }) => key));
      expect(
        send.mock.calls.filter(([target]) => target === "/channel/retrieve"),
      ).toHaveLength(1);
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
});

describe("old API edits of a calculated channel", () => {
  it("should reach a client that holds the channel in its cache", async () => {
    const local = createTestClient();
    // The local client must stream before the edit, or it misses the edit's broadcast.
    await local.connect();
    await waitForStreamLive(local.connection);
    const source = await local.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const ch = await local.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
      expression: `return ${source.name} * 2`,
    });
    await local.channels.retrieve(ch.key);
    const operations: channel.Operation[] = [
      { type: "avg", resetChannel: 0, duration: TimeSpan.seconds(5) },
    ];
    await remote.channels.dispatch(
      ch.key,
      channel.set({ path: ["operations"], value: operations }),
    );
    await expect
      .poll(async () => (await local.channels.retrieve(ch.key)).operations)
      .toEqual(operations);
  });
});
