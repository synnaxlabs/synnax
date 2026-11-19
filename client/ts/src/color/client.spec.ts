// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color as xcolor, id, uuid } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, onTestFinished } from "vitest";

import { color } from "@/color";
import { NotFoundError } from "@/errors";
import { group } from "@/group";
import { query } from "@/query";
import { resource } from "@/resource";
import { createTestClient, expectDeleted, expectLive, spyOnSend } from "@/testutil";

const client = createTestClient();
const remote = createTestClient();

const createNamed = async (c = client): Promise<color.Named> =>
  await c.colors.create({ name: `color-${id.create()}`, light: "#00ff00" });

describe("color.Client", () => {
  describe("create", () => {
    it("should create a named color", async () => {
      const named = await client.colors.create({
        name: "Nominal",
        light: "#00ff00",
        dark: "#008800",
      });
      expect(named.key).not.toHaveLength(0);
      expect(named.light).toEqual(xcolor.construct("#00ff00"));
      expect(named.dark).toEqual(xcolor.construct("#008800"));
    });

    it("should create many named colors", async () => {
      const prefix = `many-${id.create()}`;
      const created = await client.colors.create([
        { name: `${prefix}-1`, light: "#00ff00" },
        { name: `${prefix}-2`, light: "#ff0000" },
      ]);
      expect(created.map((n) => n.name)).toEqual([`${prefix}-1`, `${prefix}-2`]);
    });

    it("should reject a named color with an empty name", async () => {
      await expect(
        client.colors.create({ name: "", light: "#00ff00" }),
      ).rejects.toThrow("name");
    });
  });

  describe("retrieve", () => {
    it("should retrieve a named color by its key", async () => {
      const named = await createNamed();
      expect(await createTestClient().colors.retrieve(named.key)).toEqual(named);
    });

    it("should retrieve a named color by its name", async () => {
      const named = await createNamed();
      expect(await createTestClient().colors.retrieve(named.name)).toEqual(named);
    });

    it("should throw a NotFoundError for a name that no named color has", async () => {
      const name = `missing-${id.create()}`;
      await expect(client.colors.retrieve(name)).rejects.toThrow(NotFoundError);
    });

    it("should retrieve named colors by where", async () => {
      const prefix = `where-${id.create()}`;
      const [kept, dropped] = await client.colors.create([
        { name: `${prefix}-kept`, light: "#00ff00" },
        { name: `${prefix}-dropped`, light: "#00ff00" },
      ]);
      const retrieved = await createTestClient().colors.retrieve({
        where: { name: { regex: `^${prefix}` }, not: { key: { eq: dropped.key } } },
      });
      expect(retrieved.map((n) => n.key)).toEqual([kept.key]);
    });

    it("should retrieve named colors by search term", async () => {
      const prefix = `searchable-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.colors.create(names.map((name) => ({ name, light: "#00ff00" })));
      await expect
        .poll(async () => {
          const results = await client.colors.retrieve({ searchTerm: prefix });
          return results.map((n) => n.name).sort();
        })
        .toEqual(names);
    });

    it("should drop the keys that do not exist from a key list", async () => {
      const named = await createNamed();
      const retrieved = await createTestClient().colors.retrieve({
        key: [named.key, uuid.create()],
      });
      expect(retrieved.map((n) => n.key)).toEqual([named.key]);
    });
  });

  describe("delete", () => {
    it("should delete a named color by its key", async () => {
      const named = await createNamed();
      await client.colors.delete(named.key);
      await expect(createTestClient().colors.retrieve(named.key)).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("retrieveBuiltins", () => {
    it("should return the shipped values of the built-in colors", async () => {
      const builtins = await client.colors.retrieveBuiltins();
      expect(builtins).toHaveLength(16);
      expect(builtins[0].name).toEqual("Series 1");
      expect(xcolor.hex(builtins[0].light)).toEqual("#dc136c");
    });
  });

  describe("store", () => {
    beforeAll(async () => {
      await client.connect();
      await remote.connect();
    });

    it("should serve a key query from the record that create writes", async () => {
      const named = await createNamed();
      expect(expectLive(client.colors.watch(named.key).getCached())).toEqual(named);
    });

    it("should hold the corpse of a remote delete", async () => {
      const named = await createNamed();
      await remote.colors.delete(named.key);
      await expect
        .poll(() => query.Deleted.matches(client.colors.watch(named.key).getCached()))
        .toBe(true);
      const cached = expectDeleted(client.colors.watch(named.key).getCached());
      expect(cached.corpse.name).toEqual(named.name);
    });

    it("should ingest a named color that another client creates", async () => {
      const named = await createNamed(remote);
      await expect
        .poll(() => {
          const cached = client.colors.watch(named.key).getCached();
          return query.isLive(cached) ? cached.name : undefined;
        })
        .toBe(named.name);
    });
  });

  describe("palettes", () => {
    const moveRequests = (named: color.Named, to?: group.Group) =>
      client.groups.moveRequests(
        [color.resourceID(named.key)],
        to == null ? undefined : group.resourceID(to.key),
      );

    it("should move a named color into a palette and out", async () => {
      const palette = await client.groups.create({
        scope: "color",
        name: `palette-${id.create()}`,
      });
      const named = await createNamed();
      await client.batch.send(moveRequests(named, palette));
      expect((await createTestClient().colors.retrieve(named.key)).group).toBe(
        palette.key,
      );
      expect(expectLive(client.colors.watch(named.key).getCached()).group).toBe(
        palette.key,
      );
      await client.batch.send(moveRequests(named));
      expect(
        (await createTestClient().colors.retrieve(named.key)).group,
      ).toBeUndefined();
    });

    it("should retrieve the named colors of a palette", async () => {
      const palette = await client.groups.create({
        scope: "color",
        name: `palette-${id.create()}`,
      });
      const [inPalette, loose] = await Promise.all([createNamed(), createNamed()]);
      await client.batch.send(moveRequests(inPalette, palette));
      const reader = createTestClient();
      const grouped = await reader.colors.retrieve({ groups: [palette.key] });
      expect(grouped.map((n) => n.key)).toEqual([inPalette.key]);
      const ungrouped = await reader.colors.retrieve({
        key: [inPalette.key, loose.key],
        grouped: false,
      });
      expect(ungrouped.map((n) => n.key)).toEqual([loose.key]);
    });

    it("should add a named color moved into a watched palette", async () => {
      const palette = await client.groups.create({
        scope: "color",
        name: `palette-${id.create()}`,
      });
      const list = client.colors.watch({ groups: [palette.key] });
      onTestFinished(list.onChange(() => {}));
      await list.retrieve();
      const named = await createNamed();
      await client.batch.send(moveRequests(named, palette));
      await expect
        .poll(() => list.getCached()?.records.map((n) => n.key))
        .toEqual([named.key]);
    });

    it("should set the positions of named colors in one batch", async () => {
      const [a, b] = await Promise.all([createNamed(), createNamed()]);
      await client.batch.send([
        client.colors.setPositionRequest(a.key, "a1"),
        client.colors.setPositionRequest(b.key, "a0"),
      ]);
      expect(expectLive(client.colors.watch(a.key).getCached()).position).toBe("a1");
      const reader = createTestClient();
      expect((await reader.colors.retrieve(a.key)).position).toBe("a1");
      expect((await reader.colors.retrieve(b.key)).position).toBe("a0");
    });

    it("should delete a palette and its named colors in one batch", async () => {
      const palette = await client.groups.create({
        scope: "color",
        name: `palette-${id.create()}`,
      });
      const named = await createNamed();
      await client.batch.send(moveRequests(named, palette));
      await client.batch.send([
        client.colors.deleteRequest(named.key),
        client.groups.deleteRequest(palette.key),
      ]);
      const reader = createTestClient();
      await expect(reader.colors.retrieve(named.key)).rejects.toThrow(NotFoundError);
      await expect(reader.groups.retrieve({ key: palette.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should keep every named color when the palette delete fails", async () => {
      const palette = await client.groups.create({
        scope: "color",
        name: `palette-${id.create()}`,
      });
      const [deleted, kept] = await Promise.all([createNamed(), createNamed()]);
      await client.batch.send([
        ...moveRequests(deleted, palette),
        ...moveRequests(kept, palette),
      ]);
      await expect(
        client.batch.send([
          client.colors.deleteRequest(deleted.key),
          client.groups.deleteRequest(palette.key),
        ]),
      ).rejects.toThrow(`cannot delete group ${palette.name} because it is not empty`);
      const reader = createTestClient();
      expect((await reader.colors.retrieve(deleted.key)).key).toBe(deleted.key);
    });

    it("should refuse a move into a group of another scope", async () => {
      const channels = await client.groups.create({
        scope: "channel",
        name: `channels-${id.create()}`,
      });
      const named = await createNamed();
      await expect(client.batch.send(moveRequests(named, channels))).rejects.toThrow(
        "a color cannot sit in a channel group",
      );
    });
  });

  describe("resource search", () => {
    it("should keep a subscribed search of named colors current", async () => {
      const list = client.resources.watch({
        searchTerm: "",
        types: ["color"],
        limit: 1000,
      });
      onTestFinished(list.onChange(() => {}));
      await list.retrieve();
      const named = await createNamed();
      const key = resource.idToString(color.resourceID(named.key));
      const nameOf = () => list.getCached()?.records.find((r) => r.key === key)?.name;
      await expect.poll(nameOf).toBe(named.name);
      await client.colors.dispatch(
        named.key,
        color.set({ path: ["name"], value: "Re" }),
      );
      await expect.poll(nameOf).toBe("Re");
      await client.colors.delete(named.key);
      await expect.poll(nameOf).toBeUndefined();
    });
  });

  describe("dispatch", () => {
    beforeAll(async () => {
      await client.connect();
      await remote.connect();
    });
    const rename = (value: string) => color.set({ path: ["name"], value });

    it("should store the actions and keep them in the cached named color", async () => {
      const named = await createNamed();
      expect(
        await client.colors.dispatch(named.key, [
          rename("Anomaly"),
          color.set({ path: ["dark"], value: "#ff0000" }),
        ]),
      ).toBe(true);
      const stored = await createTestClient().colors.retrieve(named.key);
      expect(stored.name).toBe("Anomaly");
      expect(stored.dark).toEqual(xcolor.construct("#ff0000"));
      const cached = expectLive(client.colors.watch(named.key).getCached());
      expect(cached.name).toBe("Anomaly");
      expect(cached.dark).toEqual(xcolor.construct("#ff0000"));
    });

    it("should remove the dark value", async () => {
      const named = await client.colors.create({
        name: `dark-${id.create()}`,
        light: "#00ff00",
        dark: "#008800",
      });
      await client.colors.dispatch(named.key, color.remove({ path: [], key: "dark" }));
      const stored = await createTestClient().colors.retrieve(named.key);
      expect(stored.dark).toBeUndefined();
    });

    it("should return false without a send for an uncached named color", async () => {
      const named = await createNamed(remote);
      const fresh = createTestClient();
      const send = spyOnSend(fresh);
      expect(await fresh.colors.dispatch(named.key, rename("Anomaly"))).toBe(false);
      expect(send).not.toHaveBeenCalled();
    });

    it("should drop the actions that the Core refuses from the cache", async () => {
      const named = await createNamed();
      await expect(client.colors.dispatch(named.key, rename(""))).rejects.toThrow(
        "name: required",
      );
      expect(expectLive(client.colors.watch(named.key).getCached()).name).toBe(
        named.name,
      );
    });

    it("should send a transaction as one dispatch", async () => {
      const named = await createNamed();
      const send = spyOnSend(client);
      const tx = client.colors.beginTransaction(named.key);
      tx.add(rename("Anomaly"));
      tx.add(color.set({ path: ["light"], value: "#ff0000" }));
      expect(await tx.commit()).toBe(true);
      expect(
        send.mock.calls.filter(([target]) => target === "/color/dispatch"),
      ).toHaveLength(1);
      const stored = await createTestClient().colors.retrieve(named.key);
      expect(stored.name).toBe("Anomaly");
      expect(stored.light).toEqual(xcolor.construct("#ff0000"));
    });

    it("should apply a remote dispatch to a cached named color", async () => {
      const named = await createNamed();
      await remote.colors.retrieve(named.key);
      await client.colors.dispatch(named.key, rename("Anomaly"));
      await expect
        .poll(() => {
          const cached = remote.colors.watch(named.key).getCached();
          return query.isLive(cached) ? cached.name : undefined;
        })
        .toBe("Anomaly");
    });
  });
});
