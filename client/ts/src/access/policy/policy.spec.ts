// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { type policy } from "@/access/policy";
import { AuthError } from "@/errors";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Policy", () => {
  describe("create", () => {
    describe("one", () => {
      test("without key", async () => {
        const policy = await client.access.policies.create({
          name: "test",
          effect: "allow",
          objects: ["user", "channel"],
          actions: "delete",
        });
        expect(policy.key).toBeDefined();
        expect(policy.objects.length).toEqual(2);
        expect(policy.actions).toEqual(["delete"]);
        await client.access.policies.delete(policy.key);
      });

      test("missing subjects", async () => {
        const policy = await client.access.policies.create({
          name: "test",
          effect: "allow",
          objects: [],
          actions: [],
        });
        expect(policy.key).toBeDefined();
        expect(policy.actions).toHaveLength(0);
        const retrievedPolicy = await client.access.policies.retrieve({
          key: policy.key,
        });
        expect(retrievedPolicy).toMatchObject(policy);
      });
      test("missing objects", async () => {
        const policy = await client.access.policies.create({
          name: "test",
          effect: "allow",
          objects: [],
          actions: [],
        });
        expect(policy.key).toBeDefined();
        expect(policy.actions).toHaveLength(0);
        const retrievedPolicy = await client.access.policies.retrieve({
          key: policy.key,
        });
        expect(retrievedPolicy).toMatchObject(policy);
      });
      test("with key", async () => {
        const policy = await client.access.policies.create({
          name: "test",
          effect: "allow",
          objects: { type: "channel", key: "3" },
          actions: ["delete", "retrieve"],
        });
        expect(policy.key).toBeDefined();
        expect(policy.actions).toEqual(["delete", "retrieve"]);
        await client.access.policies.delete(policy.key);
      });
    });
    describe("many", () => {
      test("with keys", async () => {
        const policiesToCreate: policy.New[] = [
          {
            name: "test",
            effect: "allow",
            objects: [
              { type: "user", key: "20" },
              { type: "schematic", key: "21" },
            ],
            actions: ["retrieve"],
          },
          {
            name: "test",
            effect: "allow",
            objects: [
              { type: "user", key: "20" },
              { type: "schematic", key: "30" },
            ],
            actions: ["delete"],
          },
        ];
        const policies = await client.access.policies.create(policiesToCreate);
        expect(policies[0]).toMatchObject(policiesToCreate[0]);
        expect(policies[1]).toMatchObject(policiesToCreate[1]);
        await client.access.policies.delete([policies[0].key, policies[1].key]);
      });
      test("without keys", async () => {
        const policies = await client.access.policies.create([
          {
            name: "test",
            effect: "allow",
            objects: ["user", "schematic"],
            actions: ["retrieve"],
          },
          {
            name: "test",
            effect: "allow",
            objects: ["channel"],
            actions: "retrieve",
          },
        ]);
        expect(policies.length).toEqual(2);
        expect(policies[0].key).toBeDefined();
        expect(policies[0].actions).toEqual(["retrieve"]);
        expect(policies[1].key).toBeDefined();
        expect(policies[1].objects.length).toEqual(1);
        expect(policies[1].objects[0].key).toEqual("");
        expect(policies[1].objects[0].type).toEqual("channel");
        expect(policies[1].actions).toEqual(["retrieve"]);
        await client.access.policies.delete([policies[0].key, policies[1].key]);
      });
    });
  });
  describe("retrieve", async () => {
    test("by key", async () => {
      const policies = await client.access.policies.create([
        {
          objects: ["user", "channel"],
          name: "test",
          effect: "allow",
          actions: "delete",
        },
        {
          objects: ["schematic", "channel"],
          name: "test",
          effect: "allow",
          actions: "retrieve",
        },
      ]);
      const result = await client.access.policies.retrieve({ key: policies[0].key });
      expect(result).toMatchObject(policies[0]);
      const results = await client.access.policies.retrieve({
        keys: [policies[0].key, policies[1].key],
      });
      expect(results).toHaveLength(2);
      expect(results[0]).toMatchObject(policies[0]);
      expect(results[1]).toMatchObject(policies[1]);
      expect(results.sort()).toMatchObject(policies.sort());
      await client.access.policies.delete([policies[0].key, policies[1].key]);
    });
  });
  describe("delete", async () => {
    test("one", async () => {
      const policies: policy.New[] = [
        {
          name: "test",
          effect: "allow",
          objects: [
            { type: "user", key: "20" },
            { type: "channel", key: "30" },
          ],
          actions: ["retrieve"],
        },
        {
          name: "test",
          effect: "allow",
          objects: [
            { type: "label", key: "20" },
            { type: "channel", key: "30" },
          ],
          actions: ["delete"],
        },
      ];

      const created = await client.access.policies.create(policies);
      await client.access.policies.delete(created[0].key);
      await expect(
        client.access.policies.retrieve({ key: created[0].key }),
      ).rejects.toThrow();
      await client.access.policies.delete(created[1].key);
    });
    test("many", async () => {
      const policies: policy.New[] = [
        {
          name: "test",
          effect: "allow",
          objects: [
            { type: "user", key: "20" },
            { type: "channel", key: "30" },
          ],
          actions: ["retrieve"],
        },
        {
          name: "test",
          effect: "allow",
          objects: [
            { type: "label", key: "20" },
            { type: "channel", key: "30" },
          ],
          actions: ["delete"],
        },
      ];

      const created = await client.access.policies.create(policies);
      await client.access.policies.delete([created[0].key, created[1].key]);
      await expect(
        client.access.policies.retrieve({ key: created[0].key }),
      ).rejects.toThrow();
      await expect(
        client.access.policies.retrieve({ key: created[1].key }),
      ).rejects.toThrow();
    });
  });
});

describe("privilege", async () => {
  test("new user", async () => {
    const username = id.create();
    const user2 = await client.users.create({ username, password: "pwd1" });
    expect(user2).toBeDefined();
    const client2 = createTestClient({ username: user2.username, password: "pwd1" });
    await expect(
      client2.users.create({ username: id.create(), password: id.create() }),
    ).rejects.toThrow(AuthError);

    const policy = await client.access.policies.create({
      name: "test",
      effect: "allow",
      objects: [{ type: "user", key: username }],
      actions: ["create"],
    });

    const newUsername = id.create();

    const newUser = await client2.users.create({
      username: newUsername,
      password: id.create(),
    });

    expect(newUser.username).toEqual(newUsername);

    // Remove privileges
    await client.access.policies.delete(policy.key);

    await expect(
      client2.users.create({ username: id.create(), password: id.create() }),
    ).rejects.toThrow(AuthError);
  });
});
