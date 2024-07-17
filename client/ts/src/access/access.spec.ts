// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { QueryError } from "@/errors";
import { newClient } from "@/setupspecs";
import { v4 as uuid } from 'uuid';

const client = newClient();

describe("Channel", () => {
  describe("create", () => {
    test("create one", async () => {
      const policy = await client.access.create({
        subjects: [{type: "user", key: "1"}],
        objects: [{type: "user", key: "2"}, {type: "channel", key: "3"}],
        actions: ["update"],
      });
      expect(policy.key).not.toEqual("")
      expect(policy.subjects).toEqual([{type: "user", key: "1"}]);
      expect(policy.objects).toEqual([{type: "user", key: "2"}, {type: "channel", key: "3"}]);
      expect(policy.actions).toEqual(["update"]);
    });
    test("create many", async () => {
      const policies = await client.access.create([
        {
          subjects: [{type: "user", key: "10"}],
          objects: [{type: "user", key: "20"}, {type: "user", key: "21"}],
          actions: ["update"],
        },
        {
          subjects: [{type: "user", key: "20"}, {type: "user", key: "21"}],
          objects: [{type: "user", key: "20"}, {type: "channel", key: "30"}],
          actions: ["update"],
        },
      ]);
      expect(policies.length).toEqual(2);
      expect(policies[0].subjects[0].key).toEqual("10");
      expect(policies[1].subjects[1].key).toEqual("21");
    });
    test("create instances of channels", async () => {
      const policy ={
        key: undefined,
        subjects: [{type: "user", key: "20"}, {type: "user", key: "21"}],
        objects: [{type: "user", key: "20"}, {type: "channel", key: "30"}],
        actions: ["update"],
      };

      const p = await client.access.create(policy);
      expect(p.subjects).toEqual(policy.subjects)
      expect(p.key).not.toEqual(policy.key)
    });
  });
  test("retrieve by subject", async () => {
    const key1 = uuid().toString()
    const policies = [{
      key: undefined,
      subjects: [{type: "user", key: key1}, {type: "user", key: "21"}],
      objects: [{type: "user", key: "234"}, {type: "channel", key: "30"}],
      actions: ["update"],
    },{
      key: undefined,
      subjects: [{type: "user", key: key1}, {type: "user", key: "22"}],
      objects: [{type: "label", key: "23123"}, {type: "channel", key: "30"}],
      actions: ["delete"],
    }];

    await client.access.create(policies)

    const p = await client.access.retrieve(policies[0].subjects[0]);
    expect(p).toHaveLength(2);
    expect([p[0].actions, p[1].actions].sort()).toEqual([["delete"], ["update"]]);
  });
  test("retrieve by subject - not found", async () => {
    await expect(async () => await client.access.retrieve({type: "user", key: "999"})).rejects.toThrow(
      QueryError,
    );
  });

  describe("delete", async () => {
    test("delete one", async () => {
      const policies = [{
        key: undefined,
        subjects: [{type: "user", key: "20"}, {type: "user", key: "21"}],
        objects: [{type: "user", key: "20"}, {type: "channel", key: "30"}],
        actions: ["update"],
      },{
        key: undefined,
        subjects: [{type: "user", key: "20"}, {type: "user", key: "22"}],
        objects: [{type: "label", key: "20"}, {type: "channel", key: "30"}],
        actions: ["delete"],
      }];

      const created = await client.access.create(policies)
      await client.access.delete(created[0].key)
      await expect(
        async () => await client.channels.retrieve(created[0].key),
      ).rejects.toThrow(QueryError);
    });
    test("delete many", async () => {
      const policies = [{
        key: undefined,
        subjects: [{type: "user", key: "20"}, {type: "user", key: "21"}],
        objects: [{type: "user", key: "20"}, {type: "channel", key: "30"}],
        actions: ["update"],
      },{
        key: undefined,
        subjects: [{type: "user", key: "20"}, {type: "user", key: "22"}],
        objects: [{type: "label", key: "20"}, {type: "channel", key: "30"}],
        actions: ["delete"],
      }];

      const created = await client.access.create(policies)
      await client.access.delete([created[0].key, created[1].key])
      await expect(
        async () => await client.channels.retrieve(created[0].key),
      ).rejects.toThrow(QueryError);
      await expect(
        async () => await client.channels.retrieve(created[1].key),
      ).rejects.toThrow(QueryError);
    });
  });
});
