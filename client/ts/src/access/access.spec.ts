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

import { type NewPolicy } from "@/access/payload";
import { ChannelOntologyType } from "@/channel/payload";
import Synnax from "@/client";
import { AuthError } from "@/errors";
import { LabelOntologyType } from "@/label/payload";
import { HOST, newClient, PORT } from "@/setupspecs";
import { UserOntologyType } from "@/user/payload";
import { SchematicOntologyType } from "@/workspace/schematic/payload";

const client = newClient();

const sortByKey = (a: any, b: any) => {
  return a.key.localeCompare(b.key);
};

describe("Policy", () => {
  describe("create", () => {
    describe("one", () => {
      test("without key", async () => {
        const policy = await client.access.create({
          subjects: UserOntologyType,
          objects: [UserOntologyType, ChannelOntologyType],
          actions: "delete",
        });
        expect(policy.key).exist;
        expect(policy.subjects.length).toEqual(1);
        expect(policy.subjects[0].key).toEqual("");
        expect(policy.subjects[0].type).toEqual(UserOntologyType);
        expect(policy.objects.length).toEqual(2);
        expect(policy.objects[0].key).toEqual("");
        expect(policy.objects[1].key).toEqual("");
        expect(policy.objects[0].type).toEqual(UserOntologyType);
        expect(policy.objects[1].type).toEqual(ChannelOntologyType);
        expect(policy.actions).toEqual(["delete"]);
        await client.access.delete(policy.key);
      });
      test("with key", async () => {
        const policy = await client.access.create({
          subjects: [
            { type: UserOntologyType, key: "1" },
            { type: ChannelOntologyType, key: "2" },
          ],
          objects: { type: ChannelOntologyType, key: "3" },
          actions: ["delete", "retrieve"],
        });
        expect(policy.key).exist;
        expect(policy.subjects.length).toEqual(2);
        expect(policy.subjects[0].key).toEqual("1");
        expect(policy.subjects[0].type).toEqual(UserOntologyType);
        expect(policy.subjects[1].key).toEqual("2");
        expect(policy.subjects[1].type).toEqual(ChannelOntologyType);
        expect(policy.objects.length).toEqual(1);
        expect(policy.objects[0].key).toEqual("3");
        expect(policy.objects[0].type).toEqual(ChannelOntologyType);
        expect(policy.actions).toEqual(["delete", "retrieve"]);
        await client.access.delete(policy.key);
      });
    });
    describe("many", () => {
      test("with keys", async () => {
        const policiesToCreate: NewPolicy[] = [
          {
            subjects: [{ type: UserOntologyType, key: "10" }],
            objects: [
              { type: UserOntologyType, key: "20" },
              { type: SchematicOntologyType, key: "21" },
            ],
            actions: ["retrieve"],
          },
          {
            subjects: [
              { type: UserOntologyType, key: "20" },
              { type: SchematicOntologyType, key: "21" },
            ],
            objects: [
              { type: UserOntologyType, key: "20" },
              { type: ChannelOntologyType, key: "30" },
            ],
            actions: ["delete"],
          },
        ];
        const policies = await client.access.create(policiesToCreate);
        expect(policies[0]).toMatchObject(policiesToCreate[0]);
        expect(policies[1]).toMatchObject(policiesToCreate[1]);
        await client.access.delete([policies[0].key, policies[1].key]);
      });
      test("without keys", async () => {
        const policies = await client.access.create([
          {
            subjects: UserOntologyType,
            objects: [UserOntologyType, SchematicOntologyType],
            actions: ["retrieve"],
          },
          {
            subjects: [UserOntologyType, SchematicOntologyType],
            objects: [ChannelOntologyType],
            actions: "retrieve",
          },
        ]);
        expect(policies.length).toEqual(2);
        expect(policies[0].key).exist;
        expect(policies[0].subjects.length).toEqual(1);
        expect(policies[0].subjects[0].key).toEqual("");
        expect(policies[0].subjects[0].type).toEqual(UserOntologyType);
        expect(policies[0].objects.length).toEqual(2);
        expect(policies[0].objects[0].key).toEqual("");
        expect(policies[0].objects[1].key).toEqual("");
        expect(policies[0].objects[0].type).toEqual(UserOntologyType);
        expect(policies[0].objects[1].type).toEqual(SchematicOntologyType);
        expect(policies[0].actions).toEqual(["retrieve"]);
        expect(policies[1].key).exist;
        expect(policies[1].subjects.length).toEqual(2);
        expect(policies[1].subjects[0].key).toEqual("");
        expect(policies[1].subjects[1].key).toEqual("");
        expect(policies[1].subjects[0].type).toEqual(UserOntologyType);
        expect(policies[1].subjects[1].type).toEqual(SchematicOntologyType);
        expect(policies[1].objects.length).toEqual(1);
        expect(policies[1].objects[0].key).toEqual("");
        expect(policies[1].objects[0].type).toEqual(ChannelOntologyType);
        expect(policies[1].actions).toEqual(["retrieve"]);
        await client.access.delete([policies[0].key, policies[1].key]);
      });
    });
  });
  describe("retrieve", async () => {
    test("by key", async () => {
      const policies = await client.access.create([
        {
          subjects: UserOntologyType,
          objects: [UserOntologyType, ChannelOntologyType],
          actions: "delete",
        },
        {
          subjects: UserOntologyType,
          objects: [SchematicOntologyType, ChannelOntologyType],
          actions: "retrieve",
        },
      ]);
      const result = await client.access.retrieve(policies[0].key);
      expect(result).toMatchObject(policies[0]);
      const results = await client.access.retrieve([policies[0].key, policies[1].key]);
      expect(results).toHaveLength(2);
      expect(results[0]).toMatchObject(policies[0]);
      expect(results[1]).toMatchObject(policies[1]);
      expect(results.sort()).toMatchObject(policies.sort());
      await client.access.delete([policies[0].key, policies[1].key]);
    });
    test("by subject", async () => {
      const key1 = id.id();
      const key2 = id.id();
      const created = await client.access.create([
        {
          subjects: [
            { type: UserOntologyType, key: key1 },
            { type: UserOntologyType, key: key2 },
          ],
          objects: [
            { type: UserOntologyType, key: "234" },
            { type: ChannelOntologyType, key: "30" },
          ],
          actions: ["retrieve"],
        },
        {
          subjects: { type: UserOntologyType, key: key1 },
          objects: [
            { type: LabelOntologyType, key: "23123" },
            { type: ChannelOntologyType, key: "30" },
          ],
          actions: "delete",
        },
      ]);
      const received = await client.access.retrieveFor({
        type: UserOntologyType,
        key: key2,
      });
      expect(created[0]).toMatchObject(received[0]);
      const received2 = await client.access.retrieveFor({
        type: UserOntologyType,
        key: key1,
      });
      expect(created.sort(sortByKey)).toMatchObject(received2.sort(sortByKey));
      await client.access.delete([created[0].key, created[1].key]);
    });
  });
  describe("delete", async () => {
    test("one", async () => {
      const id1 = id.id();
      const id2 = id.id();
      const id3 = id.id();
      const policies: NewPolicy[] = [
        {
          subjects: [
            { type: UserOntologyType, key: id1 },
            { type: UserOntologyType, key: id2 },
          ],
          objects: [
            { type: UserOntologyType, key: "20" },
            { type: ChannelOntologyType, key: "30" },
          ],
          actions: ["retrieve"],
        },
        {
          subjects: [
            { type: UserOntologyType, key: id1 },
            { type: UserOntologyType, key: id3 },
          ],
          objects: [
            { type: LabelOntologyType, key: "20" },
            { type: ChannelOntologyType, key: "30" },
          ],
          actions: ["delete"],
        },
      ];

      const created = await client.access.create(policies);
      await client.access.delete(created[0].key);
      const res = await client.access.retrieveFor(created[0].subjects[0]);
      expect(res).toHaveLength(1);
      expect(res[0].actions).toEqual(["delete"]);
      await client.access.delete(created[1].key);
    });
    test("many", async () => {
      const id1 = id.id();
      const id2 = id.id();
      const id3 = id.id();
      const policies: NewPolicy[] = [
        {
          subjects: [
            { type: UserOntologyType, key: id1 },
            { type: UserOntologyType, key: id2 },
          ],
          objects: [
            { type: UserOntologyType, key: "20" },
            { type: ChannelOntologyType, key: "30" },
          ],
          actions: ["retrieve"],
        },
        {
          subjects: [
            { type: UserOntologyType, key: id1 },
            { type: UserOntologyType, key: id3 },
          ],
          objects: [
            { type: LabelOntologyType, key: "20" },
            { type: ChannelOntologyType, key: "30" },
          ],
          actions: ["delete"],
        },
      ];

      const created = await client.access.create(policies);
      await client.access.delete([created[0].key, created[1].key]);
      let res = await client.access.retrieveFor({ type: UserOntologyType, key: id1 });
      expect(res).toHaveLength(0);
      res = await client.access.retrieveFor({ type: UserOntologyType, key: id2 });
      expect(res).toHaveLength(0);
      res = await client.access.retrieveFor({ type: UserOntologyType, key: id3 });
      expect(res).toHaveLength(0);
    });
  });
});
describe("Registration", async () => {
  test("register a user", async () => {
    const username = id.id();
    await client.user.register(username, "pwd1");
    new Synnax({
      host: HOST,
      port: PORT,
      username: username,
      password: "pwd1",
    });
  });
  test("duplicate username", async () => {
    const username = id.id();
    await client.user.register(username, "pwd1");
    await expect(client.user.register(username, "pwd1")).rejects.toThrow(AuthError);
  });
});
describe("privilege", async () => {
  test("new user", async () => {
    const username = id.id();
    const user2 = await client.user.register(username, "pwd1");
    expect(user2).toBeDefined();
    const client2 = new Synnax({
      host: HOST,
      port: PORT,
      username: username,
      password: "pwd1",
    });

    await expect(
      client2.channels.create({
        name: "my_channel",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      }),
    ).rejects.toThrow(AuthError);

    const policy = await client.access.create({
      subjects: [{ type: UserOntologyType, key: user2.key }],
      objects: [{ type: ChannelOntologyType, key: "" }],
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
    await client.access.delete(policy.key);

    await expect(
      client2.channels.create({
        name: "my_channel",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      }),
    ).rejects.toThrow(AuthError);
  });
});
