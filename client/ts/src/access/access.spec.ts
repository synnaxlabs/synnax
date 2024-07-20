// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, id, Rate } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { Policy } from "@/access/payload";
import { Channel } from "@/channel/client";
import Synnax from "@/client";
import { AuthError, QueryError } from "@/errors";
import {
  ChannelOntologyType,
  LabelOntologyType,
  UserOntologyType,
} from "@/ontology/payload";
import { HOST, newClient, PORT } from "@/setupspecs";

const client = newClient();

describe("Channel", () => {
  describe("create", () => {
    test("create one", async () => {
      const policy = await client.access.create({
        subjects: [{ type: UserOntologyType, key: "1" }],
        objects: [
          { type: UserOntologyType, key: "2" },
          { type: ChannelOntologyType, key: "3" },
        ],
        actions: ["update"],
      });
      expect(policy.key).not.toEqual("");
      expect(policy.subjects).toEqual([{ type: UserOntologyType, key: "1" }]);
      expect(policy.objects).toEqual([
        { type: UserOntologyType, key: "2" },
        { type: ChannelOntologyType, key: "3" },
      ]);
      expect(policy.actions).toEqual(["update"]);
    });
    test("create many", async () => {
      const policies = await client.access.create([
        {
          subjects: [{ type: UserOntologyType, key: "10" }],
          objects: [
            { type: UserOntologyType, key: "20" },
            { type: UserOntologyType, key: "21" },
          ],
          actions: ["update"],
        },
        {
          subjects: [
            { type: UserOntologyType, key: "20" },
            { type: UserOntologyType, key: "21" },
          ],
          objects: [
            { type: UserOntologyType, key: "20" },
            { type: ChannelOntologyType, key: "30" },
          ],
          actions: ["update"],
        },
      ]);
      expect(policies.length).toEqual(2);
      expect(policies[0].subjects[0].key).toEqual("10");
      expect(policies[1].subjects[1].key).toEqual("21");
    });
    test("create instances of channels", async () => {
      const policy = {
        key: undefined,
        subjects: [
          { type: UserOntologyType, key: "20" },
          { type: UserOntologyType, key: "21" },
        ],
        objects: [
          { type: UserOntologyType, key: "20" },
          { type: ChannelOntologyType, key: "30" },
        ],
        actions: ["update"],
      };

      const p = await client.access.create(policy);
      expect(p.subjects).toEqual(policy.subjects);
      expect(p.key).not.toEqual(policy.key);
    });
  });
  test("retrieve by subject", async () => {
    const key1 = id.id();
    const policies = [
      {
        key: undefined,
        subjects: [
          { type: UserOntologyType, key: key1 },
          { type: UserOntologyType, key: "21" },
        ],
        objects: [
          { type: UserOntologyType, key: "234" },
          { type: ChannelOntologyType, key: "30" },
        ],
        actions: ["update"],
      },
      {
        key: undefined,
        subjects: [
          { type: UserOntologyType, key: key1 },
          { type: UserOntologyType, key: "22" },
        ],
        objects: [
          { type: LabelOntologyType, key: "23123" },
          { type: ChannelOntologyType, key: "30" },
        ],
        actions: ["delete"],
      },
    ];

    await client.access.create(policies);

    const p = (await client.access.retrieve(policies[0].subjects[0])) as Policy[];
    expect(p).toHaveLength(2);
    expect([p[0].actions, p[1].actions].sort()).toEqual([["delete"], ["update"]]);
  });
  test("retrieve by subject - not found", async () => {
    await expect(
      async () => await client.access.retrieve({ type: UserOntologyType, key: "999" }),
    ).rejects.toThrow(QueryError);
  });

  describe("delete", async () => {
    test("delete one", async () => {
      const policies = [
        {
          key: undefined,
          subjects: [
            { type: UserOntologyType, key: "20" },
            { type: UserOntologyType, key: "21" },
          ],
          objects: [
            { type: UserOntologyType, key: "20" },
            { type: ChannelOntologyType, key: "30" },
          ],
          actions: ["update"],
        },
        {
          key: undefined,
          subjects: [
            { type: UserOntologyType, key: "20" },
            { type: UserOntologyType, key: "22" },
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
      await expect(
        async () => await client.channels.retrieve(created[0].key),
      ).rejects.toThrow(QueryError);
    });
    test("delete many", async () => {
      const policies = [
        {
          subjects: [
            { type: UserOntologyType, key: "20" },
            { type: UserOntologyType, key: "21" },
          ],
          objects: [
            { type: UserOntologyType, key: "20" },
            { type: ChannelOntologyType, key: "30" },
          ],
          actions: ["update"],
        },
        {
          subjects: [
            { type: UserOntologyType, key: "20" },
            { type: UserOntologyType, key: "22" },
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
      await expect(
        async () => await client.channels.retrieve(created[0].key),
      ).rejects.toThrow(QueryError);
      await expect(
        async () => await client.channels.retrieve(created[1].key),
      ).rejects.toThrow(QueryError);
    });
  });
  describe("registration", async () => {
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
        client2.channels.create(
          new Channel({
            dataType: DataType.FLOAT64,
            rate: Rate.hz(1),
            name: "my_channel",
          }),
        ),
      ).rejects.toThrow(AuthError);

      const p = await client.access.create({
        subjects: [{ type: UserOntologyType, key: user2.key }],
        objects: [{ type: ChannelOntologyType, key: "" }],
        actions: ["create"],
      });

      await client2.channels.create(
        new Channel({
          dataType: DataType.FLOAT64,
          rate: Rate.hz(1),
          name: "my_channel",
        }),
      );

      // Remove privileges
      await client.access.delete(p.key);

      await expect(
        client2.channels.create(
          new Channel({
            dataType: DataType.FLOAT64,
            rate: Rate.hz(1),
            name: "my_channel",
          }),
        ),
      ).rejects.toThrow(AuthError);
    });
  });
});
