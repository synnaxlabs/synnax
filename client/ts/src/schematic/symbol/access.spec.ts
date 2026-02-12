// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { AuthError, NotFoundError } from "@/errors";
import { group } from "@/group";
import { ontology } from "@/ontology";
import { group } from "@/ontology/group";
import { symbol } from "@/schematic/symbol";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("schematic_symbol", () => {
  describe("access control", () => {
    it("should deny access when no retrieve policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const symbolGroup = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      const randomSymbol = await client.schematics.symbols.create({
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
        userClient.schematics.symbols.retrieve({ key: randomSymbol.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve symbols with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [symbol.ontologyID("")],
        actions: ["retrieve"],
      });
      const symbolGroup = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      const randomSymbol = await client.schematics.symbols.create({
        name: "test",
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "sensor",
        },
        parent: group.ontologyID(symbolGroup.key),
      });
      const retrieved = await userClient.schematics.symbols.retrieve({
        key: randomSymbol.key,
      });
      expect(retrieved.key).toBe(randomSymbol.key);
      expect(retrieved.name).toBe(randomSymbol.name);
    });

    it("should allow the caller to create symbols with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [symbol.ontologyID("")],
        actions: ["create"],
      });
      const symbolGroup = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      await userClient.schematics.symbols.create({
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

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [symbol.ontologyID("")],
        actions: [],
      });
      const symbolGroup = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      await expect(
        userClient.schematics.symbols.create({
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
        objects: [symbol.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const symbolGroup = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      const randomSymbol = await client.schematics.symbols.create({
        name: "test",
        data: {
          svg: "<svg></svg>",
          states: [],
          handles: [],
          variant: "sensor",
        },
        parent: group.ontologyID(symbolGroup.key),
      });
      await userClient.schematics.symbols.delete(randomSymbol.key);
      await expect(
        userClient.schematics.symbols.retrieve({ key: randomSymbol.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [symbol.ontologyID("")],
        actions: [],
      });
      const symbolGroup = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "Test Symbols",
      });
      const randomSymbol = await client.schematics.symbols.create({
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
        userClient.schematics.symbols.delete(randomSymbol.key),
      ).rejects.toThrow(AuthError);
    });
  });
});
