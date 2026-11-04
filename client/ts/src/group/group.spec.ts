// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { channel } from "@/channel";
import { group } from "@/group";
import { NotFoundError, ValidationError } from "@/index";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const createTopLevel = async (): Promise<group.Group> =>
  await client.groups.create({ name: `group-${id.create()}`, scope: "channel" });

describe("Group", () => {
  describe("create", () => {
    it("should create a top-level group in a scope", async () => {
      const name = `group-${id.create()}`;
      const g = await client.groups.create({ name, scope: "channel" });
      expect(g.name).toEqual(name);
      expect(g.scope).toEqual("channel");
      expect(g.parent).toBeUndefined();
    });
    it("should take the scope of its parent group", async () => {
      const parent = await createTopLevel();
      const g = await client.groups.create({
        name: `child-${id.create()}`,
        parent: group.resourceID(parent.key),
      });
      expect(g.scope).toEqual("channel");
      expect(g.parent).toEqual(group.resourceID(parent.key));
    });
    it("should reject a top-level group with no scope", async () => {
      await expect(
        client.groups.create({ name: `group-${id.create()}` }),
      ).rejects.toThrow(ValidationError);
    });
    it("should update an existing group", async () => {
      const g = await createTopLevel();
      await client.groups.create({
        key: g.key,
        name: "updated-name",
        scope: "channel",
      });
      expect((await client.groups.retrieve({ key: g.key })).name).toEqual(
        "updated-name",
      );
    });
  });
  describe("retrieve", () => {
    it("should retrieve the groups under a parent", async () => {
      const parent = await createTopLevel();
      const parentID = group.resourceID(parent.key);
      const child = await client.groups.create({ name: "child", parent: parentID });
      expect(await client.groups.retrieve({ parents: [parentID] })).toEqual([child]);
    });
    it("should retrieve the ancestors of a group", async () => {
      const top = await createTopLevel();
      const mid = await client.groups.create({
        name: "mid",
        parent: group.resourceID(top.key),
      });
      const leaf = await client.groups.create({
        name: "leaf",
        parent: group.resourceID(mid.key),
      });
      const ancestors = await client.groups.retrieveAncestors(leaf.key);
      expect(ancestors.map((g) => g.key).sort()).toEqual([top.key, mid.key].sort());
    });
  });
  describe("rename", () => {
    it("should correctly rename a group", async () => {
      const g = await createTopLevel();
      const newName = `group-${id.create()}`;
      await client.groups.rename(g.key, newName);
      expect((await client.groups.retrieve({ key: g.key })).name).toEqual(newName);
    });
  });
  describe("setParent", () => {
    it("should move a group under a group and back to the top level", async () => {
      const parent = await createTopLevel();
      const g = await createTopLevel();
      const parentID = group.resourceID(parent.key);
      await client.groups.setParent(g.key, parentID);
      expect(
        (await client.groups.retrieve({ parents: [parentID] })).map((c) => c.key),
      ).toEqual([g.key]);
      await client.groups.setParent(g.key, null);
      expect(await client.groups.retrieve({ parents: [parentID] })).toEqual([]);
    });
    it("should reject a move that makes a cycle", async () => {
      const parent = await createTopLevel();
      const child = await client.groups.create({
        name: "child",
        parent: group.resourceID(parent.key),
      });
      await expect(
        client.groups.setParent(parent.key, group.resourceID(child.key)),
      ).rejects.toThrow(ValidationError);
    });
  });
  describe("delete", () => {
    it("should correctly delete the group", async () => {
      const g = await createTopLevel();
      await client.groups.delete(g.key);
      await expect(client.groups.retrieve({ key: g.key })).rejects.toThrow(
        NotFoundError,
      );
    });
    it("should refuse to delete a group that holds a group", async () => {
      const parent = await createTopLevel();
      await client.groups.create({
        name: "child",
        parent: group.resourceID(parent.key),
      });
      await expect(client.groups.delete(parent.key)).rejects.toThrow(ValidationError);
    });
  });
  describe("batch", () => {
    it("should create a group and move a channel into it at once", async () => {
      const ch = await client.channels.create({
        name: `batch-${id.create()}`,
        dataType: "float32",
        virtual: true,
      });
      const key = uuid.create();
      await client.batch.send([
        group.createRequest({ key, name: "Batched", scope: "channel" }),
        channel.setGroupRequest(ch.key, key),
      ]);
      expect(
        (await client.channels.retrieve({ groups: [key] })).map((c) => c.key),
      ).toEqual([ch.key]);
    });
    it("should commit nothing when a request fails", async () => {
      const key = uuid.create();
      await expect(
        client.batch.send([
          group.createRequest({ key, name: "Rolled back", scope: "channel" }),
          group.setParentRequest(key, group.resourceID(key)),
        ]),
      ).rejects.toThrow(ValidationError);
      await expect(client.groups.retrieve({ key })).rejects.toThrow(NotFoundError);
    });
  });
});
