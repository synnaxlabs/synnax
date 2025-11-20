// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { role } from "@/access/role";
import { AuthError, NotFoundError } from "@/errors";
import { createClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("role", () => {
  describe("access control", () => {
    it("should prevent the caller to retrieve roles with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const randomRole = await client.access.roles.create({
        name: "test",
        description: "test",
        policies: [],
      });
      await expect(
        userClient.access.roles.retrieve({ key: randomRole.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve roles with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [role.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomRole = await client.access.roles.create({
        name: "test",
        description: "test",
        policies: [],
      });
      const retrieved = await userClient.access.roles.retrieve({ key: randomRole.key });
      expect(retrieved.key).toBe(randomRole.key);
      expect(retrieved.name).toBe(randomRole.name);
      expect(retrieved.description).toBe(randomRole.description);
      expect(retrieved.policies).toEqual(randomRole.policies);
    });

    it("should allow the caller to create roles with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [role.ontologyID("")],
        actions: ["create"],
      });
      await userClient.access.roles.create({
        name: "test",
        description: "test",
        policies: [],
      });
    });

    it("should prevent the caller to create roles with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [role.ontologyID("")],
        actions: ["create"],
      });
      await expect(
        userClient.access.roles.create({
          name: "test",
          description: "test",
          policies: [],
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete roles with the correct policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [role.ontologyID("")],
        actions: ["delete"],
      });
      const randomRole = await client.access.roles.create({
        name: "test",
        description: "test",
        policies: [],
      });
      await userClient.access.roles.delete(randomRole.key);
      await expect(
        userClient.access.roles.retrieve({ key: randomRole.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should prevent the caller to delete roles with the incorrect policy", async () => {
      const userClient = await createClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [role.ontologyID("")],
        actions: ["delete"],
      });
      const randomRole = await client.access.roles.create({
        name: "test",
        description: "test",
        policies: [],
      });
      await expect(userClient.access.roles.delete(randomRole.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
