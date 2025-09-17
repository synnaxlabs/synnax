// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, describe, expect, it } from "vitest";

import { ontology } from "@/ontology";
import { group } from "@/ontology/group";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Symbol Client", () => {
  let symbolGroup: group.Group;
  beforeAll(async () => {
    symbolGroup = await client.ontology.groups.create({
      parent: ontology.ROOT_ID,
      name: "Test Symbols",
    });
  });
  describe("create", () => {
    it("should create a single symbol", async () => {
      const symbol = await client.workspaces.schematics.symbols.create({
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
      const symbols = await client.workspaces.schematics.symbols.create({
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
      const created = await client.workspaces.schematics.symbols.create({
        name: "Retrieve Test",
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        parent: group.ontologyID(symbolGroup.key),
      });

      const retrieved = await client.workspaces.schematics.symbols.retrieve({
        key: created.key,
      });
      expect(retrieved.key).toBe(created.key);
      expect(retrieved.name).toBe("Retrieve Test");
    });

    it("should retrieve multiple symbols by keys", async () => {
      const created1 = await client.workspaces.schematics.symbols.create({
        name: "Multi Test 1",
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        parent: group.ontologyID(symbolGroup.key),
      });
      const created2 = await client.workspaces.schematics.symbols.create({
        name: "Multi Test 2",
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        parent: group.ontologyID(symbolGroup.key),
      });

      const retrieved = await client.workspaces.schematics.symbols.retrieve({
        keys: [created1.key, created2.key],
      });
      expect(retrieved).toHaveLength(2);
    });
  });

  describe("rename", () => {
    it("should rename a symbol", async () => {
      const symbol = await client.workspaces.schematics.symbols.create({
        name: "Original Name",
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        parent: group.ontologyID(symbolGroup.key),
      });

      await client.workspaces.schematics.symbols.rename(symbol.key, "New Name");

      const retrieved = await client.workspaces.schematics.symbols.retrieve({
        key: symbol.key,
      });
      expect(retrieved.name).toBe("New Name");
    });
  });

  describe("delete", () => {
    it("should delete a single symbol", async () => {
      const symbol = await client.workspaces.schematics.symbols.create({
        name: "Delete Test",
        data: { svg: "<svg></svg>", states: [], handles: [], variant: "sensor" },
        parent: group.ontologyID(symbolGroup.key),
      });

      await client.workspaces.schematics.symbols.delete(symbol.key);

      await expect(
        client.workspaces.schematics.symbols.retrieve({ key: symbol.key }),
      ).rejects.toThrow();
    });
  });

  describe("retrieveGroup", () => {
    it("should retrieve the symbol group", async () => {
      const group = await client.workspaces.schematics.symbols.retrieveGroup();
      expect(group.key).toBeDefined();
      expect(group.name).toBe("Schematic Symbols");
    });
  });
});
