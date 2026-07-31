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
import { ontology } from "@/ontology";
import { query } from "@/query";
import { type schematic } from "@/schematic";
import { createTestClient } from "@/testutil";

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

      const retrieved = await client.schematics.symbols.retrieve({
        key: created.key,
      });
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

      const retrieved = await client.schematics.symbols.retrieve({
        key: symbol.key,
      });
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

      await expect(
        client.schematics.symbols.retrieve({ key: symbol.key }),
      ).rejects.toThrow();
    });
  });

  describe("retrieveGroup", () => {
    it("should retrieve the symbol group", async () => {
      const group = await client.schematics.symbols.retrieveGroup();
      expect(group.key).toBeDefined();
      expect(group.name).toBe("Schematic Symbols");
    });
  });
});
