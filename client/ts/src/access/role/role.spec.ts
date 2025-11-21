// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil/client";
import { user } from "@/user";

const client = createTestClient();

describe("role", () => {
  describe("create", () => {
    it("should allow the caller to create a role", async () => {
      const role = await client.access.roles.create({
        name: "test",
        description: "test",
      });
      expect(role.key).toBeDefined();
      expect(role.name).toBe("test");
      expect(role.description).toBe("test");
    });
  });

  describe("retrieve", () => {
    it("should allow the caller to retrieve a role", async () => {
      const created = await client.access.roles.create({
        name: "test",
        description: "test",
      });
      const retrieved = await client.access.roles.retrieve({ key: created.key });
      expect(retrieved.key).toBe(created.key);
      expect(retrieved.name).toBe(created.name);
      expect(retrieved.description).toBe(created.description);
    });
  });

  describe("delete", () => {
    it("should allow the caller to delete a role", async () => {
      const created = await client.access.roles.create({
        name: "test",
        description: "test",
      });
      await client.access.roles.delete(created.key);
      await expect(client.access.roles.retrieve({ key: created.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("assign", () => {
    it("should allow the caller to assign a role to a user", async () => {
      const role = await client.access.roles.create({
        name: "test",
        description: "test",
      });
      const username = id.create();
      const u = await client.users.create({
        username,
        password: "test",
        firstName: "test",
        lastName: "test",
      });
      await client.access.roles.assign({
        user: user.ontologyID(u.key),
        role: role.key,
      });
    });
  });
});
