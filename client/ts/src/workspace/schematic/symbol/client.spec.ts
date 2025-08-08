// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { newTestClient } from "@/testutil/client";

const client = newTestClient();

describe("Symbol Client", () => {
  describe("create", () => {
    it("should create a single symbol", async () => {
      const symbol = await client.workspaces.schematic.symbols.create({
        name: "Test Symbol",
        data: {
          svg: "<svg></svg>",
          states: [],
        },
      });
      expect(symbol.name).toBe("Test Symbol");
      expect(symbol.key).toBeDefined();
    });

    it("should create multiple symbols", async () => {
      const symbols = await client.workspaces.schematic.symbols.create([
        {
          name: "Symbol 1",
          data: { svg: "<svg></svg>", states: [] },
        },
        {
          name: "Symbol 2",
          data: { svg: "<svg></svg>", states: [] },
        },
      ]);
      expect(symbols).toHaveLength(2);
      expect(symbols[0].name).toBe("Symbol 1");
      expect(symbols[1].name).toBe("Symbol 2");
    });
  });

  describe("retrieve", () => {
    it("should retrieve a single symbol by key", async () => {
      const created = await client.workspaces.schematic.symbols.create({
        name: "Retrieve Test",
        data: { svg: "<svg></svg>", states: [] },
      });

      const retrieved = await client.workspaces.schematic.symbols.retrieve({
        key: created.key,
      });
      expect(retrieved.key).toBe(created.key);
      expect(retrieved.name).toBe("Retrieve Test");
    });

    it("should retrieve multiple symbols by keys", async () => {
      const created1 = await client.workspaces.schematic.symbols.create({
        name: "Multi Test 1",
        data: { svg: "<svg></svg>", states: [] },
      });
      const created2 = await client.workspaces.schematic.symbols.create({
        name: "Multi Test 2",
        data: { svg: "<svg></svg>", states: [] },
      });

      const retrieved = await client.workspaces.schematic.symbols.retrieve({
        keys: [created1.key, created2.key],
      });
      expect(retrieved).toHaveLength(2);
    });
  });

  describe("rename", () => {
    it("should rename a symbol", async () => {
      const symbol = await client.workspaces.schematic.symbols.create({
        name: "Original Name",
        data: { svg: "<svg></svg>", states: [] },
      });

      await client.workspaces.schematic.symbols.rename(symbol.key, "New Name");

      const retrieved = await client.workspaces.schematic.symbols.retrieve({
        key: symbol.key,
      });
      expect(retrieved.name).toBe("New Name");
    });
  });

  describe("delete", () => {
    it("should delete a single symbol", async () => {
      const symbol = await client.workspaces.schematic.symbols.create({
        name: "Delete Test",
        data: { svg: "<svg></svg>", states: [] },
      });

      await client.workspaces.schematic.symbols.delete(symbol.key);

      await expect(
        client.workspaces.schematic.symbols.retrieve({ key: symbol.key }),
      ).rejects.toThrow();
    });
  });
});
