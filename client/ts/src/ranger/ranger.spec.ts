// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  color,
  DataType,
  id,
  math,
  sleep,
  TimeRange,
  TimeSpan,
  TimeStamp,
  uuid,
} from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, onTestFinished, vi } from "vitest";

import { NotFoundError, type Synnax, ValidationError } from "@/index";
import { query } from "@/query";
import { ranger } from "@/ranger";
import {
  createTestClient,
  expectDeleted,
  expectLive,
  holdEachResponse,
  holdResponses,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";

const client = createTestClient();

describe("range", () => {
  describe("payload", () => {
    it("should validate the time range", () => {
      const payload = ranger.payloadZ.parse({
        name: "My New One Second Range",
        key: uuid.create(),
        timeRange: { start: 0, end: 1 },
      });
      expect(payload).toBeDefined();
      expect(payload.timeRange.start.valueOf()).toBe(0n);
      expect(payload.timeRange.end.valueOf()).toBe(1n);
    });
    it("should not validate the time range if it is invalid", () => {
      const input = {
        name: "My New One Second Range",
        key: uuid.create(),
        timeRange: { start: 1, end: 0 },
      };
      expect(() => ranger.payloadZ.parse(input)).toThrow(
        "Time range start time must be before or equal to time range end time",
      );
    });
    it("should validate the time range if the end is less than or equal to the maximum value of an int64", () => {
      const input = {
        name: "range with end greater than max int64",
        key: uuid.create(),
        timeRange: { start: 1, end: math.MAX_INT64 },
      };
      const payload = ranger.payloadZ.parse(input);
      expect(payload).toBeDefined();
      expect(payload.timeRange.end.valueOf()).toBe(math.MAX_INT64);
    });
    it("should not validate the time range if the end is greater than the maximum value of an int64", () => {
      const input = {
        name: "range with end greater than max int64",
        key: uuid.create(),
        timeRange: { start: 1, end: math.MAX_INT64 + 1n },
      };
      expect(() => ranger.payloadZ.parse(input)).toThrow(
        "Time range end time must be less than or equal to the maximum value of an int64",
      );
    });
    it("should validate the time range if start is greater than the minimum value of an int64", () => {
      const input = {
        name: "range with start greater than min int64",
        key: uuid.create(),
        timeRange: { start: math.MIN_INT64, end: 0 },
      };
      const payload = ranger.payloadZ.parse(input);
      expect(payload).toBeDefined();
      expect(payload.timeRange.start.valueOf()).toBe(math.MIN_INT64);
      expect(payload.timeRange.end.valueOf()).toBe(0n);
    });
    it("should not validate the time range if start is less than the minimum value of an int64", () => {
      const input = {
        name: "range with start less than min int64",
        key: uuid.create(),
        timeRange: { start: -1n * 2n ** 63n - 1n, end: 0 },
      };
      expect(() => ranger.payloadZ.parse(input)).toThrow(
        "Time range start time must be greater than or equal to the minimum value of an int64",
      );
    });
  });
  describe("create", () => {
    it("should create a single range", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.seconds(1));
      const range = await client.ranges.create({
        name: "My New One Second Range",
        timeRange,
        color: "#E774D0",
      });
      expect(range.key).not.toHaveLength(0);
      expect(timeRange).toEqual(range.timeRange);
      expect(range.color).toEqual([231, 116, 208, 1]);
    });
    it("should create multiple ranges", async () => {
      const ranges: ranger.New[] = [
        {
          name: "My New One Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        },
        {
          name: "My New Two Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
        },
      ];
      const createdRanges = await client.ranges.create(ranges);
      expect(createdRanges).toHaveLength(2);
      expect(createdRanges[0].key).not.toHaveLength(0);
      expect(createdRanges[1].key).not.toHaveLength(0);
      expect(createdRanges[0].timeRange).toEqual(ranges[0].timeRange);
      expect(createdRanges[1].timeRange).toEqual(ranges[1].timeRange);
    });
    it("should create a range with a parent", async () => {
      const parentRange = await client.ranges.create({
        name: "My New Parent Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const child = await client.ranges.create({
        name: "My New Child Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        parent: parentRange.key,
      });
      expect(child.parent).toEqual(parentRange.key);
      const children = await client.ranges.retrieve({ parents: [parentRange.key] });
      expect(children.map(({ key }) => key)).toEqual([child.key]);
      const retrieved = await client.ranges.retrieve(parentRange.key);
      expect((await retrieved.retrieveChildren()).map(({ key }) => key)).toEqual([
        child.key,
      ]);
    });
    it("should create a range with labels", async () => {
      const lbl = await client.labels.create({ name: "Created", color: "#E774D0" });
      const rng = await client.ranges.create({
        name: "Labeled",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        labels: [lbl.key],
      });
      expect(rng.labels).toEqual([lbl.key]);
    });
    it("should refuse a range whose key exists and keep its labels", async () => {
      const [held, carried] = await client.labels.create([
        { name: "Held", color: "#E774D0" },
        { name: "Carried", color: "#E774D0" },
      ]);
      const rng = await client.ranges.create({
        name: "Labeled",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        labels: [held.key],
      });
      const create = client.ranges.create({ ...rng.payload, labels: [carried.key] });
      await expect(create).rejects.toThrow(ValidationError);
      await expect(create).rejects.toThrow(`range ${rng.key} already exists`);
      const retrieved = await createTestClient().ranges.retrieve(rng.key);
      expect(retrieved.labels).toEqual([held.key]);
    });
    it("should keep a label change that arrives before the create reply", async () => {
      const lbl = await client.labels.create({ name: "Late", color: "#E774D0" });
      const key = uuid.create();
      const local = createTestClient();
      await local.connect();
      await waitForStreamLive(local.connection);
      const { responded, release } = holdResponses(local, "/range/create");
      const creating = local.ranges.create({
        key,
        name: "Labeled late",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await responded;
      await client.labels.label(ranger.resourceID(key), [lbl.key]);
      await expect
        .poll(() => expectLive(local.ranges.watch(key).getCached()).labels)
        .toContain(lbl.key);
      release();
      await creating;
      expect(expectLive(local.ranges.watch(key).getCached()).labels).toContain(lbl.key);
    });
  });

  describe("delete", () => {
    it("should delete a single range", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.seconds(1));
      const range = await client.ranges.create({
        name: "My New One Second Range",
        timeRange,
      });
      await client.ranges.delete(range.key);
      await expect(async () => await client.ranges.retrieve(range.key)).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("rename", () => {
    it("should rename a single range", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.seconds(1));
      const range = await client.ranges.create({
        name: "My New One Second Range",
        timeRange,
      });
      await client.ranges.rename(range.key, "My New One Second Range Renamed");
      const renamed = await client.ranges.retrieve(range.key);
      expect(renamed.name).toEqual("My New One Second Range Renamed");
    });
  });

  describe("retrieve", () => {
    it("should retrieve a range by key", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.seconds(1));
      const range = await client.ranges.create({
        name: "My New One Second Range",
        timeRange,
      });
      const retrieved = await client.ranges.retrieve(range.key);
      expect(retrieved.key).toEqual(range.key);
      expect(retrieved.timeRange).toEqual(range.timeRange);
    });
    it("should retrieve a range by a key object", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.seconds(1));
      const range = await client.ranges.create({ name: "Key Object Range", timeRange });
      const retrieved = await client.ranges.retrieve({ key: range.key });
      expect(retrieved.key).toEqual(range.key);
    });
    it("should retrieve ranges by a flat filter", async () => {
      const prefix = `flat-${id.create()}`;
      const timeRange = TimeStamp.now().spanRange(TimeSpan.seconds(1));
      const [labeled] = await client.ranges.create([
        { name: `${prefix}-labeled`, timeRange },
        { name: `${prefix}-plain`, timeRange },
      ]);
      const lbl = await client.labels.create({ name: prefix, color: "#E774D0" });
      await labeled.addLabel(lbl.key);
      const retrieved = await client.ranges.retrieve({
        name: { startsWith: prefix },
        labels: lbl.key,
      });
      expect(retrieved.map((r) => r.key)).toEqual([labeled.key]);
    });
    it("should retrieve a range by name", async () => {
      const timeRange = TimeStamp.now().spanRange(TimeSpan.seconds(1));
      const range = await client.ranges.create({
        name: "My New Three Second Range",
        timeRange,
      });
      const retrieved = await client.ranges.retrieve([range.name]);
      expect(retrieved.length).toBeGreaterThan(0);
      expect(retrieved[0].name).toEqual(range.name);
    });
    it("should retrieve ranges that overlap with the given time range", async () => {
      const timeRange = TimeStamp.hours(500).spanRange(TimeSpan.seconds(1));
      const range = await client.ranges.create({
        name: "My New One Second Range",
        timeRange,
      });
      const retrieved = await client.ranges.retrieve(timeRange);
      expect(retrieved.length).toBeGreaterThan(0);
      expect(retrieved.map((r) => r.key)).toContain(range.key);
    });
    it("should page ranges by end time", async () => {
      const parent = await client.ranges.create({
        name: "End Order Parent",
        timeRange: TimeStamp.seconds(1).spanRange(TimeSpan.seconds(100)),
      });
      const child = async (start: number, end: number) =>
        await client.ranges.create({
          name: `${start}-${end}`,
          timeRange: new TimeRange(TimeStamp.seconds(start), TimeStamp.seconds(end)),
          parent: parent.key,
        });
      const late = await child(1, 50);
      const early = await child(20, 25);
      const middle = await child(30, 35);
      const order: ranger.Order = { field: "time_range_end", descending: false };
      const first = await client.ranges.retrieve({
        where: { parents: [parent.key] },
        order,
        limit: 2,
      });
      expect(first.map((r) => r.key)).toEqual([early.key, middle.key]);
      const last = first[1];
      const rest = await client.ranges.retrieve({
        where: { parents: [parent.key] },
        order,
        after: { key: last.key, timeRangeEnd: last.timeRange.end },
      });
      expect(rest.map((r) => r.key)).toEqual([late.key]);
    });

    it("should retrieve ranges by search term", async () => {
      const prefix = `searchable-range-${id.create()}`;
      const timeRange = TimeStamp.now().spanRange(TimeSpan.seconds(1));
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.ranges.create(names.map((name) => ({ name, timeRange })));
      await expect
        .poll(async () => {
          const results = await client.ranges.retrieve({ searchTerm: prefix });
          return results.map((r) => r.name).sort();
        })
        .toEqual(names);
    });
  });

  describe("retrieveParent", () => {
    it("should retrieve the parent of a range", async () => {
      const parentRange = await client.ranges.create({
        name: "My New Parent Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const childRange = await client.ranges.create({
        name: "My New Child Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        parent: parentRange.key,
      });
      const parent = await childRange.retrieveParent();
      expect(parent?.key).toEqual(parentRange.key);
    });
  });

  describe("KV", () => {
    it("should set, get, and delete a single key", async () => {
      const rng = await client.ranges.create({
        name: "My New One Second Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await rng.kv.set("foo", "bar");
      const val = await rng.kv.get("foo");
      expect(val).toEqual("bar");
      await rng.kv.delete("foo");
      await expect(async () => await rng.kv.get("foo")).rejects.toThrow(NotFoundError);
    });

    it("should set and get multiple keys", async () => {
      const rng = await client.ranges.create({
        name: "My New One Second Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await rng.kv.set({ foo: "bar", baz: "qux" });
      const res = await rng.kv.list();
      expect(res).toEqual({ foo: "bar", baz: "qux" });
    });

    it("should list all keys", async () => {
      const rng = await client.ranges.create({
        name: "My New One Second Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await rng.kv.set({ foo: "bar", baz: "qux" });
      const res = await rng.kv.list();
      expect(res).toEqual({ foo: "bar", baz: "qux" });
    });
  });

  describe("label", () => {
    it("should set and get a label for the range", async () => {
      const rng = await client.ranges.create({
        name: "My New One Second Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const label = await client.labels.create({
        name: "My New Label",
        color: "#E774D0",
      });
      await rng.addLabel(label.key);
      const newRange = await createTestClient().ranges.retrieve(rng.key);
      expect(newRange.labels).toEqual([label.key]);
      expect(await newRange.retrieveLabels()).toEqual([label]);
    });

    it("should retrieve labels added after the range was created", async () => {
      const rng = await client.ranges.create({
        name: "Labeled later",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const label = await client.labels.create({ name: "Later", color: "#E774D0" });
      await rng.addLabel(label.key);
      expect(await rng.retrieveLabels()).toEqual([label]);
    });

    it("should remove a label from the range", async () => {
      const label = await client.labels.create({ name: "Removed", color: "#E774D0" });
      const rng = await client.ranges.create({
        name: "My New One Second Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        labels: [label.key],
      });
      await rng.removeLabel(label.key);
      const newRange = await createTestClient().ranges.retrieve(rng.key);
      expect(newRange.labels).toEqual([]);
    });
  });

  describe("parent", () => {
    it("should set and get a parent for the range", async () => {
      const parent = await client.ranges.create({
        name: "My New One Second Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const child = await client.ranges.create({
        name: "My New One Second Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await client.ranges.dispatch(
        child.key,
        ranger.set({ path: ["parent"], value: parent.key }),
      );
      const stored = await createTestClient().ranges.retrieve(child.key);
      expect(stored.parent).toEqual(parent.key);
      expect((await stored.retrieveParent())?.key).toEqual(parent.key);
    });

    it("should retrieve a parent set after the range was created", async () => {
      const parent = await client.ranges.create({
        name: "Parent",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const child = await client.ranges.create({
        name: "Child",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await client.ranges.dispatch(
        child.key,
        ranger.set({ path: ["parent"], value: parent.key }),
      );
      expect((await child.retrieveParent())?.key).toEqual(parent.key);
    });

    it("should remove the parent of the range", async () => {
      const parent = await client.ranges.create({
        name: "Parent",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const child = await client.ranges.create({
        name: "Child",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        parent: parent.key,
      });
      await client.ranges.dispatch(
        child.key,
        ranger.remove({ path: [], key: "parent" }),
      );
      expect(
        expectLive(client.ranges.watch(child.key).getCached()).parent,
      ).toBeUndefined();
      const stored = await createTestClient().ranges.retrieve(child.key);
      expect(stored.parent).toBeUndefined();
    });
  });

  describe("hasChildren", () => {
    const timeRange = TimeStamp.now().spanRange(TimeSpan.seconds(1));

    it("should mark a range that has a nested range", async () => {
      const parent = await client.ranges.create({ name: "Parent", timeRange });
      const child = await client.ranges.create({
        name: "Child",
        timeRange,
        parent: parent.key,
      });
      const stored = await createTestClient().ranges.retrieve([parent.key, child.key]);
      expect(stored.map(({ name, hasChildren }) => [name, hasChildren])).toEqual(
        expect.arrayContaining([
          ["Parent", true],
          ["Child", false],
        ]),
      );
    });

    it("should mark a cached range when another client creates a range in it", async () => {
      const remote = createTestClient();
      const parent = await client.ranges.create({ name: "Parent", timeRange });
      await waitForStreamLive(client.connection);
      await remote.ranges.create({ name: "Created", timeRange, parent: parent.key });
      await expect
        .poll(() => expectLive(client.ranges.watch(parent.key).getCached()).hasChildren)
        .toBe(true);
    });

    it("should mark a cached range when another client moves a range into it", async () => {
      const remote = createTestClient();
      const parent = await client.ranges.create({ name: "Parent", timeRange });
      const moved = await client.ranges.create({ name: "Moved", timeRange });
      await remote.ranges.retrieve(moved.key);
      await waitForStreamLive(client.connection);
      await remote.ranges.dispatch(
        moved.key,
        ranger.set({ path: ["parent"], value: parent.key }),
      );
      await expect
        .poll(() => expectLive(client.ranges.watch(parent.key).getCached()).hasChildren)
        .toBe(true);
    });
  });

  describe("Alias", () => {
    describe("set + resolve", () => {
      it("should set and resolve an alias for the range", async () => {
        const ch = await client.channels.create({
          name: id.create(),
          dataType: DataType.FLOAT32,
          virtual: true,
        });
        const rng = await client.ranges.create({
          name: "My New One Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        });
        await rng.setAlias(ch.key, "myalias");
        const resolved = await rng.resolveAlias("myalias");
        expect(resolved).toEqual(ch.key);
      });

      it("should resolve an alias to the channel it moved to", async () => {
        const [first, second] = await client.channels.create([
          { name: id.create(), dataType: DataType.FLOAT32, virtual: true },
          { name: id.create(), dataType: DataType.FLOAT32, virtual: true },
        ]);
        const rng = await client.ranges.create({
          name: "My New One Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        });
        await rng.setAlias(first.key, "myalias");
        expect(await rng.resolveAlias("myalias")).toEqual(first.key);
        await rng.setAlias(first.key, "otheralias");
        await rng.setAlias(second.key, "myalias");
        expect(await rng.resolveAlias("myalias")).toEqual(second.key);
      });

      it("should reject an alias named after an object member", async () => {
        const rng = await client.ranges.create({
          name: "My New One Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        });
        const resolve = rng.resolveAlias("constructor");
        await expect(resolve).rejects.toThrow(NotFoundError);
        await expect(resolve).rejects.toThrow('alias not found matching "constructor"');
      });

      it("should reject a channel name that matches no channel", async () => {
        const rng = await client.ranges.create({
          name: "My New One Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        });
        const name = id.create();
        const set = rng.setAlias(name, "myalias");
        await expect(set).rejects.toThrow(NotFoundError);
        await expect(set).rejects.toThrow(`Found no channel where name = "${name}"`);
      });

      it("should read a name pattern as an exact channel name", async () => {
        const prefix = `alias_${id.create()}`;
        await client.channels.create([
          { name: `${prefix}_a`, dataType: DataType.FLOAT32, virtual: true },
          { name: `${prefix}_b`, dataType: DataType.FLOAT32, virtual: true },
        ]);
        const rng = await client.ranges.create({
          name: "My New One Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        });
        const name = `${prefix}_.*`;
        const set = rng.setAlias(name, "myalias");
        await expect(set).rejects.toThrow(NotFoundError);
        await expect(set).rejects.toThrow(`Found no channel where name = "${name}"`);
      });
    });
    describe("deleteAlias", () => {
      it("should remove an alias for the range", async () => {
        const ch = await client.channels.create({
          name: id.create(),
          dataType: DataType.FLOAT32,
          virtual: true,
        });
        const rng = await client.ranges.create({
          name: "My New One Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        });
        await rng.setAlias(ch.key, "myalias");
        await rng.deleteAlias(ch.key);
        const resolve = rng.resolveAlias("myalias");
        await expect(resolve).rejects.toThrow(NotFoundError);
        await expect(resolve).rejects.toThrow('alias not found matching "myalias"');
        const retrieve = client.ranges.retrieveAlias(rng.key, ch.key);
        await expect(retrieve).rejects.toThrow(NotFoundError);
        await expect(retrieve).rejects.toThrow(
          `alias not found matching channel ${ch.key}`,
        );
      });
    });
    describe("list", () => {
      it("should list the aliases for the range", async () => {
        const ch = await client.channels.create({
          name: id.create(),
          dataType: DataType.FLOAT32,
          virtual: true,
        });
        const rng = await client.ranges.create({
          name: "My New One Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        });
        await rng.setAlias(ch.key, "myalias");
        const aliases = await rng.listAliases();
        expect(aliases).toEqual({ [ch.key]: "myalias" });
      });
    });
  });

  describe("optimistic writes", () => {
    const WRITE_FAILED = new Error("write failed");
    const fail = () => {
      throw WRITE_FAILED;
    };

    const createRange = async () =>
      await client.ranges.create({
        name: `optimistic-${id.create()}`,
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });

    const createChannel = async () =>
      await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });

    describe("delete", () => {
      it("should drop the range from the cache before the write commits", async () => {
        const rng = await createRange();
        let duringWrite: query.Cached<ranger.Range> | undefined;
        await client.ranges.delete(rng.key, {
          onOptimistic: () => {
            duringWrite = client.ranges.watch(rng.key).getCached();
          },
        });
        expect(query.isLive(duringWrite)).toBe(false);
      });

      it("should restore the range when the write fails", async () => {
        const rng = await createRange();
        await expect(
          client.ranges.delete(rng.key, { onOptimistic: fail }),
        ).rejects.toBe(WRITE_FAILED);
        expect(expectLive(client.ranges.watch(rng.key).getCached()).name).toEqual(
          rng.name,
        );
      });

      it("should restore the range in its parent's children when the write fails", async () => {
        const parent = await createRange();
        const child = await client.ranges.create({
          name: `optimistic-child-${id.create()}`,
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
          parent: parent.key,
        });
        const list = client.ranges.watch({ parents: [parent.key] });
        const off = list.onChange(() => {});
        try {
          expect((await list.retrieve()).records).toHaveLength(1);
          await expect(
            client.ranges.delete(child.key, { onOptimistic: fail }),
          ).rejects.toBe(WRITE_FAILED);
          expect(list.getCached()?.records.map((c) => c.key)).toEqual([child.key]);
        } finally {
          off();
        }
      });
    });

    describe("setAlias", () => {
      it("should cache the alias before the write commits", async () => {
        const rng = await createRange();
        const ch = await createChannel();
        const key = ranger.alias.createKey({ range: rng.key, channel: ch.key });
        let duringWrite: ranger.alias.Alias | undefined;
        await client.ranges.setAlias(rng.key, ch.key, "optimistic-alias", {
          onOptimistic: () => {
            duringWrite = client.ranges.aliases.get(key);
          },
        });
        expect(duringWrite?.alias).toEqual("optimistic-alias");
      });

      it("should drop the alias when the write fails", async () => {
        const rng = await createRange();
        const ch = await createChannel();
        const key = ranger.alias.createKey({ range: rng.key, channel: ch.key });
        await expect(
          client.ranges.setAlias(rng.key, ch.key, "optimistic-alias", {
            onOptimistic: fail,
          }),
        ).rejects.toBe(WRITE_FAILED);
        expect(client.ranges.aliases.get(key)).toBeUndefined();
      });
    });

    describe("deleteAlias", () => {
      it("should drop the alias before the write commits", async () => {
        const rng = await createRange();
        const ch = await createChannel();
        const key = ranger.alias.createKey({ range: rng.key, channel: ch.key });
        await client.ranges.setAlias(rng.key, ch.key, "optimistic-alias");
        let duringWrite: ranger.alias.Alias | undefined;
        await client.ranges.deleteAlias(rng.key, ch.key, {
          onOptimistic: () => {
            duringWrite = client.ranges.aliases.get(key);
          },
        });
        expect(duringWrite).toBeUndefined();
      });

      it("should restore the alias when the write fails", async () => {
        const rng = await createRange();
        const ch = await createChannel();
        const key = ranger.alias.createKey({ range: rng.key, channel: ch.key });
        await client.ranges.setAlias(rng.key, ch.key, "optimistic-alias");
        await expect(
          client.ranges.deleteAlias(rng.key, ch.key, { onOptimistic: fail }),
        ).rejects.toBe(WRITE_FAILED);
        expect(client.ranges.aliases.get(key)?.alias).toEqual("optimistic-alias");
      });
    });

    describe("kv", () => {
      const cachedPairs = (key: ranger.Key): ranger.kv.Pair[] =>
        expectLive(client.ranges.kv.watch(key).getCached());

      // An unsubscribed query never patches its membership, so every spec here
      // holds a subscription for as long as it reads the cached pairs.
      const openRange = async () => {
        const rng = await createRange();
        const off = client.ranges.kv.watch(rng.key).onChange(vi.fn());
        await client.ranges.kv.retrieve(rng.key);
        return { rng, off };
      };

      it("should cache a pair before the write commits", async () => {
        const { rng, off } = await openRange();
        try {
          let duringWrite: ranger.kv.Pair[] = [];
          await rng.kv.set("foo", "bar", {
            onOptimistic: () => {
              duringWrite = cachedPairs(rng.key);
            },
          });
          expect(duringWrite).toEqual([{ range: rng.key, key: "foo", value: "bar" }]);
        } finally {
          off();
        }
      });

      it("should cache a keyed-object set before the write commits", async () => {
        const { rng, off } = await openRange();
        try {
          let duringWrite: ranger.kv.Pair[] = [];
          await rng.kv.set(
            { foo: "bar", baz: "qux" },
            {
              onOptimistic: () => {
                duringWrite = cachedPairs(rng.key);
              },
            },
          );
          expect(duringWrite).toEqual([
            { range: rng.key, key: "foo", value: "bar" },
            { range: rng.key, key: "baz", value: "qux" },
          ]);
        } finally {
          off();
        }
      });

      it("should restore the previous pairs when the set fails", async () => {
        const { rng, off } = await openRange();
        try {
          await rng.kv.set("foo", "bar");
          await expect(
            rng.kv.set("foo", "replacement", { onOptimistic: fail }),
          ).rejects.toBe(WRITE_FAILED);
          expect(cachedPairs(rng.key)).toEqual([
            { range: rng.key, key: "foo", value: "bar" },
          ]);
        } finally {
          off();
        }
      });

      it("should drop a pair before the delete commits", async () => {
        const { rng, off } = await openRange();
        try {
          await rng.kv.set("foo", "bar");
          let duringWrite: ranger.kv.Pair[] = [];
          await rng.kv.delete("foo", {
            onOptimistic: () => {
              duringWrite = cachedPairs(rng.key);
            },
          });
          expect(duringWrite).toEqual([]);
        } finally {
          off();
        }
      });

      it("should restore a pair when the delete fails", async () => {
        const { rng, off } = await openRange();
        try {
          await rng.kv.set("foo", "bar");
          await expect(rng.kv.delete("foo", { onOptimistic: fail })).rejects.toBe(
            WRITE_FAILED,
          );
          expect(cachedPairs(rng.key)).toEqual([
            { range: rng.key, key: "foo", value: "bar" },
          ]);
        } finally {
          off();
        }
      });
    });
  });
});

// A second client with its own cache: its writes reach the first client only
// through the cluster's change streams, never through a shared cache.
const remote = createTestClient();

const createRange = async (c = client, overrides: Partial<ranger.New> = {}) => {
  const start = TimeStamp.now();
  return await c.ranges.create({
    name: `qry-${id.create()}`,
    timeRange: new TimeRange(start, start.add(TimeStamp.seconds(1))),
    ...overrides,
  });
};

describe("cached reads", () => {
  // Changes made while the stream is still opening are lost (epoch
  // reconciliation repairs them in production); open it up front so remote
  // writes in these specs are always observed.
  beforeAll(async () => await client.connect());

  describe("retrieve", () => {
    it("coalesces concurrent single retrieves into one request", async () => {
      const created = [await createRange(), await createRange()];
      const local = createTestClient();
      await local.connect();
      const send = spyOnSend(local);
      const res = await Promise.all(
        created.map(async ({ key }) => await local.ranges.retrieve(key)),
      );
      expect(res.map(({ key }) => key)).toEqual(created.map(({ key }) => key));
      expect(
        send.mock.calls.filter(([target]) => target === "/range/retrieve"),
      ).toHaveLength(1);
    });

    it("does not reject concurrent retrieves when a key in the window is missing", async () => {
      const rng = await createRange();
      const local = createTestClient();
      await local.connect();
      const [ok, missing] = await Promise.allSettled([
        local.ranges.retrieve(rng.key),
        local.ranges.retrieve(uuid.create()),
      ]);
      expect(ok.status).toEqual("fulfilled");
      expect(missing.status).toEqual("rejected");
      if (missing.status === "rejected")
        expect(NotFoundError.matches(missing.reason)).toBe(true);
    });

    it("reflects remote changes on an unsubscribed repeat retrieve", async () => {
      const rng = await createRange();
      const first = await client.ranges.retrieve(rng.key);
      expect(first.name).toEqual(rng.name);
      const renamed = `qry-renamed-${id.create()}`;
      await remote.ranges.rename(rng.key, renamed);
      // Unsubscribed queries hold no frozen answer: repeat retrieves refetch
      // and converge on the remote change once it streams in.
      await expect
        .poll(async () => (await client.ranges.retrieve(rng.key)).name)
        .toBe(renamed);
    });

    it("preserves key order across cached and fetched entries", async () => {
      const a = await createRange();
      const b = await createRange();
      await client.ranges.retrieve(b.key);
      const res = await client.ranges.retrieve([a.key, b.key]);
      expect(res.map((r) => r.key)).toEqual([a.key, b.key]);
    });
  });

  describe("writes during a fetch", () => {
    const cachedName = (c: Synnax, key: ranger.Key): string | undefined => {
      const cached = c.ranges.watch(key).getCached();
      return query.isLive(cached) ? cached.name : undefined;
    };

    const expectRenameKept = async (
      fetch: (local: Synnax, rng: ranger.Range) => Promise<unknown>,
    ) => {
      const rng = await createRange();
      const local = createTestClient();
      await local.connect();
      await waitForStreamLive(local.connection);
      await local.ranges.retrieve(rng.key);
      const { responded, release } = holdResponses(local, "/range/retrieve");
      const fetching = fetch(local, rng);
      await responded;
      const renamed = `qry-renamed-${id.create()}`;
      await remote.ranges.rename(rng.key, renamed);
      await expect.poll(() => cachedName(local, rng.key)).toBe(renamed);
      release();
      await fetching;
      expect(cachedName(local, rng.key)).toBe(renamed);
    };

    it("keeps a change that streams in during a search", async () => {
      await expectRenameKept(
        async (local, rng) => await local.ranges.retrieve({ searchTerm: rng.name }),
      );
    });

    it("keeps a change that streams in during a retrieve by name", async () => {
      await expectRenameKept(
        async (local, rng) => await local.ranges.retrieve(rng.name),
      );
    });
  });

  describe("getCached", () => {
    it("serves a key query straight from the record written by create", async () => {
      const rng = await createRange();
      const cached = expectLive(client.ranges.watch(rng.key).getCached());
      expect(cached.name).toEqual(rng.name);
    });
  });

  describe("onChange", () => {
    it("delivers a remote rename to a subscribed single query", async () => {
      const rng = await createRange();
      const handler = vi.fn();
      const off = client.ranges.watch(rng.key).onChange(handler);
      try {
        await client.ranges.retrieve(rng.key);
        const renamed = `qry-renamed-${id.create()}`;
        await remote.ranges.rename(rng.key, renamed);
        await expect
          .poll(() => {
            const cached = client.ranges.watch(rng.key).getCached();
            return query.isLive(cached) && cached.name === renamed;
          })
          .toBe(true);
        expect(handler).toHaveBeenCalled();
      } finally {
        off();
      }
    });

    it("delivers a remote delete as a deleted result carrying the corpse", async () => {
      const rng = await createRange();
      const results: Array<query.Cached<ranger.Range> | undefined> = [];
      const off = client.ranges.watch(rng.key).onChange((r) => results.push(r));
      try {
        await client.ranges.retrieve(rng.key);
        await remote.ranges.delete(rng.key);
        await expect
          .poll(() => query.Deleted.matches(client.ranges.watch(rng.key).getCached()))
          .toBe(true);
        const last = expectDeleted(results.at(-1));
        expect(last.corpse.name).toEqual(rng.name);
        await expect(client.ranges.retrieve(rng.key)).rejects.toThrow("deleted");
      } finally {
        off();
      }
    });

    it("delivers the delete of a nested range that a remote delete cascades to", async () => {
      const parent = await createRange();
      const child = await createRange(client, { parent: parent.key });
      const off = client.ranges.watch(child.key).onChange(vi.fn());
      try {
        await client.ranges.retrieve(child.key);
        await remote.ranges.delete(parent.key);
        await expect
          .poll(() => query.Deleted.matches(client.ranges.watch(child.key).getCached()))
          .toBe(true);
      } finally {
        off();
      }
    });

    it("serves a record recreated while nothing subscribed to its delete", async () => {
      const rng = await createRange();
      const off = client.ranges.watch(rng.key).onChange(() => {});
      await client.ranges.retrieve(rng.key);
      await remote.ranges.delete(rng.key);
      await expect
        .poll(() => query.Deleted.matches(client.ranges.watch(rng.key).getCached()))
        .toBe(true);
      // A surface that tombstones drops its subscriptions, so the recreate
      // lands with nothing listening and the resubscribe follows it.
      off();
      await client.ranges.create({
        key: rng.key,
        name: rng.name,
        timeRange: rng.timeRange,
      });
      const off2 = client.ranges.watch(rng.key).onChange(() => {});
      try {
        expect(query.isLive(client.ranges.watch(rng.key).getCached())).toBe(true);
        expect((await client.ranges.retrieve(rng.key)).name).toEqual(rng.name);
      } finally {
        off2();
      }
    });

    it("delivers a remote label attachment", async () => {
      const rng = await createRange();
      const lbl = await remote.labels.create({
        name: `qry-label-${id.create()}`,
        color: "#E774D0",
      });
      const off = client.ranges.watch(rng.key).onChange(vi.fn());
      try {
        await client.ranges.retrieve(rng.key);
        await remote.labels.label(rng.resourceID, [lbl.key]);
        await expect
          .poll(() => {
            const cached = client.ranges.watch(rng.key).getCached();
            return query.isLive(cached) && cached.labels.includes(lbl.key);
          })
          .toBe(true);
      } finally {
        off();
      }
    });

    it("drops a deleted label from the cached ranges that hold it", async () => {
      const [kept, deleted] = await client.labels.create([
        { name: `qry-kept-${id.create()}`, color: "#E774D0" },
        { name: `qry-deleted-${id.create()}`, color: "#E774D0" },
      ]);
      const rng = await createRange(client, { labels: [kept.key, deleted.key] });
      const off = client.ranges.watch(rng.key).onChange(vi.fn());
      try {
        await client.ranges.retrieve(rng.key);
        await remote.labels.delete(deleted.key);
        await expect
          .poll(() => {
            const cached = client.ranges.watch(rng.key).getCached();
            return query.isLive(cached) && cached.labels;
          })
          .toEqual([kept.key]);
      } finally {
        off();
      }
    });
  });

  describe("children", () => {
    it("retrieves children and appends a remotely created child while subscribed", async () => {
      const parent = await createRange();
      const child = await createRange(client, { parent: parent.key });
      const list = client.ranges.watch({ parents: [parent.key] });
      const off = list.onChange(vi.fn());
      try {
        const children = await list.retrieve();
        expect(children.records.map((c) => c.key)).toContain(child.key);
        const second = await createRange(remote, { parent: parent.key });
        await expect
          .poll(() => list.getCached()?.records.some((c) => c.key === second.key))
          .toBe(true);
      } finally {
        off();
      }
    });

    it("pages children with a limit and a cursor in a stable order", async () => {
      const parent = await createRange();
      await Promise.all(
        Array.from({ length: 5 }, () => createRange(client, { parent: parent.key })),
      );
      const parents = [parent.key];
      const all = await client.ranges.retrieve({ parents });
      expect(all).toHaveLength(5);
      const first = await client.ranges.retrieve({ parents, limit: 3 });
      const second = await client.ranges.retrieve({
        parents,
        limit: 3,
        after: { key: first[2].key },
      });
      expect(first).toHaveLength(3);
      expect(second).toHaveLength(2);
      const paged = [...first, ...second].map((r) => r.key).sort();
      expect(paged).toEqual(all.map((r) => r.key).sort());
    });

    it("sends the page bounds to the cluster instead of fetching every child", async () => {
      const parent = await createRange();
      await Promise.all(
        Array.from({ length: 3 }, () => createRange(client, { parent: parent.key })),
      );
      const send = spyOnSend(client);
      try {
        const page = await client.ranges.retrieve({ parents: [parent.key], limit: 2 });
        expect(page).toHaveLength(2);
        expect(send).toHaveBeenCalledWith(
          "/range/retrieve",
          expect.objectContaining({ where: { parents: [parent.key] }, limit: 2 }),
          expect.anything(),
          expect.anything(),
        );
      } finally {
        send.mockRestore();
      }
    });
  });

  describe("parent", () => {
    it("retrieves the parent of a child range", async () => {
      const parent = await createRange();
      const child = await createRange(client, { parent: parent.key });
      const res = await client.ranges.retrieveParent(child.key);
      expect(res?.key).toEqual(parent.key);
      expect((await client.ranges.parent.retrieve(child.key))?.key).toEqual(parent.key);
      expect(
        expectLive(client.ranges.parent.watch(child.key).getCached())?.key,
      ).toEqual(parent.key);
    });

    it("throws a NotFoundError for a range that does not exist", async () => {
      await expect(client.ranges.retrieveParent(uuid.create())).rejects.toThrow(
        NotFoundError,
      );
    });

    it("moves a subscribed parent answer when the child moves", async () => {
      const [first, second] = await Promise.all([createRange(), createRange()]);
      const child = await createRange(client, { parent: first.key });
      const off = client.ranges.parent.watch(child.key).onChange(vi.fn());
      try {
        expect((await client.ranges.parent.retrieve(child.key))?.key).toEqual(
          first.key,
        );
        await client.ranges.dispatch(
          child.key,
          ranger.set({ path: ["parent"], value: second.key }),
        );
        await expect
          .poll(() => {
            const cached = client.ranges.parent.watch(child.key).getCached();
            return query.isLive(cached) ? cached?.key : undefined;
          })
          .toEqual(second.key);
      } finally {
        off();
      }
    });

    it("updates a subscribed parent answer when the parent is renamed", async () => {
      const parent = await createRange();
      const child = await createRange(client, { parent: parent.key });
      const off = client.ranges.parent.watch(child.key).onChange(vi.fn());
      try {
        await client.ranges.parent.retrieve(child.key);
        const name = `renamed-${id.create()}`;
        await client.ranges.rename(parent.key, name);
        await expect
          .poll(() => {
            const cached = client.ranges.parent.watch(child.key).getCached();
            return query.isLive(cached) ? cached?.name : undefined;
          })
          .toEqual(name);
      } finally {
        off();
      }
    });
  });

  describe("labels membership", () => {
    it("admits a newly labeled range and evicts an unlabeled one", async () => {
      const lbl = await client.labels.create({
        name: `qry-lbl-${id.create()}`,
        color: "#FF0000",
      });
      const labeled = await createRange();
      await client.labels.label(ranger.resourceID(labeled.key), [lbl.key]);
      const handler = vi.fn();
      const list = client.ranges.watch({ where: { labels: [lbl.key] } });
      const off = list.onChange(handler);
      try {
        const initial = await list.retrieve();
        expect(initial.records.map((r) => r.key)).toEqual([labeled.key]);
        const late = await createRange();
        await client.labels.label(ranger.resourceID(late.key), [lbl.key]);
        await expect
          .poll(() => list.getCached()?.records.some((r) => r.key === late.key))
          .toBe(true);
        await client.labels.remove(ranger.resourceID(labeled.key), [lbl.key]);
        await expect
          .poll(() => list.getCached()?.records.every((r) => r.key !== labeled.key))
          .toBe(true);
      } finally {
        off();
      }
    });
  });

  describe("kv", () => {
    it("retrieves a range's pairs and delivers remote sets to a subscription", async () => {
      const rng = await createRange();
      await rng.kv.set("foo", "bar");
      const handler = vi.fn();
      const off = client.ranges.kv.watch(rng.key).onChange(handler);
      try {
        const pairs = await client.ranges.kv.retrieve(rng.key);
        expect(pairs).toEqual([{ range: rng.key, key: "foo", value: "bar" }]);
        const remoteRng = await remote.ranges.retrieve(rng.key);
        await remoteRng.kv.set("baz", "qux");
        await expect
          .poll(() => {
            const cached = client.ranges.kv.watch(rng.key).getCached();
            return (
              query.isLive(cached) &&
              cached.some((p) => p.key === "baz" && p.value === "qux")
            );
          })
          .toBe(true);
        await remoteRng.kv.delete("foo");
        await expect
          .poll(() => {
            const cached = client.ranges.kv.watch(rng.key).getCached();
            return query.isLive(cached) && cached.every((p) => p.key !== "foo");
          })
          .toBe(true);
      } finally {
        off();
      }
    });
  });

  describe("resolvedAliases", () => {
    const createChannel = async () =>
      await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });

    it("should resolve the aliases of the range and of each range above it", async () => {
      const [own, inherited] = await Promise.all([createChannel(), createChannel()]);
      const parent = await createRange();
      const child = await createRange(client, { parent: parent.key });
      await parent.setAlias(inherited.key, "from parent");
      await child.setAlias(own.key, "own");
      expect(await client.ranges.resolvedAliases.retrieve(child.key)).toEqual({
        [own.key]: "own",
        [inherited.key]: "from parent",
      });
    });

    it("should read the aliases again when a range above sets one", async () => {
      const ch = await createChannel();
      const grandparent = await createRange();
      const parent = await createRange(client, { parent: grandparent.key });
      const child = await createRange(client, { parent: parent.key });
      const off = client.ranges.resolvedAliases.watch(child.key).onChange(vi.fn());
      try {
        expect(await client.ranges.resolvedAliases.retrieve(child.key)).toEqual({});
        const remoteGrandparent = await remote.ranges.retrieve(grandparent.key);
        await remoteGrandparent.setAlias(ch.key, "far");
        await expect
          .poll(() => client.ranges.resolvedAliases.watch(child.key).getCached())
          .toEqual({ [ch.key]: "far" });
      } finally {
        off();
      }
    });

    it("should keep the newer aliases when an older read lands late", async () => {
      const ch = await createChannel();
      const rng = await createRange();
      const local = createTestClient();
      await local.connect();
      await waitForStreamLive(local.connection);
      const watched = local.ranges.resolvedAliases.watch(rng.key);
      const off = watched.onChange(vi.fn());
      try {
        expect(await local.ranges.resolvedAliases.retrieve(rng.key)).toEqual({});
        const { held } = holdEachResponse(local, "/range/alias/list");
        await rng.setAlias(ch.key, "old");
        const older = await held(0);
        await rng.setAlias(ch.key, "new");
        (await held(1)).release();
        await expect.poll(() => watched.getCached()).toEqual({ [ch.key]: "new" });
        older.release();
        // A timer runs only after the late answer's microtasks.
        await sleep.sleep(TimeSpan.ZERO);
        expect(watched.getCached()).toEqual({ [ch.key]: "new" });
      } finally {
        off();
      }
    });

    /**
     * Starts the first read of the aliases on a fresh client, then holds it while the
     * change stream comes live and refetches the answer.
     */
    const overtakeFirstRead = async (rng: ranger.Range) => {
      const local = createTestClient();
      const { held } = holdEachResponse(local, "/range/alias/list");
      const off = local.ranges.resolvedAliases.watch(rng.key).onChange(vi.fn());
      onTestFinished(off);
      const read = local.ranges.resolvedAliases.retrieve(rng.key);
      const first = await held(0);
      await waitForStreamLive(local.connection);
      return { read, first, refetch: await held(1) };
    };

    it("should answer the first read when a refetch overtakes it", async () => {
      const ch = await createChannel();
      const rng = await createRange();
      await rng.setAlias(ch.key, "first");
      const { read, first, refetch } = await overtakeFirstRead(rng);
      first.release();
      await expect(read).resolves.toEqual({ [ch.key]: "first" });
      refetch.release();
    });

    it("should answer the first read when the refetch that overtakes it fails", async () => {
      const ch = await createChannel();
      const rng = await createRange();
      await rng.setAlias(ch.key, "first");
      const { read, first, refetch } = await overtakeFirstRead(rng);
      refetch.fail(new Error("refetch failed"));
      first.release();
      await expect(read).resolves.toEqual({ [ch.key]: "first" });
    });
  });
});

describe("store", () => {
  it("caches sugared sets and corpses deletes from live signals", async () => {
    await client.connect();
    const range = await createRange();
    await expect
      .poll(() => {
        const cached = client.ranges.watch(range.key).getCached();
        return query.isLive(cached) ? cached.name : undefined;
      })
      .toEqual(range.name);
    await client.ranges.delete(range.key);
    await expect
      .poll(() => query.Deleted.matches(client.ranges.watch(range.key).getCached()))
      .toBe(true);
    const cached = expectDeleted(client.ranges.watch(range.key).getCached());
    expect(cached.corpse.name).toEqual(range.name);
  });
});

describe("dispatch", () => {
  beforeAll(async () => {
    await client.connect();
    await remote.connect();
  });
  const rename = (value: string) => ranger.set({ path: ["name"], value });

  it("should store the actions and keep them in the cached range", async () => {
    const rng = await createRange();
    const end = rng.timeRange.end.add(TimeSpan.seconds(9));
    expect(
      await client.ranges.dispatch(rng.key, [
        rename("Cold flow"),
        ranger.set({ path: ["timeRange", "end"], value: end.valueOf() }),
      ]),
    ).toBe(true);
    const stored = await createTestClient().ranges.retrieve(rng.key);
    expect(stored.name).toBe("Cold flow");
    expect(stored.timeRange.end).toEqual(end);
    const cached = expectLive(client.ranges.watch(rng.key).getCached());
    expect(cached.name).toBe("Cold flow");
    expect(cached.timeRange.end).toEqual(end);
  });

  it("should keep a dispatch that lands before an older fetch", async () => {
    const rng = await createRange();
    await remote.ranges.retrieve(rng.key);
    const local = createTestClient();
    await local.connect();
    await waitForStreamLive(local.connection);
    await local.ranges.retrieve(rng.key);
    const { responded, release } = holdResponses(local, "/range/retrieve");
    const fetching = local.ranges.retrieve({ name: [rng.name] });
    await responded;
    await remote.ranges.dispatch(rng.key, rename("Cold flow"));
    await expect
      .poll(() => expectLive(local.ranges.watch(rng.key).getCached()).name)
      .toBe("Cold flow");
    release();
    await fetching;
    expect(expectLive(local.ranges.watch(rng.key).getCached()).name).toBe("Cold flow");
  });

  it("should return false without a send when the range is not cached", async () => {
    const rng = await createRange(remote);
    const fresh = createTestClient();
    const send = spyOnSend(fresh);
    expect(await fresh.ranges.dispatch(rng.key, rename("Cold flow"))).toBe(false);
    expect(send).not.toHaveBeenCalled();
  });

  it("should drop actions the Core refuses from the cached range", async () => {
    const rng = await createRange();
    await expect(
      client.ranges.dispatch(rng.key, [
        rename("Cold flow"),
        ranger.set({ path: ["timeRange", "start"], value: "0" }),
      ]),
    ).rejects.toThrow("time_range.start");
    const cached = expectLive(client.ranges.watch(rng.key).getCached());
    expect(cached.name).toBe(rng.name);
    expect(cached.timeRange).toEqual(rng.timeRange);
  });

  it("should throw for an action the range cannot hold without a send", async () => {
    const rng = await createRange();
    const send = spyOnSend(client);
    await expect(
      client.ranges.dispatch(rng.key, ranger.set({ path: ["key"], value: "k" })),
    ).rejects.toThrow("Failed to parse range");
    expect(send).not.toHaveBeenCalled();
    expect(expectLive(client.ranges.watch(rng.key).getCached()).name).toBe(rng.name);
  });

  it("should reopen a range with an end that a number cannot hold", async () => {
    const rng = await createRange();
    const { start } = rng.timeRange.numeric;
    expect(
      await client.ranges.dispatch(
        rng.key,
        ranger.set({
          path: ["timeRange"],
          value: { start, end: TimeStamp.MAX.nanoseconds },
        }),
      ),
    ).toBe(true);
    const stored = await createTestClient().ranges.retrieve(rng.key);
    expect(stored.timeRange.end).toEqual(TimeStamp.MAX);
  });

  it("should send a transaction as one dispatch", async () => {
    const rng = await createRange();
    const send = spyOnSend(client);
    const tx = client.ranges.beginTransaction(rng.key);
    tx.add(rename("Cold flow"));
    tx.add(ranger.set({ path: ["color"], value: "#ff0000" }));
    expect(expectLive(client.ranges.watch(rng.key).getCached()).name).toBe("Cold flow");
    expect(await tx.commit()).toBe(true);
    expect(
      send.mock.calls.filter(([target]) => target === "/range/dispatch"),
    ).toHaveLength(1);
    const stored = await createTestClient().ranges.retrieve(rng.key);
    expect(stored.name).toBe("Cold flow");
    expect(stored.color).toEqual(color.construct("#ff0000"));
  });

  it("applies another client's dispatch to a cached range", async () => {
    const rng = await createRange();
    await remote.ranges.retrieve(rng.key);
    await client.ranges.dispatch(rng.key, rename("Cold flow"));
    await expect
      .poll(() => {
        const cached = remote.ranges.watch(rng.key).getCached();
        return query.isLive(cached) ? cached.name : undefined;
      })
      .toBe("Cold flow");
  });

  it("ingests a range that another client creates", async () => {
    const rng = await createRange(remote);
    await expect
      .poll(() => {
        const cached = client.ranges.watch(rng.key).getCached();
        return query.isLive(cached) ? cached.name : undefined;
      })
      .toBe(rng.name);
  });
});
