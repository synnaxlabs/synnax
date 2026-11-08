// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { role } from "@/access/role";
import { AccessDeniedError, NotFoundError } from "@/index";
import {
  createEmptyRole,
  createTestClient,
  createTestClientWithPolicy,
} from "@/testutil";
import { user } from "@/user";

const client = createTestClient();

const empty = await createEmptyRole(client);

describe("user", () => {
  describe("access control", () => {
    it("should deny access when no retrieve policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [],
        actions: [],
      });
      const randomUser = await client.users.create({
        username: id.create(),
        password: "test",
        role: empty.key,
      });
      await expect(userClient.users.retrieve(randomUser.key)).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
    });

    it("should allow the caller to retrieve users with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [user.resourceID("")],
        actions: ["retrieve"],
      });
      const randomUser = await client.users.create({
        username: id.create(),
        password: "test",
        role: empty.key,
      });
      const retrieved = await userClient.users.retrieve(randomUser.key);
      expect(retrieved.key).toBe(randomUser.key);
    });

    it("should allow the caller to create users with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, [
        { name: "test", objects: [user.resourceID("")], actions: ["create", "update"] },
        { name: "test", objects: [role.resourceID(empty.key)], actions: ["update"] },
      ]);
      const created = await userClient.users.create({
        username: id.create(),
        password: "test",
        role: empty.key,
      });
      expect(created.role).toBe(empty.key);
    });

    it("should deny creating a user whose role the caller cannot update", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [user.resourceID("")],
        actions: ["create", "update"],
      });
      await expect(
        userClient.users.create({
          username: id.create(),
          password: "test",
          role: empty.key,
        }),
      ).rejects.toSatisfy(AccessDeniedError.matches);
    });

    it("should deny access when no create policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [user.resourceID("")],
        actions: [],
      });
      await expect(
        userClient.users.create({
          username: id.create(),
          password: "test",
          role: empty.key,
        }),
      ).rejects.toSatisfy(AccessDeniedError.matches);
    });

    it("should allow the caller to delete users with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [user.resourceID("")],
        actions: ["delete"],
      });
      const randomUser = await client.users.create({
        username: id.create(),
        password: "test",
        role: empty.key,
      });
      await userClient.users.delete(randomUser.key);
      await expect(userClient.users.retrieve(randomUser.key)).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should deny access when no delete policy exists", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        objects: [user.resourceID("")],
        actions: [],
      });
      const randomUser = await client.users.create({
        username: id.create(),
        password: "test",
        role: empty.key,
      });
      await expect(userClient.users.delete(randomUser.key)).rejects.toSatisfy(
        AccessDeniedError.matches,
      );
    });
  });
});
