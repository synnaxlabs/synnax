// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type change } from "@synnaxlabs/x";
import { DataType, Rate, TimeSpan, TimeStamp } from "@synnaxlabs/x/telem";
import { describe, expect, it } from "vitest";

import { QueryError } from "@/errors";
import { type ranger } from "@/ranger";
import { type NewPayload } from "@/ranger/payload";
import { newClient } from "@/setupspecs";

const client = newClient();

describe("Ranger", () => {
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
      const ranges: NewPayload[] = [
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
        QueryError,
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

  describe("page", () => {
    it("should page through ranges", async () => {
      await client.ranges.create({
        name: "My New One Second Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
      });
      await client.ranges.create({
        name: "My New Two Second Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(2)),
      });
      const ranges = await client.ranges.page(0, 1);
      expect(ranges.length).toEqual(1);
      const keys = ranges.map((r) => r.key);
      const next = await client.ranges.page(1, 1);
      expect(next.length).toEqual(1);
      expect(next.map((r) => r.key)).not.toContain(keys[0]);
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
      await expect(async () => await rng.kv.get("foo")).rejects.toThrow(QueryError);
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

    describe("observable", () => {
      it("should listen to key-value sets on the range", async () => {
        const rng = await client.ranges.create({
          name: "My New One Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        });
        const obs = await rng.kv.openTracker();
        const res = new Promise<change.Change<string, ranger.KVPair>[]>((resolve) => {
          obs.onChange((pair) => resolve(pair));
        });
        await rng.kv.set("foo", "bar");
        const pair = await res;
        expect(pair.length).toBeGreaterThan(0);
        expect(pair[0].value?.range).toEqual(rng.key);
        expect(pair[0].value?.key).toEqual("foo");
        expect(pair[0].value?.value).toEqual("bar");
      });
      it("should listen to key-value deletes on the range", async () => {
        const rng = await client.ranges.create({
          name: "My New One Second Range",
          timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        });
        await rng.kv.set("foo", "bar");
        const obs = await rng.kv.openTracker();
        const res = new Promise<change.Change<string, ranger.KVPair>[]>((resolve) => {
          obs.onChange((changes) => {
            if (changes.every((c) => c.variant === "delete")) resolve(changes);
          });
        });
        await rng.kv.delete("foo");
        const pair = await res;
        expect(pair.length).toBeGreaterThan(0);
        expect(pair[0].value?.range).toEqual(rng.key);
        expect(pair[0].value?.key).toEqual("foo");
        expect(pair[0].value?.value).toHaveLength(0);
      });
    });
  });

  describe("Alias", () => {
    describe("set + resolve", () => {
      it("should set and resolve an alias for the range", async () => {
        const ch = await client.channels.create({
          name: "My New Channel",
          dataType: DataType.FLOAT32,
          rate: Rate.hz(1),
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
          rate: Rate.hz(1),
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
          rate: Rate.hz(1),
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
