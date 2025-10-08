// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, math, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError } from "@/errors";
import { ranger } from "@/ranger";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Ranger", () => {
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
      expect(range.color).toEqual("#E774D0");
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
      await client.ranges.create(
        {
          name: "My New Child Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        },
        { parent: parentRange.ontologyID },
      );
      const children = await client.ontology.retrieveChildren(parentRange.ontologyID);
      expect(children).toHaveLength(1);
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
  });

  describe("retrieveParent", () => {
    it("should retrieve the parent of a range", async () => {
      const parentRange = await client.ranges.create({
        name: "My New Parent Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      const childRange = await client.ranges.create(
        {
          name: "My New Child Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        },
        { parent: parentRange.ontologyID },
      );
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
      await expect(async () => await rng.kv.get("foo")).rejects.toThrowError(
        NotFoundError,
      );
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
      const newRange = await client.ranges.retrieve({
        keys: [rng.key],
        includeLabels: true,
      });
      expect(newRange[0].labels).toHaveLength(1);
      expect(newRange[0].labels?.[0]).toEqual(label);
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
      await client.ontology.addChildren(parent.ontologyID, child.ontologyID);
      const newParent = await client.ranges.retrieve({
        keys: [child.key],
        includeParent: true,
      });
      expect(newParent[0].parent).toEqual(parent.payload);
    });
  });

  describe("Alias", () => {
    describe("set + resolve", () => {
      it("should set and resolve an alias for the range", async () => {
        const ch = await client.channels.create({
          name: "My New Channel",
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
    });
    describe("deleteAlias", () => {
      it("should remove an alias for the range", async () => {
        const ch = await client.channels.create({
          name: "My New Channel",
          dataType: DataType.FLOAT32,
          virtual: true,
        });
        const rng = await client.ranges.create({
          name: "My New One Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        });
        await rng.setAlias(ch.key, "myalias");
        await rng.deleteAlias(ch.key);
        expect(await rng.resolveAlias("myalias")).toBeUndefined();
      });
    });
    describe("list", () => {
      it("should list the aliases for the range", async () => {
        const ch = await client.channels.create({
          name: "My New Channel",
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
});
