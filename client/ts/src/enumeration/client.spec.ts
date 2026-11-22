// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color as xcolor, id, uuid } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import type { color } from "@/color";
import { enumeration } from "@/enumeration";
import { NotFoundError } from "@/errors";
import { query } from "@/query";
import { createTestClient, expectDeleted, expectLive } from "@/testutil";

const client = createTestClient();
const remote = createTestClient();

const createEnumeration = async (c = client): Promise<enumeration.Enumeration> =>
  await c.enumerations.create({
    name: `enumeration-${id.create()}`,
    entries: [
      { key: "closed", value: 0, name: "Closed" },
      { key: "open", value: 1, name: "Open" },
    ],
  });

describe("enumeration.Client", () => {
  describe("create", () => {
    it("should create an enumeration with its entries", async () => {
      const created = await client.enumerations.create({
        name: "Breaker",
        entries: [
          { key: "open", value: 0, name: "Open" },
          {
            key: "tripped",
            value: 2,
            name: "Tripped",
            abnormal: true,
            color: { type: "fixed", value: "#ff0000" },
          },
        ],
      });
      expect(created.key).not.toHaveLength(0);
      expect(created.entries[1]).toEqual({
        key: "tripped",
        value: 2,
        name: "Tripped",
        abnormal: true,
        color: { type: "fixed", value: xcolor.construct("#ff0000") },
      });
      expect(await createTestClient().enumerations.retrieve(created.key)).toEqual(
        created,
      );
    });

    it("should create many enumerations", async () => {
      const prefix = `many-${id.create()}`;
      const created = await client.enumerations.create([
        { name: `${prefix}-1` },
        { name: `${prefix}-2` },
      ]);
      expect(created.map((e) => e.name)).toEqual([`${prefix}-1`, `${prefix}-2`]);
    });

    it("should reject two entries with one value", async () => {
      await expect(
        client.enumerations.create({
          name: "Breaker",
          entries: [
            { key: "a", value: 1, name: "A" },
            { key: "b", value: 1, name: "B" },
          ],
        }),
      ).rejects.toThrow("another entry has the value 1");
    });
  });

  describe("retrieve", () => {
    it("should retrieve an enumeration by its name", async () => {
      const created = await createEnumeration();
      expect(await createTestClient().enumerations.retrieve(created.name)).toEqual(
        created,
      );
    });

    it("should throw a NotFoundError for a key that no enumeration has", async () => {
      await expect(client.enumerations.retrieve(uuid.create())).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should retrieve enumerations by where", async () => {
      const prefix = `where-${id.create()}`;
      const [kept, dropped] = await client.enumerations.create([
        { name: `${prefix}-kept` },
        { name: `${prefix}-dropped` },
      ]);
      const retrieved = await createTestClient().enumerations.retrieve({
        where: { name: { regex: `^${prefix}` }, not: { key: { eq: dropped.key } } },
      });
      expect(retrieved.map((e) => e.key)).toEqual([kept.key]);
    });

    it("should retrieve enumerations by search term", async () => {
      const prefix = `searchable-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.enumerations.create(names.map((name) => ({ name })));
      await expect
        .poll(async () => {
          const results = await client.enumerations.retrieve({ searchTerm: prefix });
          return results.map((e) => e.name).sort();
        })
        .toEqual(names);
    });
  });

  describe("dispatch", () => {
    it("should insert an entry and store it on the Core", async () => {
      const created = await createEnumeration();
      await client.enumerations.retrieve(created.key);
      expect(
        await client.enumerations.dispatch(
          created.key,
          enumeration.insert({
            path: ["entries"],
            value: { key: "fault", value: 2, name: "Fault", abnormal: true },
            after: "open",
          }),
        ),
      ).toBe(true);
      const stored = await createTestClient().enumerations.retrieve(created.key);
      expect(stored.entries.map((e) => e.name)).toEqual(["Closed", "Open", "Fault"]);
    });

    it("should store a named color on an entry", async () => {
      const named = await client.colors.create({
        name: `color-${id.create()}`,
        light: "#00ff00",
      });
      const created = await createEnumeration();
      await client.enumerations.retrieve(created.key);
      const ref: color.Ref = { type: "named", key: named.key };
      await client.enumerations.dispatch(
        created.key,
        enumeration.set({ path: ["entries", "open", "color"], value: ref }),
      );
      const stored = await createTestClient().enumerations.retrieve(created.key);
      expect(stored.entries[1].color).toEqual(ref);
    });
  });

  describe("delete", () => {
    it("should delete an enumeration by its key", async () => {
      const created = await createEnumeration();
      await client.enumerations.delete(created.key);
      await expect(
        createTestClient().enumerations.retrieve(created.key),
      ).rejects.toThrow(NotFoundError);
    });

    it("should refuse to delete a built-in enumeration", async () => {
      const [valve] = await client.enumerations.retrieveBuiltins();
      await expect(client.enumerations.delete(valve.key)).rejects.toThrow(
        "cannot delete built-in enumeration Valve",
      );
    });

    it("should delete an enumeration in a batch", async () => {
      const created = await createEnumeration();
      await client.batch.send([client.enumerations.deleteRequest(created.key)]);
      await expect(
        createTestClient().enumerations.retrieve(created.key),
      ).rejects.toThrow(NotFoundError);
    });
  });

  describe("replace", () => {
    it("should delete the replaced enumeration and keep the other", async () => {
      const [from, to] = await Promise.all([createEnumeration(), createEnumeration()]);
      await client.enumerations.replace(from.key, to.key);
      const other = createTestClient();
      await expect(other.enumerations.retrieve(from.key)).rejects.toThrow(
        NotFoundError,
      );
      expect((await other.enumerations.retrieve(to.key)).name).toEqual(to.name);
    });

    it("should refuse to replace a built-in enumeration", async () => {
      const [valve] = await client.enumerations.retrieveBuiltins();
      const to = await createEnumeration();
      await expect(client.enumerations.replace(valve.key, to.key)).rejects.toThrow(
        "cannot delete built-in enumeration Valve",
      );
    });
  });

  describe("retrieveBuiltins", () => {
    it("should return the shipped values of the built-in enumerations", async () => {
      const builtins = await client.enumerations.retrieveBuiltins();
      expect(builtins.map((e) => e.name)).toEqual([
        "Valve",
        "Normally open valve",
        "Pump",
        "Switch",
        "Light",
      ]);
      expect(builtins[0].entries.map((e) => [e.value, e.name])).toEqual([
        [0, "Closed"],
        [1, "Open"],
      ]);
    });
  });

  describe("retrieveDependents", () => {
    it("should return no records for an enumeration that nothing uses", async () => {
      const created = await createEnumeration();
      expect(await client.enumerations.retrieveDependents(created.key)).toEqual([]);
    });
  });

  describe("store", () => {
    beforeAll(async () => {
      await client.connect();
      await remote.connect();
    });

    it("should serve a key query from the record that create writes", async () => {
      const created = await createEnumeration();
      expect(expectLive(client.enumerations.watch(created.key).getCached())).toEqual(
        created,
      );
    });

    it("should hold the corpse of a remote delete", async () => {
      const created = await createEnumeration();
      await remote.enumerations.delete(created.key);
      await expect
        .poll(() =>
          query.Deleted.matches(client.enumerations.watch(created.key).getCached()),
        )
        .toBe(true);
      const cached = expectDeleted(client.enumerations.watch(created.key).getCached());
      expect(cached.corpse.name).toEqual(created.name);
    });

    it("should apply an entry edit that another client makes", async () => {
      const created = await createEnumeration();
      await client.enumerations.retrieve(created.key);
      await remote.enumerations.retrieve(created.key);
      await remote.enumerations.dispatch(
        created.key,
        enumeration.set({ path: ["entries", "open", "name"], value: "Opened" }),
      );
      await expect
        .poll(() => {
          const cached = client.enumerations.watch(created.key).getCached();
          return query.isLive(cached) ? cached.entries[1].name : undefined;
        })
        .toBe("Opened");
    });

    it("should inline a deleted named color into the cached entry", async () => {
      const named = await client.colors.create({
        name: `color-${id.create()}`,
        light: "#00ff00",
      });
      const created = await client.enumerations.create({
        name: `enumeration-${id.create()}`,
        entries: [
          { key: "on", value: 1, name: "On", color: { type: "named", key: named.key } },
        ],
      });
      await remote.colors.delete(named.key);
      await expect
        .poll(() => {
          const cached = client.enumerations.watch(created.key).getCached();
          return query.isLive(cached) ? cached.entries[0].color : undefined;
        })
        .toEqual({ type: "fixed", value: xcolor.construct("#00ff00") });
    });
  });
});
