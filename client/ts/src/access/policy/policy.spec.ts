// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { policy } from "@/access/policy";
import { channel } from "@/channel";
import Synnax from "@/client";
import { AuthError } from "@/errors";
import { label } from "@/label";
import { HOST, newClient, PORT } from "@/setupspecs";
import { user } from "@/user";
import { schematic } from "@/workspace/schematic";

const client = newClient();

const sortByKey = (a: any, b: any) => a.key.localeCompare(b.key);

describe("Policy", () => {
  describe.skip("create", () => {
    describe("one", () => {
      test("without key", async () => {
        const policy = await client.access.policy.create({
          subjects: user.ONTOLOGY_TYPE,
          objects: [user.ONTOLOGY_TYPE, channel.ONTOLOGY_TYPE],
          actions: "delete",
        });
        expect(policy.key).exist;
        expect(policy.subjects.length).toEqual(1);
        expect(policy.subjects[0].key).toEqual("");
        expect(policy.subjects[0].type).toEqual(user.ONTOLOGY_TYPE);
        expect(policy.objects.length).toEqual(2);
        expect(policy.objects[0].key).toEqual("");
        expect(policy.objects[1].key).toEqual("");
        expect(policy.objects[0].type).toEqual(user.ONTOLOGY_TYPE);
        expect(policy.objects[1].type).toEqual(channel.ONTOLOGY_TYPE);
        expect(policy.actions).toEqual(["delete"]);
        await client.access.policy.delete(policy.key);
      });
      test("missing subjects", async () => {
        const policy = await client.access.policy.create({
          subjects: [],
          objects: [],
          actions: [],
        });
        console.log("policy", policy);
        expect(policy.key).exist;
        expect(policy.subjects).toHaveLength(0);
        expect(policy.objects).toHaveLength(0);
        expect(policy.actions).toHaveLength(0);
        const retrievedPolicy = await client.access.policy.retrieve(policy.key);
        expect(retrievedPolicy).toMatchObject(policy);
      });
      test("missing objects", async () => {
        const policy = await client.access.policy.create({
          subjects: user.ontologyID("1"),
          objects: [],
          actions: [],
        });
        expect(policy.key).exist;
        expect(policy.subjects.length).toEqual(1);
        expect(policy.subjects[0].key).toEqual("1");
        expect(policy.subjects[0].type).toEqual(user.ONTOLOGY_TYPE);
        expect(policy.objects).toHaveLength(0);
        expect(policy.actions).toHaveLength(0);
        const retrievedPolicy = await client.access.policy.retrieve(policy.key);
        expect(retrievedPolicy).toMatchObject(policy);
      });
      test("with key", async () => {
        const policy = await client.access.policy.create({
          subjects: [
            { type: user.ONTOLOGY_TYPE, key: "1" },
            { type: channel.ONTOLOGY_TYPE, key: "2" },
          ],
          objects: { type: channel.ONTOLOGY_TYPE, key: "3" },
          actions: ["delete", "retrieve"],
        });
        expect(policy.key).exist;
        expect(policy.subjects.length).toEqual(2);
        expect(policy.subjects[0].key).toEqual("1");
        expect(policy.subjects[0].type).toEqual(user.ONTOLOGY_TYPE);
        expect(policy.subjects[1].key).toEqual("2");
        expect(policy.subjects[1].type).toEqual(channel.ONTOLOGY_TYPE);
        expect(policy.objects.length).toEqual(1);
        expect(policy.objects[0].key).toEqual("3");
        expect(policy.objects[0].type).toEqual(channel.ONTOLOGY_TYPE);
        expect(policy.actions).toEqual(["delete", "retrieve"]);
        await client.access.policy.delete(policy.key);
      });
    });
    describe("many", () => {
      test("with keys", async () => {
        const policiesToCreate: policy.NewPolicy[] = [
          {
            subjects: [{ type: user.ONTOLOGY_TYPE, key: "10" }],
            objects: [
              { type: user.ONTOLOGY_TYPE, key: "20" },
              { type: schematic.ONTOLOGY_TYPE, key: "21" },
            ],
            actions: ["retrieve"],
          },
          {
            subjects: [
              { type: user.ONTOLOGY_TYPE, key: "20" },
              { type: schematic.ONTOLOGY_TYPE, key: "21" },
            ],
            objects: [
              { type: user.ONTOLOGY_TYPE, key: "20" },
              { type: schematic.ONTOLOGY_TYPE, key: "30" },
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
            subjects: user.ONTOLOGY_TYPE,
            objects: [user.ONTOLOGY_TYPE, schematic.ONTOLOGY_TYPE],
            actions: ["retrieve"],
          },
          {
            subjects: [user.ONTOLOGY_TYPE, schematic.ONTOLOGY_TYPE],
            objects: [channel.ONTOLOGY_TYPE],
            actions: "retrieve",
          },
        ]);
        expect(policies.length).toEqual(2);
        expect(policies[0].key).exist;
        expect(policies[0].subjects.length).toEqual(1);
        expect(policies[0].subjects[0].key).toEqual("");
        expect(policies[0].subjects[0].type).toEqual(user.ONTOLOGY_TYPE);
        expect(policies[0].objects.length).toEqual(2);
        expect(policies[0].objects[0].key).toEqual("");
        expect(policies[0].objects[1].key).toEqual("");
        expect(policies[0].objects[0].type).toEqual(user.ONTOLOGY_TYPE);
        expect(policies[0].objects[1].type).toEqual(schematic.ONTOLOGY_TYPE);
        expect(policies[0].actions).toEqual(["retrieve"]);
        expect(policies[1].key).exist;
        expect(policies[1].subjects.length).toEqual(2);
        expect(policies[1].subjects[0].key).toEqual("");
        expect(policies[1].subjects[1].key).toEqual("");
        expect(policies[1].subjects[0].type).toEqual(user.ONTOLOGY_TYPE);
        expect(policies[1].subjects[1].type).toEqual(schematic.ONTOLOGY_TYPE);
        expect(policies[1].objects.length).toEqual(1);
        expect(policies[1].objects[0].key).toEqual("");
        expect(policies[1].objects[0].type).toEqual(channel.ONTOLOGY_TYPE);
        expect(policies[1].actions).toEqual(["retrieve"]);
        await client.access.policy.delete([policies[0].key, policies[1].key]);
      });
    });
  });
  describe.skip("retrieve", async () => {
    test("by key", async () => {
      const policies = await client.access.policy.create([
        {
          subjects: user.ONTOLOGY_TYPE,
          objects: [user.ONTOLOGY_TYPE, channel.ONTOLOGY_TYPE],
          actions: "delete",
        },
        {
          subjects: user.ONTOLOGY_TYPE,
          objects: [schematic.ONTOLOGY_TYPE, channel.ONTOLOGY_TYPE],
          actions: "retrieve",
        },
      ]);
      const result = await client.access.policy.retrieve(policies[0].key);
      expect(result).toMatchObject(policies[0]);
      const results = await client.access.policy.retrieve([
        policies[0].key,
        policies[1].key,
      ]);
      expect(results).toHaveLength(2);
      expect(results[0]).toMatchObject(policies[0]);
      expect(results[1]).toMatchObject(policies[1]);
      expect(results.sort()).toMatchObject(policies.sort());
      await client.access.policy.delete([policies[0].key, policies[1].key]);
    });
    test("by subject", async () => {
      const key1 = id.id();
      const key2 = id.id();
      const created = await client.access.policy.create([
        {
          subjects: [
            { type: user.ONTOLOGY_TYPE, key: key1 },
            { type: user.ONTOLOGY_TYPE, key: key2 },
          ],
          objects: [
            { type: user.ONTOLOGY_TYPE, key: "234" },
            { type: channel.ONTOLOGY_TYPE, key: "30" },
          ],
          actions: ["retrieve"],
        },
        {
          subjects: { type: user.ONTOLOGY_TYPE, key: key1 },
          objects: [
            { type: label.ONTOLOGY_TYPE, key: "23123" },
            { type: channel.ONTOLOGY_TYPE, key: "30" },
          ],
          actions: "delete",
        },
      ]);
      const received = await client.access.policy.retrieveFor(user.ontologyID(key2));
      expect(created[0]).toMatchObject(received[0]);
      const received2 = await client.access.policy.retrieveFor({
        type: user.ONTOLOGY_TYPE,
        key: key1,
      });
      expect(created.sort(sortByKey)).toMatchObject(received2.sort(sortByKey));
      await client.access.policy.delete([created[0].key, created[1].key]);
    });
  });
  describe.skip("delete", async () => {
    test("one", async () => {
      const id1 = id.id();
      const id2 = id.id();
      const id3 = id.id();
      const policies: policy.NewPolicy[] = [
        {
          subjects: [
            { type: user.ONTOLOGY_TYPE, key: id1 },
            { type: user.ONTOLOGY_TYPE, key: id2 },
          ],
          objects: [
            { type: user.ONTOLOGY_TYPE, key: "20" },
            { type: channel.ONTOLOGY_TYPE, key: "30" },
          ],
          actions: ["retrieve"],
        },
        {
          subjects: [
            { type: user.ONTOLOGY_TYPE, key: id1 },
            { type: user.ONTOLOGY_TYPE, key: id3 },
          ],
          objects: [
            { type: label.ONTOLOGY_TYPE, key: "20" },
            { type: channel.ONTOLOGY_TYPE, key: "30" },
          ],
          actions: ["delete"],
        },
      ];

      const created = await client.access.policy.create(policies);
      await client.access.policy.delete(created[0].key);
      const res = await client.access.policy.retrieveFor(created[0].subjects[0]);
      expect(res).toHaveLength(1);
      expect(res[0].actions).toEqual(["delete"]);
      await client.access.policy.delete(created[1].key);
    });
    test("many", async () => {
      const id1 = id.id();
      const id2 = id.id();
      const id3 = id.id();
      const policies: policy.NewPolicy[] = [
        {
          subjects: [
            { type: user.ONTOLOGY_TYPE, key: id1 },
            { type: user.ONTOLOGY_TYPE, key: id2 },
          ],
          objects: [
            { type: user.ONTOLOGY_TYPE, key: "20" },
            { type: channel.ONTOLOGY_TYPE, key: "30" },
          ],
          actions: ["retrieve"],
        },
        {
          subjects: [
            { type: user.ONTOLOGY_TYPE, key: id1 },
            { type: user.ONTOLOGY_TYPE, key: id3 },
          ],
          objects: [
            { type: label.ONTOLOGY_TYPE, key: "20" },
            { type: channel.ONTOLOGY_TYPE, key: "30" },
          ],
          actions: ["delete"],
        },
      ];

      const created = await client.access.policy.create(policies);
      await client.access.policy.delete([created[0].key, created[1].key]);
      let res = await client.access.policy.retrieveFor({
        type: user.ONTOLOGY_TYPE,
        key: id1,
      });
      expect(res).toHaveLength(0);
      res = await client.access.policy.retrieveFor({
        type: user.ONTOLOGY_TYPE,
        key: id2,
      });
      expect(res).toHaveLength(0);
      res = await client.access.policy.retrieveFor({
        type: user.ONTOLOGY_TYPE,
        key: id3,
      });
      expect(res).toHaveLength(0);
    });
  });
  describe("current test I'm failing", async () => {
    test.skip("create and retrieve", async () => {
      const userKey = "0b22b11d-b659-4061-b8cd-5cc7ef8b6800";
      const newPolicy = {
        subjects: [{ type: user.ONTOLOGY_TYPE, key: userKey }],
        objects: [{ type: policy.ALLOW_ALL_ONTOLOGY_TYPE, key: "" }],
        actions: [],
        key: "9438295b-fc2d-4ec9-a012-763c43ac0bb6",
      };
      const created = await client.access.policy.create(newPolicy);
      expect(created).toBeDefined();
      expect(created.key).toBeDefined;
      expect(created.subjects).toHaveLength(1);
      expect(created.subjects[0].key).toEqual(userKey);
      expect(created.subjects[0].type).toEqual(user.ONTOLOGY_TYPE);
      expect(created.objects).toHaveLength(1);
      expect(created.objects[0].key).toEqual("");
      expect(created.objects[0].type).toEqual(policy.ALLOW_ALL_ONTOLOGY_TYPE);
      const retrieved = (
        await client.access.policy.retrieveFor(user.ontologyID(userKey))
      )[0];
      expect(retrieved).toMatchObject(created);
      console.log("createdPolicy", created);
      console.log("retrievedPolicy", retrieved);
    });
    test("root user", async () => {
      const username = client.props.username;
      const rootUser = await client.user.retrieveByName(username);
      await expect(
        client.access.policy.retrieveFor(user.ontologyID(rootUser.key)),
      ).resolves.toBeDefined();
    });
  });
});

describe.skip("privilege", async () => {
  test("new user", async () => {
    const username = id.id();
    const user2 = await client.user.create({ username, password: "pwd1" });
    expect(user2).toBeDefined();
    const client2 = new Synnax({
      host: HOST,
      port: PORT,
      username: user2.username,
      password: "pwd1",
    });
    await expect(
      client2.channels.create({
        name: "my_channel",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      }),
    ).rejects.toThrow(AuthError);

    const policy = await client.access.policy.create({
      subjects: [{ type: user.ONTOLOGY_TYPE, key: user2.key }],
      objects: [{ type: channel.ONTOLOGY_TYPE, key: "" }],
      actions: ["create"],
    });

    const chan = await client2.channels.create({
      name: "my_channel",
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });

    expect(chan).toBeDefined();
    expect(chan.name).toEqual("my_channel");
    expect(chan.dataType).toEqual(DataType.TIMESTAMP);
    expect(chan.isIndex).toEqual(true);

    // Remove privileges
    await client.access.policy.delete(policy.key);

    await expect(
      client2.channels.create({
        name: "my_channel",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      }),
    ).rejects.toThrow(AuthError);
  });
});