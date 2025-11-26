// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { policy } from "@/access/policy";
import { AuthError, NotFoundError } from "@/errors";
import { createTestClientWithPolicy } from "@/testutil/access";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("policy", () => {
  describe("retrieve", () => {
    it("should filter by internal flag when retrieving policies", async () => {
      // Create a non-internal policy
      const created = await client.access.policies.create({
        name: "test-non-internal",
        effect: "allow",
        objects: [],
        actions: ["retrieve"],
      });

      // Retrieve only internal policies (built-in system policies)
      const internalPolicies = await client.access.policies.retrieve({
        internal: true,
      });
      expect(internalPolicies.length).toBeGreaterThan(0);
      expect(internalPolicies.every((p) => p.internal === true)).toBe(true);
      expect(internalPolicies.find((p) => p.key === created.key)).toBeUndefined();

      // Retrieve only non-internal policies
      const nonInternalPolicies = await client.access.policies.retrieve({
        internal: false,
      });
      expect(nonInternalPolicies.every((p) => p.internal !== true)).toBe(true);
      expect(nonInternalPolicies.find((p) => p.key === created.key)).toBeDefined();
    });
  });

  describe("access control", () => {
    it("should prevent the caller to retrieve policies with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [],
        actions: ["retrieve"],
      });
      const randomPolicy = await client.access.policies.create({
        name: "test",
        effect: "allow",
        objects: [],
        actions: ["retrieve"],
      });
      await expect(
        userClient.access.policies.retrieve({ key: randomPolicy.key }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to retrieve policies with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [policy.ontologyID("")],
        actions: ["retrieve"],
      });
      const randomPolicy = await client.access.policies.create({
        name: "test",
        effect: "allow",
        objects: [],
        actions: ["retrieve"],
      });
      const retrieved = await userClient.access.policies.retrieve({
        key: randomPolicy.key,
      });
      expect(retrieved.key).toBe(randomPolicy.key);
      expect(retrieved.name).toBe(randomPolicy.name);
      expect(retrieved.effect).toBe(randomPolicy.effect);
      expect(retrieved.objects).toEqual(randomPolicy.objects);
      expect(retrieved.actions).toEqual(randomPolicy.actions);
    });

    it("should allow the caller to create policies with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [policy.ontologyID("")],
        actions: ["create"],
      });
      await userClient.access.policies.create({
        name: "test",
        effect: "allow",
        objects: [],
        actions: ["retrieve"],
      });
    });

    it("should prevent the caller to create policies with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [policy.ontologyID("")],
        actions: ["create"],
      });
      await expect(
        userClient.access.policies.create({
          name: "test",
          effect: "allow",
          objects: [],
          actions: ["retrieve"],
        }),
      ).rejects.toThrow(AuthError);
    });

    it("should allow the caller to delete policies with the correct policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "allow",
        objects: [policy.ontologyID("")],
        actions: ["delete"],
      });
      const randomPolicy = await client.access.policies.create({
        name: "test",
        effect: "allow",
        objects: [],
        actions: ["retrieve"],
      });
      await userClient.access.policies.delete(randomPolicy.key);
      await expect(
        userClient.access.policies.retrieve({ key: randomPolicy.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should prevent the caller to delete policies with the incorrect policy", async () => {
      const userClient = await createTestClientWithPolicy(client, {
        name: "test",
        effect: "deny",
        objects: [policy.ontologyID("")],
        actions: ["delete"],
      });
      const randomPolicy = await client.access.policies.create({
        name: "test",
        effect: "allow",
        objects: [],
        actions: ["retrieve"],
      });
      await expect(userClient.access.policies.delete(randomPolicy.key)).rejects.toThrow(
        AuthError,
      );
    });
  });
});
