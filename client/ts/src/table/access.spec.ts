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
import { table } from "@/table";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("table", () => {
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
      const randomTable = await client.tables.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(
        userClient.tables.retrieve({ key: randomTable.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve tables with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [table.ontologyID("")],
        actions: ["retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomTable = await client.tables.create(ws.key, {
        name: "test",
        data: {},
      });
      const retrieved = await userClient.tables.retrieve({
        key: randomTable.key,
      });
      expect(retrieved.key).toBe(randomTable.key);
      expect(retrieved.name).toBe(randomTable.name);
    });

    it("should allow the caller to create tables with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [table.ontologyID("")],
        actions: ["create"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await userClient.tables.create(ws.key, {
        name: "test",
        data: {},
      });
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [table.ontologyID("")],
        actions: [],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      await expect(
        userClient.tables.create(ws.key, {
          name: "test",
          data: {},
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete tables with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [table.ontologyID("")],
        actions: ["delete", "retrieve"],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomTable = await client.tables.create(ws.key, {
        name: "test",
        data: {},
      });
      await userClient.tables.delete(randomTable.key);
      await expect(
        userClient.tables.retrieve({ key: randomTable.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [table.ontologyID("")],
        actions: [],
      });
      const ws = await client.workspaces.create({
        name: "test",
        layout: {},
      });
      const randomTable = await client.tables.create(ws.key, {
        name: "test",
        data: {},
      });
      await expect(userClient.tables.delete(randomTable.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
