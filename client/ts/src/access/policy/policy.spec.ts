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
import { user } from "@/user";

const client = createTestClient();

describe("Policy", () => {
  describe("create", () => {
    describe("one", () => {
      test("without key", async () => {
        const policy = await client.access.policy.create({
          subjects: "user",
          objects: ["user", "channel"],
          actions: "delete",
        });
        expect(policy.key).toBeDefined();
        expect(policy.subjects.length).toEqual(1);
        expect(policy.subjects[0].key).toEqual("");
        expect(policy.subjects[0].type).toEqual("user");
        expect(policy.objects.length).toEqual(2);
        expect(policy.objects[0].key).toEqual("");
        expect(policy.objects[1].key).toEqual("");
        expect(policy.objects[0].type).toEqual("user");
        expect(policy.objects[1].type).toEqual("channel");
        expect(policy.actions).toEqual(["delete"]);
        await client.access.policy.delete(policy.key);
      });

      test("missing subjects", async () => {
        const policy = await client.access.policy.create({
          subjects: [],
          objects: [],
          actions: [],
        });
        expect(policy.key).toBeDefined();
        expect(policy.subjects).toHaveLength(0);
        expect(policy.objects).toHaveLength(0);
        expect(policy.actions).toHaveLength(0);
        const retrievedPolicy = await client.access.policy.retrieve({
          key: policy.key,
        });
        expect(retrievedPolicy).toMatchObject(policy);
      });
      test("missing objects", async () => {
        const policy = await client.access.policy.create({
          subjects: "user",
          objects: [],
          actions: [],
        });
        expect(policy.key).toBeDefined();
        expect(policy.subjects.length).toEqual(1);
        expect(policy.subjects[0].key).toEqual("");
        expect(policy.subjects[0].type).toEqual("user");
        expect(policy.objects).toHaveLength(0);
        expect(policy.actions).toHaveLength(0);
        const retrievedPolicy = await client.access.policy.retrieve({
          key: policy.key,
        });
        expect(retrievedPolicy).toMatchObject(policy);
      });
      test("with key", async () => {
        const policy = await client.access.policy.create({
          subjects: [
            { type: "user", key: "1" },
            { type: "channel", key: "2" },
          ],
          objects: { type: "channel", key: "3" },
          actions: ["delete", "retrieve"],
        });
        expect(policy.key).toBeDefined();
        expect(policy.subjects.length).toEqual(2);
        expect(policy.subjects[0].key).toEqual("1");
        expect(policy.subjects[0].type).toEqual("user");
        expect(policy.subjects[1].key).toEqual("2");
        expect(policy.subjects[1].type).toEqual("channel");
        expect(policy.objects.length).toEqual(1);
        expect(policy.objects[0].key).toEqual("3");
        expect(policy.objects[0].type).toEqual("channel");
        expect(policy.actions).toEqual(["delete", "retrieve"]);
        await client.access.policy.delete(policy.key);
      });
    });
    describe("many", () => {
      test("with keys", async () => {
        const policiesToCreate: policy.New[] = [
          {
            subjects: [{ type: "user", key: "10" }],
            objects: [
              { type: "user", key: "20" },
              { type: "schematic", key: "21" },
            ],
            actions: ["retrieve"],
          },
          {
            subjects: [
              { type: "user", key: "20" },
              { type: "schematic", key: "21" },
            ],
            objects: [
              { type: "user", key: "20" },
              { type: "schematic", key: "30" },
            ],
            actions: ["delete"],
          },
        ];
        const policies = await client.access.policy.create(policiesToCreate);
        expect(policies[0]).toMatchObject(policiesToCreate[0]);
        expect(policies[1]).toMatchObject(policiesToCreate[1]);
        await client.access.policy.delete([policies[0].key, policies[1].key]);
      });
      test("without keys", async () => {
        const policies = await client.access.policy.create([
          {
            subjects: "user",
            objects: ["user", "schematic"],
            actions: ["retrieve"],
          },
          {
            subjects: ["user", "schematic"],
            objects: ["channel"],
            actions: "retrieve",
          },
        ]);
        expect(policies.length).toEqual(2);
        expect(policies[0].key).toBeDefined();
        expect(policies[0].subjects.length).toEqual(1);
        expect(policies[0].subjects[0].key).toEqual("");
        expect(policies[0].subjects[0].type).toEqual("user");
        expect(policies[0].objects.length).toEqual(2);
        expect(policies[0].objects[0].key).toEqual("");
        expect(policies[0].objects[1].key).toEqual("");
        expect(policies[0].objects[0].type).toEqual("user");
        expect(policies[0].objects[1].type).toEqual("schematic");
        expect(policies[0].actions).toEqual(["retrieve"]);
        expect(policies[1].key).toBeDefined();
        expect(policies[1].subjects.length).toEqual(2);
        expect(policies[1].subjects[0].key).toEqual("");
        expect(policies[1].subjects[1].key).toEqual("");
        expect(policies[1].subjects[0].type).toEqual("user");
        expect(policies[1].subjects[1].type).toEqual("schematic");
        expect(policies[1].objects.length).toEqual(1);
        expect(policies[1].objects[0].key).toEqual("");
        expect(policies[1].objects[0].type).toEqual("channel");
        expect(policies[1].actions).toEqual(["retrieve"]);
        await client.access.policy.delete([policies[0].key, policies[1].key]);
      });
    });
  });
  describe("retrieve", async () => {
    test("by key", async () => {
      const policies = await client.access.policy.create([
        {
          subjects: "user",
          objects: ["user", "channel"],
          actions: "delete",
        },
        {
          subjects: "user",
          objects: ["schematic", "channel"],
          actions: "retrieve",
        },
      ]);
      const result = await client.access.policy.retrieve({ key: policies[0].key });
      expect(result).toMatchObject(policies[0]);
      const results = await client.access.policy.retrieve({
        keys: [policies[0].key, policies[1].key],
      });
      expect(results).toHaveLength(2);
      expect(results[0]).toMatchObject(policies[0]);
      expect(results[1]).toMatchObject(policies[1]);
      expect(results.sort()).toMatchObject(policies.sort());
      await client.access.policy.delete([policies[0].key, policies[1].key]);
    });
    test("by subject", async () => {
      const key1 = id.create();
      const key2 = id.create();
      const created = await client.access.policy.create([
        {
          subjects: [
            { type: "user", key: key1 },
            { type: "user", key: key2 },
          ],
          objects: [
            { type: "user", key: "234" },
            { type: "channel", key: "30" },
          ],
          actions: ["retrieve"],
        },
        {
          subjects: { type: "user", key: key1 },
          objects: [
            { type: "label", key: "23123" },
            { type: "channel", key: "30" },
          ],
          actions: "delete",
        },
      ]);
      const received = await client.access.policy.retrieve({
        for: user.ontologyID(key2),
      });
      const newReceived = received.filter((p) => created.some((c) => c.key === p.key));
      expect(created[0]).toMatchObject(newReceived[0]);
      await client.access.policy.delete([created[0].key, created[1].key]);
    });
  });
  describe("delete", async () => {
    test("one", async () => {
      const id1 = id.create();
      const id2 = id.create();
      const id3 = id.create();
      const policies: policy.New[] = [
        {
          subjects: [
            { type: "user", key: id1 },
            { type: "user", key: id2 },
          ],
          objects: [
            { type: "user", key: "20" },
            { type: "channel", key: "30" },
          ],
          actions: ["retrieve"],
        },
        {
          subjects: [
            { type: "user", key: id1 },
            { type: "user", key: id3 },
          ],
          objects: [
            { type: "label", key: "20" },
            { type: "channel", key: "30" },
          ],
          actions: ["delete"],
        },
      ];

      const created = await client.access.policy.create(policies);
      await client.access.policy.delete(created[0].key);
      await expect(
        client.access.policy.retrieve({ key: created[0].key }),
      ).rejects.toThrow();
      await client.access.policy.delete(created[1].key);
    });
    test("many", async () => {
      const id1 = id.create();
      const id2 = id.create();
      const id3 = id.create();
      const policies: policy.New[] = [
        {
          subjects: [
            { type: "user", key: id1 },
            { type: "user", key: id2 },
          ],
          objects: [
            { type: "user", key: "20" },
            { type: "channel", key: "30" },
          ],
          actions: ["retrieve"],
        },
        {
          subjects: [
            { type: "user", key: id1 },
            { type: "user", key: id3 },
          ],
          objects: [
            { type: "label", key: "20" },
            { type: "channel", key: "30" },
          ],
          actions: ["delete"],
        },
      ];

      const created = await client.access.policy.create(policies);
      await client.access.policy.delete([created[0].key, created[1].key]);
      await expect(
        client.access.policy.retrieve({ key: created[0].key }),
      ).rejects.toThrow();
      await expect(
        client.access.policy.retrieve({ key: created[1].key }),
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

    const policy = await client.access.policy.create({
      subjects: "user",
      objects: "user",
      actions: ["create"],
    });

    const newUsername = id.create();

    const newUser = await client2.users.create({
      username: newUsername,
      password: id.create(),
    });

    expect(newUser.username).toEqual(newUsername);

    // Remove privileges
    await client.access.policy.delete(policy.key);

    await expect(
      client2.users.create({ username: id.create(), password: id.create() }),
    ).rejects.toThrow(AuthError);
  });
});
