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

import { AuthError, NotFoundError } from "@/errors";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";
import { user } from "@/user";

const client = createTestClient();

describe("user", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve users with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const randomUser = await client.users.create({
        username: id.create(),
        password: "test",
      });
      await expect(userClient.users.retrieve({ key: randomUser.key })).rejects.toThrow(
        AuthError,
      );
    });

    it("should allow the caller to retrieve users with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [user.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomUser = await client.users.create({
        username: id.create(),
        password: "test",
      });
      const retrieved = await userClient.users.retrieve({ key: randomUser.key });
      expect(retrieved.key).toBe(randomUser.key);
      expect(retrieved.username).toBe(randomUser.username);
    });

    it("should allow the caller to create users with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [user.ontologyID("")],
        actions: ["create"],
      });
      await userClient.users.create({
        username: id.create(),
        password: "test",
      });
    });

    it("should prevent the caller to create users with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [user.ontologyID("")],
        actions: ["create"],
      });
      await expect(
        userClient.users.create({
          username: id.create(),
          password: "test",
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete users with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [user.ontologyID("")],
        actions: ["delete"],
      });
      const randomUser = await client.users.create({
        username: id.create(),
        password: "test",
      });
      await userClient.users.delete(randomUser.key);
      await expect(userClient.users.retrieve({ key: randomUser.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should prevent the caller to delete users with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [user.ontologyID("")],
        actions: ["delete"],
      });
      const randomUser = await client.users.create({
        username: id.create(),
        password: "test",
      });
      await expect(userClient.users.delete(randomUser.key)).rejects.toThrow(AuthError);
    });
  });
});
