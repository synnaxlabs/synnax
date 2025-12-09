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
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";
import { schematic } from "@/workspace/schematic";

const client = createTestClient();

describe("schematic", () => {
  describe("access control", () => {
    it("should deny access when no retrieve policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomSchematic = await client.workspaces.schematics.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(
        userClient.workspaces.schematics.retrieve({ key: randomSchematic.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve schematics with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [schematic.ontologyID("")],
        actions: ["retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomSchematic = await client.workspaces.schematics.create(ws.key, {
        name: "test",
        data: {},
      });
      const retrieved = await userClient.workspaces.schematics.retrieve({
        key: randomSchematic.key,
      });
      expect(retrieved.key).toBe(randomSchematic.key);
      expect(retrieved.name).toBe(randomSchematic.name);
    });

    it("should allow the caller to create schematics with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [schematic.ontologyID("")],
        actions: ["create"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await userClient.workspaces.schematics.create(ws.key, {
        name: "test",
        data: {},
      });
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [schematic.ontologyID("")],
        actions: [],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await expect(
        userClient.workspaces.schematics.create(ws.key, {
          name: "test",
          data: {},
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete schematics with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [schematic.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomSchematic = await client.workspaces.schematics.create(ws.key, {
        name: "test",
        data: {},
      });
      await userClient.workspaces.schematics.delete(randomSchematic.key);
      await expect(
        userClient.workspaces.schematics.retrieve({ key: randomSchematic.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [schematic.ontologyID("")],
        actions: [],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomSchematic = await client.workspaces.schematics.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(
        userClient.workspaces.schematics.delete(randomSchematic.key),
      ).rejects.toThrow(AuthError);
    });
  });
});
