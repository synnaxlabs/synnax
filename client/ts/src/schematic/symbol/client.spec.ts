// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import { group } from "@/group";
import { NotFoundError } from "@/index";
import { query } from "@/query";
import { type resource } from "@/resource";
import { schematic } from "@/schematic";
import { createTestClient, expectLive } from "@/testutil";

const client = createTestClient();

const SYMBOL_DATA: schematic.symbol.New["data"] = {
  svg: "<svg></svg>",
  states: [],
  handles: [],
  variant: "sensor",
};

const createSymbolGroup = async (parent?: resource.ID): Promise<group.Group> =>
  await client.groups.create(
    parent == null
      ? { name: `symbol-group-${id.create()}`, scope: "schematic_symbol" }
      : { name: `symbol-group-${id.create()}`, parent },
  );

describe("Symbol Client", () => {
  let symbolGroup: group.Group;
  beforeAll(async () => {
    symbolGroup = await client.groups.create({
      name: "Test Symbols",
      scope: "schematic_symbol",
    });
  });
  describe("create", () => {
    it("should create a single symbol", async () => {
      const symbol = await client.schematics.symbols.create({
        name: "Test Symbol",
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "sensor",
        },
        group: symbolGroup.key,
      });
      expect(symbol.name).toBe("Test Symbol");
      expect(symbol.key).toBeDefined();
    });

    it("should create multiple symbols", async () => {
      const symbols = await client.schematics.symbols.create({
        symbols: [
          {
            name: "Symbol 1",
            data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
            group: symbolGroup.key,
          },
          {
            name: "Symbol 2",
            data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
            group: symbolGroup.key,
          },
        ],
      });
      expect(symbols).toHaveLength(2);
      expect(symbols[0].name).toBe("Symbol 1");
      expect(symbols[1].name).toBe("Symbol 2");
    });
  });

  describe("retrieve", () => {
    it("should retrieve a single symbol by key", async () => {
      const created = await client.schematics.symbols.create({
        name: "Retrieve Test",
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        group: symbolGroup.key,
      });

      const retrieved = await client.schematics.symbols.retrieve(created.key);
      expect(retrieved.key).toBe(created.key);
      expect(retrieved.name).toBe("Retrieve Test");
    });

    it("should retrieve multiple symbols by keys", async () => {
      const created = await client.schematics.symbols.create({
        symbols: [
          {
            name: "Multi Test 1",
            data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
            group: symbolGroup.key,
          },
          {
            name: "Multi Test 2",
            data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
            group: symbolGroup.key,
          },
        ],
      });

      const retrieved = await client.schematics.symbols.retrieve(
        created.map((s) => s.key),
      );
      expect(retrieved).toHaveLength(2);
    });

    it("should retrieve symbols by search term", async () => {
      const prefix = `searchable-symbol-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.schematics.symbols.create({
        symbols: names.map((name) => ({
          name,
          data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
          group: symbolGroup.key,
        })),
      });
      await expect
        .poll(async () => {
          const results = await client.schematics.symbols.retrieve({
            searchTerm: prefix,
          });
          return results.map((s) => s.name).sort();
        })
        .toEqual(names);
    });

    it("should retrieve symbols by where", async () => {
      const prefix = `where-symbol-${id.create()}`;
      const [first] = await client.schematics.symbols.create({
        symbols: [
          { name: `${prefix}-1`, data: SYMBOL_DATA, group: symbolGroup.key },
          { name: `${prefix}-2`, data: SYMBOL_DATA, group: symbolGroup.key },
        ],
      });
      await expect
        .poll(async () => {
          const results = await client.schematics.symbols.retrieve({
            searchTerm: prefix,
            where: { not: { key: { eq: first.key } } },
          });
          return results.map((s) => s.name);
        })
        .toEqual([`${prefix}-2`]);
    });
  });

  describe("group-scoped retrieve", () => {
    it("should retrieve only the symbols in the given group", async () => {
      const g = await createSymbolGroup();
      const other = await createSymbolGroup();
      const created = await client.schematics.symbols.create({
        symbols: [
          { name: "child-1", data: SYMBOL_DATA, group: g.key },
          { name: "child-2", data: SYMBOL_DATA, group: g.key },
        ],
      });
      await client.schematics.symbols.create({
        name: "outsider",
        data: SYMBOL_DATA,
        group: other.key,
      });

      const retrieved = await client.schematics.symbols.retrieve({ groups: [g.key] });
      expect(retrieved.map((s) => s.key).sort()).toEqual(
        created.map((s) => s.key).sort(),
      );
    });

    it("should combine a group scope with a search term", async () => {
      const g = await createSymbolGroup();
      const prefix = `scoped-${id.create()}`;
      await client.schematics.symbols.create({
        symbols: [
          { name: `${prefix}-valve`, data: SYMBOL_DATA, group: g.key },
          { name: "pump", data: SYMBOL_DATA, group: g.key },
        ],
      });
      await expect
        .poll(async () => {
          const results = await client.schematics.symbols.retrieve({
            groups: [g.key],
            searchTerm: prefix,
          });
          return results.map((s) => s.name);
        })
        .toEqual([`${prefix}-valve`]);
    });

    it("should combine a group with a where", async () => {
      const g = await createSymbolGroup();
      const [kept] = await client.schematics.symbols.create({
        symbols: [
          { name: "valve", data: SYMBOL_DATA, group: g.key },
          { name: "pump", data: SYMBOL_DATA, group: g.key },
        ],
      });
      const outsider = await client.schematics.symbols.create({
        name: "valve",
        data: SYMBOL_DATA,
        group: (await createSymbolGroup()).key,
      });
      const retrieved = await client.schematics.symbols.retrieve({
        where: {
          and: [{ groups: [g.key] }, { key: { in: [kept.key, outsider.key] } }],
        },
      });
      expect(retrieved.map((s) => s.key)).toEqual([kept.key]);
    });

    it("should combine a group with a flat filter", async () => {
      const g = await createSymbolGroup();
      const [kept] = await client.schematics.symbols.create({
        symbols: [
          { name: "valve", data: SYMBOL_DATA, group: g.key },
          { name: "pump", data: SYMBOL_DATA, group: g.key },
        ],
      });
      const outsider = await client.schematics.symbols.create({
        name: "valve",
        data: SYMBOL_DATA,
        group: (await createSymbolGroup()).key,
      });
      const retrieved = await client.schematics.symbols.retrieve({
        groups: [g.key],
        key: [kept.key, outsider.key],
      });
      expect(retrieved.map((s) => s.key)).toEqual([kept.key]);
    });

    it("should return an empty list for an empty group", async () => {
      const g = await createSymbolGroup();
      expect(await client.schematics.symbols.retrieve({ groups: [g.key] })).toEqual([]);
    });

    it("should deliver newly created members to group subscribers", async () => {
      const g = await createSymbolGroup();
      const list = client.schematics.symbols.watch({ groups: [g.key] });
      await client.schematics.symbols.create({
        name: "first",
        data: SYMBOL_DATA,
        group: g.key,
      });
      const answers: schematic.symbol.Symbol[][] = [];
      const stop = list.onChange((w) => {
        if (w != null) answers.push(w.records);
      });
      try {
        await list.retrieve();
        await client.schematics.symbols.create({
          name: "second",
          data: SYMBOL_DATA,
          group: g.key,
        });
        await expect
          .poll(() =>
            answers
              .at(-1)
              ?.map((s) => s.name)
              .sort(),
          )
          .toEqual(["first", "second"]);
      } finally {
        stop();
      }
    });

    it("should drop deleted members from group subscribers", async () => {
      const g = await createSymbolGroup();
      const list = client.schematics.symbols.watch({ groups: [g.key] });
      const created = await client.schematics.symbols.create({
        symbols: [
          { name: "keep", data: SYMBOL_DATA, group: g.key },
          { name: "drop", data: SYMBOL_DATA, group: g.key },
        ],
      });
      const stop = list.onChange(() => {});
      try {
        await list.retrieve();
        const doomed = created.find(
          (s) => s.name === "drop",
        ) as schematic.symbol.Symbol;
        await client.schematics.symbols.delete(doomed.key);
        await expect
          .poll(() => list.getCached()?.records.map((s) => s.name))
          .toEqual(["keep"]);
      } finally {
        stop();
      }
    });

    it("should retrieve only symbols in no group when grouped is false", async () => {
      const loose = await client.schematics.symbols.create({
        name: `loose-${id.create()}`,
        data: SYMBOL_DATA,
      });
      const ungrouped = await client.schematics.symbols.retrieve({ grouped: false });
      expect(ungrouped.map((s) => s.key)).toContain(loose.key);
      expect(ungrouped.every((s) => s.group == null)).toBe(true);
    });
  });

  describe("setGroup", () => {
    it("should move symbols between groups and out of every group", async () => {
      const from = await createSymbolGroup();
      const to = await createSymbolGroup();
      const symbol = await client.schematics.symbols.create({
        name: "Mover",
        data: SYMBOL_DATA,
        group: from.key,
      });
      await client.batch.send(
        client.schematics.symbols.setGroupRequest(symbol.key, to.key),
      );
      expect((await client.schematics.symbols.retrieve(symbol.key)).group).toEqual(
        to.key,
      );
      expect(
        (await client.schematics.symbols.retrieve({ groups: [to.key] })).map(
          (s) => s.key,
        ),
      ).toEqual([symbol.key]);
      await client.batch.send(
        client.schematics.symbols.setGroupRequest(symbol.key, null),
      );
      expect(
        (await client.schematics.symbols.retrieve(symbol.key)).group,
      ).toBeUndefined();
    });
  });

  describe("rename", () => {
    it("should rename a symbol", async () => {
      const symbol = await client.schematics.symbols.create({
        name: "Original Name",
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        group: symbolGroup.key,
      });

      await client.schematics.symbols.rename(symbol.key, "New Name");

      const retrieved = await client.schematics.symbols.retrieve(symbol.key);
      expect(retrieved.name).toBe("New Name");
    });
  });

  describe("dispatch", () => {
    it("should apply the action to the symbol", async () => {
      const symbol = await client.schematics.symbols.create({
        name: "Dispatch Test",
        data: SYMBOL_DATA,
        group: symbolGroup.key,
      });
      const svg = "<svg><g></g></svg>";
      await client.schematics.symbols.dispatch(
        symbol.key,
        schematic.symbol.set({ path: ["data", "svg"], value: svg }),
      );
      const stored = await createTestClient().schematics.symbols.retrieve(symbol.key);
      expect(stored.data.svg).toBe(svg);
    });
  });

  describe("delete", () => {
    it("should delete a single symbol", async () => {
      const symbol = await client.schematics.symbols.create({
        name: "Delete Test",
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        group: symbolGroup.key,
      });

      await client.schematics.symbols.delete(symbol.key);

      await expect(client.schematics.symbols.retrieve(symbol.key)).rejects.toThrow();
    });
  });

  describe("exportGroup", () => {
    // Zip entry names are stored uncompressed, so the raw archive names the bundle's
    // files without the spec needing a zip reader.
    const download = async (key: group.Key): Promise<string> => {
      const stream = await client.schematics.symbols.exportGroup(key, {
        encoding: "JSON",
      });
      return new TextDecoder().decode(await new Response(stream).arrayBuffer());
    };

    it("should export the group's symbols as a zip bundle", async () => {
      const exported = await createSymbolGroup();
      await client.schematics.symbols.create({
        name: "Inlet",
        data: SYMBOL_DATA,
        group: exported.key,
      });
      const archive = await download(exported.key);
      expect(archive.startsWith("PK")).toBe(true);
      expect(archive).toContain("manifest.json");
      expect(archive).toContain("Inlet.json");
    });

    it("should skip a nested group", async () => {
      const exported = await createSymbolGroup();
      await client.groups.create({
        parent: group.resourceID(exported.key),
        name: "Nested",
      });
      await client.schematics.symbols.create({
        name: "Inlet",
        data: SYMBOL_DATA,
        group: exported.key,
      });
      const archive = await download(exported.key);
      expect(archive).toContain("manifest.json");
      expect(archive).toContain("Inlet.json");
      expect(archive).not.toContain("Nested");
    });
  });

  describe("importGroup", () => {
    it("should import an exported bundle back as a fresh group", async () => {
      const exported = await createSymbolGroup();
      await client.schematics.symbols.create({
        name: "Inlet",
        data: SYMBOL_DATA,
        group: exported.key,
      });
      const stream = await client.schematics.symbols.exportGroup(exported.key, {
        encoding: "JSON",
      });
      const bundle = new Uint8Array(await new Response(stream).arrayBuffer());
      const imported = await client.schematics.symbols.importGroup(bundle);
      expect(imported.key).not.toEqual(exported.key);
      expect(imported.name).toEqual(exported.name);
      const symbols = await client.schematics.symbols.retrieve({
        groups: [imported.key],
      });
      expect(symbols).toHaveLength(1);
      expect(symbols[0].name).toEqual("Inlet");
    });

    it("should import the group as a top-level symbol group", async () => {
      const exported = await createSymbolGroup();
      const stream = await client.schematics.symbols.exportGroup(exported.key, {
        encoding: "JSON",
      });
      const bundle = new Uint8Array(await new Response(stream).arrayBuffer());
      const imported = await client.schematics.symbols.importGroup(bundle);
      expect(imported.scope).toEqual("schematic_symbol");
      expect(imported.parent).toBeUndefined();
    });

    it("should reject a bundle without a manifest", async () => {
      // The 22-byte end-of-central-directory record alone: a valid, empty archive.
      const emptyArchive = new Uint8Array(22);
      emptyArchive.set([0x50, 0x4b, 0x05, 0x06]);
      await expect(client.schematics.symbols.importGroup(emptyArchive)).rejects.toThrow(
        "bundle holds no manifest.json",
      );
    });
  });

  describe("deleteGroup", () => {
    it("should delete the group's symbols", async () => {
      const target = await createSymbolGroup();
      const symbols = await client.schematics.symbols.create({
        symbols: [
          { name: "Inlet", data: SYMBOL_DATA, group: target.key },
          { name: "Outlet", data: SYMBOL_DATA, group: target.key },
        ],
      });
      await client.schematics.symbols.deleteGroup(target.key);
      await expect(client.schematics.symbols.retrieve(symbols[0].key)).rejects.toThrow(
        NotFoundError,
      );
      await expect(client.schematics.symbols.retrieve(symbols[1].key)).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should drop the deleted group from the group cache", async () => {
      const target = await createSymbolGroup();
      // Warm the cache, so a record the delete leaves behind answers the next read.
      expect((await client.groups.retrieve({ key: target.key })).key).toEqual(
        target.key,
      );
      await client.schematics.symbols.deleteGroup(target.key);
      await expect(client.groups.retrieve({ key: target.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should drop the group from its parent's children", async () => {
      const parent = await createSymbolGroup();
      const list = client.groups.watch({ parents: [group.resourceID(parent.key)] });
      const target = await createSymbolGroup(group.resourceID(parent.key));
      const stop = list.onChange(() => {});
      try {
        expect((await list.retrieve()).records).toHaveLength(1);
        await client.schematics.symbols.deleteGroup(target.key);
        await expect.poll(() => list.getCached()?.records).toEqual([]);
      } finally {
        stop();
      }
    });

    it("should drop the deleted symbols from group subscribers", async () => {
      const target = await createSymbolGroup();
      const list = client.schematics.symbols.watch({ groups: [target.key] });
      await client.schematics.symbols.create({
        name: "Inlet",
        data: SYMBOL_DATA,
        group: target.key,
      });
      const stop = list.onChange(() => {});
      try {
        await list.retrieve();
        await client.schematics.symbols.deleteGroup(target.key);
        await expect.poll(() => list.getCached()?.records).toEqual([]);
      } finally {
        stop();
      }
    });

    describe("optimistic", () => {
      const WRITE_FAILED = new Error("write failed");
      const fail = () => {
        throw WRITE_FAILED;
      };

      const createPopulated = async (parent?: resource.ID) => {
        const target = await createSymbolGroup(parent);
        const symbol = await client.schematics.symbols.create({
          name: "Inlet",
          data: SYMBOL_DATA,
          group: target.key,
        });
        return { target, symbol };
      };

      it("should drop the group and its symbols before the write commits", async () => {
        const { target, symbol } = await createPopulated();
        let group_: query.Cached<group.Group> | undefined;
        let symbol_: query.Cached<schematic.symbol.Symbol> | undefined;
        await client.schematics.symbols.deleteGroup(target.key, {
          onOptimistic: () => {
            group_ = client.groups.watch({ key: target.key }).getCached();
            symbol_ = client.schematics.symbols.watch(symbol.key).getCached();
          },
        });
        expect(query.isLive(group_)).toBe(false);
        expect(query.isLive(symbol_)).toBe(false);
      });

      it("should restore the group and its symbols when the write fails", async () => {
        const { target, symbol } = await createPopulated();
        await expect(
          client.schematics.symbols.deleteGroup(target.key, { onOptimistic: fail }),
        ).rejects.toBe(WRITE_FAILED);
        expect(
          expectLive(client.groups.watch({ key: target.key }).getCached()).name,
        ).toEqual(target.name);
        expect(
          expectLive(client.schematics.symbols.watch(symbol.key).getCached()).name,
        ).toEqual(symbol.name);
      });

      it("should restore the parent's children when the write fails", async () => {
        const parent = await createSymbolGroup();
        const list = client.groups.watch({ parents: [group.resourceID(parent.key)] });
        const { target } = await createPopulated(group.resourceID(parent.key));
        const stop = list.onChange(() => {});
        try {
          expect((await list.retrieve()).records).toHaveLength(1);
          await expect(
            client.schematics.symbols.deleteGroup(target.key, { onOptimistic: fail }),
          ).rejects.toBe(WRITE_FAILED);
          expect(list.getCached()?.records.map((g) => g.key)).toEqual([target.key]);
        } finally {
          stop();
        }
      });
    });
  });
});
