// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { AuthError, NotFoundError } from "@/errors";
import { ontology } from "@/ontology";
import { group } from "@/ontology/group";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";
import { symbol } from "@/workspace/schematic/symbol";

const client = createTestClient();

describe("schematic_symbol", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve symbols with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const symbolGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      const randomSymbol = await client.workspaces.schematics.symbols.create({
        name: "test",
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "sensor",
        },
        parent: group.ontologyID(symbolGroup.key),
      });
      await expect(
        userClient.workspaces.schematics.symbols.retrieve({ key: randomSymbol.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve symbols with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [symbol.ontologyID("")],
        actions: ["retrieve"],
      });
      const symbolGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      const randomSymbol = await client.workspaces.schematics.symbols.create({
        name: "test",
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "sensor",
        },
        parent: group.ontologyID(symbolGroup.key),
      });
      const retrieved = await userClient.workspaces.schematics.symbols.retrieve({
        key: randomSymbol.key,
      });
      expect(retrieved.key).toBe(randomSymbol.key);
      expect(retrieved.name).toBe(randomSymbol.name);
    });

    it("should allow the caller to create symbols with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [symbol.ontologyID("")],
        actions: ["create"],
      });
      const symbolGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      await userClient.workspaces.schematics.symbols.create({
        name: "test",
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "sensor",
        },
        parent: group.ontologyID(symbolGroup.key),
      });
    });

    it("should prevent the caller to create symbols with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [symbol.ontologyID("")],
        actions: ["create"],
      });
      const symbolGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      await expect(
        userClient.workspaces.schematics.symbols.create({
          name: "test",
          data: {
            svg: "<svg></svg>",
            states: [],
            handles: [],
            variant: "sensor",
          },
          parent: group.ontologyID(symbolGroup.key),
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete symbols with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [symbol.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const symbolGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      const randomSymbol = await client.workspaces.schematics.symbols.create({
        name: "test",
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "sensor",
        },
        parent: group.ontologyID(symbolGroup.key),
      });
      await userClient.workspaces.schematics.symbols.delete(randomSymbol.key);
      await expect(
        userClient.workspaces.schematics.symbols.retrieve({ key: randomSymbol.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should prevent the caller to delete symbols with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [symbol.ontologyID("")],
        actions: ["delete"],
      });
      const symbolGroup = await client.ontology.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      const randomSymbol = await client.workspaces.schematics.symbols.create({
        name: "test",
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "sensor",
        },
        parent: group.ontologyID(symbolGroup.key),
      });
      await expect(
        userClient.workspaces.schematics.symbols.delete(randomSymbol.key),
      ).rejects.toThrow(AuthError);
    });
  });
});
