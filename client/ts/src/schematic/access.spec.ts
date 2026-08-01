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
import { schematic } from "@/schematic";
import { createTestClient, createTestClientWithPolicy } from "@/testutil";

const client = createTestClient();

describe("schematic", () => {
  describe("access control", () => {
    it("should deny access when no retrieve policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const proj = await client.projects.create({
        name: "test",
        layout: {},
      });
      const randomSchematic = await client.schematics.create(proj.key, {
        name: "test",
      });
      await expect(userClient.schematics.retrieve(randomSchematic.key)).rejects.toThrow(
        AuthError,
      );
    });

    it("should allow the caller to retrieve schematics with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [schematic.ontologyID("")],
        actions: ["retrieve"],
      });
      const proj = await client.projects.create({
        name: "test",
        layout: {},
      });
      const randomSchematic = await client.schematics.create(proj.key, {
        name: "test",
      });
      const retrieved = await userClient.schematics.retrieve(randomSchematic.key);
      expect(retrieved.key).toBe(randomSchematic.key);
      expect(retrieved.name).toBe(randomSchematic.name);
    });

    it("should allow the caller to create schematics with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [schematic.ontologyID("")],
        actions: ["create"],
      });
      const proj = await client.projects.create({
        name: "test",
        layout: {},
      });
      await userClient.schematics.create(proj.key, {
        name: "test",
      });
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [schematic.ontologyID("")],
        actions: [],
      });
      const proj = await client.projects.create({
        name: "test",
        layout: {},
      });
      await expect(
        userClient.schematics.create(proj.key, {
          name: "test",
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete schematics with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [schematic.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const proj = await client.projects.create({
        name: "test",
        layout: {},
      });
      const randomSchematic = await client.schematics.create(proj.key, {
        name: "test",
      });
      await userClient.schematics.delete(randomSchematic.key);
      await expect(userClient.schematics.retrieve(randomSchematic.key)).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [schematic.ontologyID("")],
        actions: [],
      });
      const proj = await client.projects.create({
        name: "test",
        layout: {},
      });
      const randomSchematic = await client.schematics.create(proj.key, {
        name: "test",
      });
      await expect(userClient.schematics.delete(randomSchematic.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
