// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DataType,
  id,
  math,
  TimeRange,
  TimeSpan,
  TimeStamp,
  uuid,
} from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { NotFoundError } from "@/errors";
import { type query } from "@/query";
import { ranger } from "@/ranger";
import { createTestClient } from "@/testutil";

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
      await client.ranges.create({
        name: "My New Child Range",
        timeRange: TimeStamp.now().spanRange(TimeSpan.seconds(1)),
        parent: parentRange,
      });
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
        parent: parentRange,
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
        expect(await rng.resolveAlias("myalias")).toBeUndefined();
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

  describe("getCached", () => {
    it("serves a key query straight from the record written by create", async () => {
      const rng = await createRange();
      const cached = client.ranges.getCached(rng.key);
      expect(cached?.variant).toEqual("changed");
      if (cached?.variant === "changed") expect(cached.data.name).toEqual(rng.name);
    });
  });

  describe("onChange", () => {
    it("delivers a remote rename to a subscribed single query", async () => {
      const rng = await createRange();
      const handler = vi.fn();
      const off = client.ranges.onChange(rng.key, handler);
      try {
        await client.ranges.retrieve(rng.key);
        const renamed = `qry-renamed-${id.create()}`;
        await remote.ranges.rename(rng.key, renamed);
        await expect
          .poll(() => {
            const cached = client.ranges.getCached(rng.key);
            return cached?.variant === "changed" && cached.data.name === renamed;
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
      const off = client.ranges.onChange(rng.key, (r) => results.push(r));
      try {
        await client.ranges.retrieve(rng.key);
        await remote.ranges.delete(rng.key);
        await expect
          .poll(() => client.ranges.getCached(rng.key)?.variant)
          .toBe("deleted");
        const last = results.at(-1);
        expect(last?.variant).toEqual("deleted");
        if (last?.variant === "deleted") expect(last.corpse.name).toEqual(rng.name);
        await expect(client.ranges.retrieve(rng.key)).rejects.toThrow("deleted");
      } finally {
        off();
      }
    });

    it("delivers remote label attachments through composition", async () => {
      const rng = await createRange();
      const lbl = await remote.labels.create({
        name: `qry-label-${id.create()}`,
        color: "#E774D0",
      });
      const off = client.ranges.onChange(rng.key, vi.fn());
      try {
        await client.ranges.retrieve(rng.key);
        await remote.labels.label(rng.ontologyID, [lbl.key]);
        await expect
          .poll(() => {
            const cached = client.ranges.getCached(rng.key);
            return (
              cached?.variant === "changed" &&
              (cached.data.labels ?? []).some((l) => l.key === lbl.key)
            );
          })
          .toBe(true);
      } finally {
        off();
      }
    });
  });

  describe("children", () => {
    it("retrieves children and appends a remotely created child while subscribed", async () => {
      const parent = await createRange();
      const child = await createRange(client, { parent: { key: parent.key } });
      const off = client.ranges.children.onChange(parent.key, vi.fn());
      try {
        const children = await client.ranges.children.retrieve(parent.key);
        expect(children.map((c) => c.key)).toContain(child.key);
        const second = await createRange(remote, { parent: { key: parent.key } });
        await expect
          .poll(() => {
            const cached = client.ranges.children.getCached(parent.key);
            return (
              cached?.variant === "changed" &&
              cached.data.some((c) => c.key === second.key)
            );
          })
          .toBe(true);
      } finally {
        off();
      }
    });
  });

  describe("parent", () => {
    it("retrieves the parent of a child range", async () => {
      const parent = await createRange();
      const child = await createRange(client, { parent: { key: parent.key } });
      const res = await client.ranges.retrieveParent(child.key);
      expect(res?.key).toEqual(parent.key);
      const cached = client.ranges.parent.getCached(ranger.ontologyID(child.key));
      expect(cached?.variant).toEqual("changed");
    });
  });

  describe("hasLabels membership", () => {
    it("admits a newly labeled range and evicts an unlabeled one", async () => {
      const lbl = await client.labels.create({
        name: `qry-lbl-${id.create()}`,
        color: "#FF0000",
      });
      const labeled = await createRange();
      await client.labels.label(ranger.ontologyID(labeled.key), [lbl.key]);
      const handler = vi.fn();
      const query = { hasLabels: [lbl.key] };
      const off = client.ranges.onChange(query, handler);
      try {
        const initial = await client.ranges.retrieve(query);
        expect(initial.map((r) => r.key)).toEqual([labeled.key]);
        const late = await createRange();
        await client.labels.label(ranger.ontologyID(late.key), [lbl.key]);
        await expect
          .poll(() => {
            const cached = client.ranges.getCached(query);
            return (
              cached?.variant === "changed" &&
              cached.data.some((r) => r.key === late.key)
            );
          })
          .toBe(true);
        await client.labels.remove(ranger.ontologyID(labeled.key), [lbl.key]);
        await expect
          .poll(() => {
            const cached = client.ranges.getCached(query);
            return (
              cached?.variant === "changed" &&
              cached.data.every((r) => r.key !== labeled.key)
            );
          })
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
      const off = client.ranges.kv.onChange(rng.key, handler);
      try {
        const pairs = await client.ranges.kv.retrieve(rng.key);
        expect(pairs).toEqual([{ range: rng.key, key: "foo", value: "bar" }]);
        const remoteRng = await remote.ranges.retrieve(rng.key);
        await remoteRng.kv.set("baz", "qux");
        await expect
          .poll(() => {
            const cached = client.ranges.kv.getCached(rng.key);
            return (
              cached?.variant === "changed" &&
              cached.data.some((p) => p.key === "baz" && p.value === "qux")
            );
          })
          .toBe(true);
        await remoteRng.kv.delete("foo");
        await expect
          .poll(() => {
            const cached = client.ranges.kv.getCached(rng.key);
            return (
              cached?.variant === "changed" && cached.data.every((p) => p.key !== "foo")
            );
          })
          .toBe(true);
      } finally {
        off();
      }
    });
  });
});

describe("store", () => {
  it("caches sugared sets and corpses deletes from live signals", async () => {
    await client.connect();
    const range = await createRange();
    await expect
      .poll(() => {
        const cached = client.ranges.getCached(range.key);
        return cached?.variant === "changed" ? cached.data.name : undefined;
      })
      .toEqual(range.name);
    await client.ranges.delete(range.key);
    await expect
      .poll(() => client.ranges.getCached(range.key)?.variant)
      .toBe("deleted");
    const cached = client.ranges.getCached(range.key);
    if (cached?.variant === "deleted") expect(cached.corpse.name).toEqual(range.name);
  });
});
