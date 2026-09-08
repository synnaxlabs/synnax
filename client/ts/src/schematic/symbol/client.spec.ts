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

import { NotFoundError } from "@/errors";
import { group } from "@/group";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { type schematic } from "@/schematic";
import { createTestClient, expectLive } from "@/testutil";

const client = createTestClient();

const SYMBOL_DATA: schematic.symbol.New["data"] = {
  svg: "<svg></svg>",
  states: [],
  handles: [],
  variant: "sensor",
};

const createParent = async (): Promise<ontology.ID> => {
  const parent = await client.groups.create({
    parent: ontology.ROOT_ID,
    name: `symbol-parent-${id.create()}`,
  });
  return group.ontologyID(parent.key);
};

describe("Symbol Client", () => {
  let symbolGroup: group.Group;
  beforeAll(async () => {
    symbolGroup = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: "Test Symbols",
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
        parent: group.ontologyID(symbolGroup.key),
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
          },
          {
            name: "Symbol 2",
            data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
          },
        ],
        parent: group.ontologyID(symbolGroup.key),
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
        parent: group.ontologyID(symbolGroup.key),
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
          },
          {
            name: "Multi Test 2",
            data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
          },
        ],
        parent: group.ontologyID(symbolGroup.key),
      });

      const retrieved = await client.schematics.symbols.retrieve({
        keys: created.map((s) => s.key),
      });
      expect(retrieved).toHaveLength(2);
    });

    it("should retrieve symbols by search term", async () => {
      const prefix = `searchable-symbol-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.schematics.symbols.create({
        symbols: names.map((name) => ({
          name,
          data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        })),
        parent: group.ontologyID(symbolGroup.key),
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
  });

  describe("parent-scoped retrieve", () => {
    it("should retrieve only the symbols under the given parent", async () => {
      const parent = await createParent();
      const other = await createParent();
      const created = await client.schematics.symbols.create({
        symbols: [
          { name: "child-1", data: SYMBOL_DATA },
          { name: "child-2", data: SYMBOL_DATA },
        ],
        parent,
      });
      await client.schematics.symbols.create({
        name: "outsider",
        data: SYMBOL_DATA,
        parent: other,
      });

      const retrieved = await client.schematics.symbols.retrieve({ parent });
      expect(retrieved.map((s) => s.key).sort()).toEqual(
        created.map((s) => s.key).sort(),
      );
    });

    it("should combine a parent scope with a search term", async () => {
      const parent = await createParent();
      const prefix = `scoped-${id.create()}`;
      await client.schematics.symbols.create({
        symbols: [
          { name: `${prefix}-valve`, data: SYMBOL_DATA },
          { name: "pump", data: SYMBOL_DATA },
        ],
        parent,
      });
      await expect
        .poll(async () => {
          const results = await client.schematics.symbols.retrieve({
            parent,
            searchTerm: prefix,
          });
          return results.map((s) => s.name);
        })
        .toEqual([`${prefix}-valve`]);
    });

    it("should return an empty list for a childless parent", async () => {
      const parent = await createParent();
      expect(await client.schematics.symbols.retrieve({ parent })).toEqual([]);
    });

    it("should deliver newly created children to parent subscribers", async () => {
      const parent = await createParent();
      await client.schematics.symbols.create({
        name: "first",
        data: SYMBOL_DATA,
        parent,
      });
      const answers: schematic.symbol.Symbol[][] = [];
      const stop = client.schematics.symbols.onChange({ parent }, (cached) => {
        if (query.isLive(cached)) answers.push(cached);
      });
      try {
        await client.schematics.symbols.retrieve({ parent });
        await client.schematics.symbols.create({
          name: "second",
          data: SYMBOL_DATA,
          parent,
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

    it("should drop deleted children from parent subscribers", async () => {
      const parent = await createParent();
      const created = await client.schematics.symbols.create({
        symbols: [
          { name: "keep", data: SYMBOL_DATA },
          { name: "drop", data: SYMBOL_DATA },
        ],
        parent,
      });
      let latest: query.Cached<schematic.symbol.Symbol[]> | undefined;
      const stop = client.schematics.symbols.onChange({ parent }, (cached) => {
        latest = cached;
      });
      try {
        await client.schematics.symbols.retrieve({ parent });
        const doomed = created.find(
          (s) => s.name === "drop",
        ) as schematic.symbol.Symbol;
        await client.schematics.symbols.delete(doomed.key);
        await expect
          .poll(() => (query.isLive(latest) ? latest.map((s) => s.name) : undefined))
          .toEqual(["keep"]);
      } finally {
        stop();
      }
    });
  });

  describe("rename", () => {
    it("should rename a symbol", async () => {
      const symbol = await client.schematics.symbols.create({
        name: "Original Name",
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        parent: group.ontologyID(symbolGroup.key),
      });

      await client.schematics.symbols.rename(symbol.key, "New Name");

      const retrieved = await client.schematics.symbols.retrieve(symbol.key);
      expect(retrieved.name).toBe("New Name");
    });
  });

  describe("delete", () => {
    it("should delete a single symbol", async () => {
      const symbol = await client.schematics.symbols.create({
        name: "Delete Test",
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        parent: group.ontologyID(symbolGroup.key),
      });

      await client.schematics.symbols.delete(symbol.key);

      await expect(client.schematics.symbols.retrieve(symbol.key)).rejects.toThrow();
    });
  });

  describe("retrieveGroup", () => {
    it("should retrieve the symbol group", async () => {
      const group = await client.schematics.symbols.retrieveGroup();
      expect(group.key).toBeDefined();
      expect(group.name).toBe("Schematic Symbols");
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
      const exported = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `symbol-export-${id.create()}`,
      });
      await client.schematics.symbols.create({
        name: "Inlet",
        data: SYMBOL_DATA,
        parent: group.ontologyID(exported.key),
      });
      const archive = await download(exported.key);
      expect(archive.startsWith("PK")).toBe(true);
      expect(archive).toContain("manifest.json");
      expect(archive).toContain("Inlet.json");
    });

    it("should skip a resource that is not a symbol", async () => {
      const exported = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `symbol-export-${id.create()}`,
      });
      const parent = group.ontologyID(exported.key);
      await client.groups.create({ parent, name: "Nested" });
      await client.schematics.symbols.create({
        name: "Inlet",
        data: SYMBOL_DATA,
        parent,
      });
      const archive = await download(exported.key);
      expect(archive).toContain("manifest.json");
      expect(archive).toContain("Inlet.json");
      expect(archive).not.toContain("Nested");
    });
  });

  describe("importGroup", () => {
    it("should import an exported bundle back as a fresh group", async () => {
      const exported = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `symbol-import-${id.create()}`,
      });
      await client.schematics.symbols.create({
        name: "Inlet",
        data: SYMBOL_DATA,
        parent: group.ontologyID(exported.key),
      });
      const stream = await client.schematics.symbols.exportGroup(exported.key, {
        encoding: "JSON",
      });
      const bundle = new Uint8Array(await new Response(stream).arrayBuffer());
      const imported = await client.schematics.symbols.importGroup(bundle);
      expect(imported.key).not.toEqual(exported.key);
      expect(imported.name).toEqual(exported.name);
      const symbols = await client.schematics.symbols.retrieve({
        parent: group.ontologyID(imported.key),
      });
      expect(symbols).toHaveLength(1);
      expect(symbols[0].name).toEqual("Inlet");
    });

    it("should create the imported group under the permanent group", async () => {
      const exported = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `symbol-import-${id.create()}`,
      });
      const stream = await client.schematics.symbols.exportGroup(exported.key, {
        encoding: "JSON",
      });
      const bundle = new Uint8Array(await new Response(stream).arrayBuffer());
      const imported = await client.schematics.symbols.importGroup(bundle);
      const permanent = await client.schematics.symbols.retrieveGroup();
      const children = await client.groups.retrieve({
        parent: group.ontologyID(permanent.key),
      });
      expect(children.map(({ key }) => key)).toContain(imported.key);
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
    const createTarget = async (parent: ontology.ID = ontology.ROOT_ID) =>
      await client.groups.create({ parent, name: `symbol-delete-${id.create()}` });

    it("should delete the group's symbols", async () => {
      const target = await createTarget();
      const symbols = await client.schematics.symbols.create({
        symbols: [
          { name: "Inlet", data: SYMBOL_DATA },
          { name: "Outlet", data: SYMBOL_DATA },
        ],
        parent: group.ontologyID(target.key),
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
      const target = await createTarget();
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
      const parent = await createParent();
      const target = await createTarget(parent);
      let latest: query.Cached<group.Group[]> | undefined;
      const stop = client.groups.onChange({ parent }, (cached) => {
        latest = cached;
      });
      try {
        expect(await client.groups.retrieve({ parent })).toHaveLength(1);
        await client.schematics.symbols.deleteGroup(target.key);
        await expect
          .poll(() => (query.isLive(latest) ? latest : undefined))
          .toEqual([]);
      } finally {
        stop();
      }
    });

    it("should drop the deleted symbols from parent subscribers", async () => {
      const target = await createTarget();
      const parent = group.ontologyID(target.key);
      await client.schematics.symbols.create({
        name: "Inlet",
        data: SYMBOL_DATA,
        parent,
      });
      let latest: query.Cached<schematic.symbol.Symbol[]> | undefined;
      const stop = client.schematics.symbols.onChange({ parent }, (cached) => {
        latest = cached;
      });
      try {
        await client.schematics.symbols.retrieve({ parent });
        await client.schematics.symbols.deleteGroup(target.key);
        await expect
          .poll(() => (query.isLive(latest) ? latest : undefined))
          .toEqual([]);
      } finally {
        stop();
      }
    });

    describe("optimistic", () => {
      const WRITE_FAILED = new Error("write failed");
      const fail = () => {
        throw WRITE_FAILED;
      };

      const createPopulated = async (parent: ontology.ID = ontology.ROOT_ID) => {
        const target = await createTarget(parent);
        const symbol = await client.schematics.symbols.create({
          name: "Inlet",
          data: SYMBOL_DATA,
          parent: group.ontologyID(target.key),
        });
        return { target, symbol };
      };

      it("should drop the group and its symbols before the write commits", async () => {
        const { target, symbol } = await createPopulated();
        let group_: query.Cached<group.Group> | undefined;
        let symbol_: query.Cached<schematic.symbol.Symbol> | undefined;
        await client.schematics.symbols.deleteGroup(target.key, {
          onOptimistic: () => {
            group_ = client.groups.getCached({ key: target.key });
            symbol_ = client.schematics.symbols.getCached(symbol.key);
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
        expect(expectLive(client.groups.getCached({ key: target.key })).name).toEqual(
          target.name,
        );
        expect(
          expectLive(client.schematics.symbols.getCached(symbol.key)).name,
        ).toEqual(symbol.name);
      });

      it("should restore the parent's children when the write fails", async () => {
        const parent = await createParent();
        const { target } = await createPopulated(parent);
        let latest: query.Cached<group.Group[]> | undefined;
        const stop = client.groups.onChange({ parent }, (cached) => {
          latest = cached;
        });
        try {
          expect(await client.groups.retrieve({ parent })).toHaveLength(1);
          await expect(
            client.schematics.symbols.deleteGroup(target.key, { onOptimistic: fail }),
          ).rejects.toBe(WRITE_FAILED);
          expect(expectLive(latest).map((g) => g.key)).toEqual([target.key]);
        } finally {
          stop();
        }
      });
    });

    it("should skip a resource that is not a symbol", async () => {
      const target = await createTarget();
      const parent = group.ontologyID(target.key);
      const created = await client.schematics.symbols.create({
        name: "Inlet",
        data: SYMBOL_DATA,
        parent,
      });
      const stray = await client.groups.create({ parent, name: "Nested" });
      await client.schematics.symbols.deleteGroup(target.key);
      await expect(client.schematics.symbols.retrieve(created.key)).rejects.toThrow(
        NotFoundError,
      );
      // The stray survives the delete under the permanent symbol group.
      const root = await client.schematics.symbols.retrieveGroup();
      const parents = await client.ontology.retrieveParents(
        group.ontologyID(stray.key),
      );
      expect(parents.map((p) => p.id.key)).toEqual([root.key]);
    });
  });
});
