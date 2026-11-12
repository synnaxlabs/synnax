// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { destructor, id, uuid } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { channel } from "@/channel";
import { group } from "@/group";
import { NotFoundError, ValidationError } from "@/index";
import { query } from "@/query";
import { type resource } from "@/resource";
import { createTestClient, newVirtualChannel } from "@/testutil";

const client = createTestClient();

const createTopLevel = async (): Promise<group.Group> =>
  await client.groups.create({ name: `group-${id.create()}`, scope: "channel" });

const createChain = async (): Promise<group.Group[]> => {
  const root = await createTopLevel();
  const mid = await client.groups.create({
    name: "mid",
    parent: group.resourceID(root.key),
  });
  const leaf = await client.groups.create({
    name: "leaf",
    parent: group.resourceID(mid.key),
  });
  return [root, mid, leaf];
};

describe("Group", () => {
  describe("request builders", () => {
    const RACK: resource.ID = { type: "rack", key: "1" };
    const KEY = "00000000-0000-4000-8000-000000000001";
    it("should build a create request", () => {
      expect(client.groups.createRequest({ name: "Pumps", parent: RACK })).toEqual({
        endpoint: "/group/create",
        payload: { name: "Pumps", parent: RACK },
      });
    });
    it("should write a group with a key to the cache, and roll the write back", () => {
      const key = uuid.create();
      const { apply } = client.groups.createRequest({
        key,
        name: "Pumps",
        scope: "device",
        parent: RACK,
      });
      const rollback = apply?.();
      expect(client.groups.watch(key).getCached()).toEqual({
        key,
        name: "Pumps",
        scope: "device",
        parent: RACK,
      });
      rollback?.();
      expect(query.isLive(client.groups.watch(key).getCached())).toBe(false);
    });
    it("should not write a group with no scope to the cache", () => {
      expect(
        client.groups.createRequest({ key: uuid.create(), name: "Pumps", parent: RACK })
          .apply,
      ).toBeUndefined();
    });
    it("should not write a group with no key to the cache", () => {
      expect(
        client.groups.createRequest({ name: "Pumps", scope: "channel" }).apply,
      ).toBeUndefined();
    });
    it("should build a delete request from a single key", () => {
      expect(client.groups.deleteRequest(KEY)).toMatchObject({
        endpoint: "/group/delete",
        payload: { keys: [KEY] },
      });
    });
    it("should build one set-parent dispatch entry per group", () => {
      const KEY2 = "00000000-0000-4000-8000-000000000002";
      expect(client.groups.setParentRequest([KEY, KEY2], RACK)).toEqual([
        {
          endpoint: "/group/dispatch",
          payload: { key: KEY, actions: [group.setParent({ parent: RACK })] },
          apply: expect.any(Function),
        },
        {
          endpoint: "/group/dispatch",
          payload: { key: KEY2, actions: [group.setParent({ parent: RACK })] },
          apply: expect.any(Function),
        },
      ]);
    });
    it("should build a set-parent entry that moves a group to the top level", () => {
      const [entry] = client.groups.setParentRequest(KEY, null);
      expect(entry.payload.actions).toEqual([
        { type: "set_parent", setParent: { parent: undefined } },
      ]);
    });
  });
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
      ).rejects.toThrow("scope: invalid scope");
    });
    it("should drop the cached group when the Core rejects the create", async () => {
      const key = uuid.create();
      await expect(
        client.groups.create({ key, name: "", scope: "channel" }),
      ).rejects.toThrow("name: required");
      expect(query.isLive(client.groups.watch(key).getCached())).toBe(false);
    });
    it("should refuse a group whose key exists and keep the stored group", async () => {
      const g = await createTopLevel();
      const create = client.groups.create({
        key: g.key,
        name: "updated-name",
        scope: "channel",
      });
      await expect(create).rejects.toThrow(ValidationError);
      await expect(create).rejects.toThrow(`group ${g.key} already exists`);
      expect((await client.groups.retrieve({ key: g.key })).name).toEqual(g.name);
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
    it("should keep a parent query live as a matching group lands", async () => {
      const parent = await createTopLevel();
      const list = client.groups.watch({ parents: [group.resourceID(parent.key)] });
      const handler = vi.fn();
      const stop = list.onChange(handler);
      expect((await list.retrieve()).records).toEqual([]);
      const child = await client.groups.create({
        name: "child",
        parent: group.resourceID(parent.key),
      });
      await expect
        .poll(() => list.getCached()?.records)
        .toEqual([expect.objectContaining({ key: child.key })]);
      stop();
    });
  });
  describe("retrieve", () => {
    it("should retrieve the groups that match where", async () => {
      const a = await createTopLevel();
      const b = await createTopLevel();
      const groups = await client.groups.retrieve({
        where: { key: { in: [a.key, b.key] } },
      });
      expect(groups.map((g) => g.key).sort()).toEqual([a.key, b.key].sort());
    });
    it("should leave out a key that no group has", async () => {
      const a = await createTopLevel();
      const groups = await client.groups.retrieve({ key: [a.key, uuid.create()] });
      expect(groups.map((g) => g.key)).toEqual([a.key]);
    });
    it("should retrieve the children of a parent that match where", async () => {
      const parent = await createTopLevel();
      const parentID = group.resourceID(parent.key);
      const a = await client.groups.create({ parent: parentID, name: "a" });
      await client.groups.create({ parent: parentID, name: "b" });
      const outside = await createTopLevel();
      const groups = await client.groups.retrieve({
        where: {
          and: [{ parents: [parentID] }, { key: { in: [a.key, outside.key] } }],
        },
      });
      expect(groups.map((g) => g.key)).toEqual([a.key]);
    });
    it("should retrieve the children of a parent by a flat filter", async () => {
      const parent = await createTopLevel();
      const parentID = group.resourceID(parent.key);
      const a = await client.groups.create({ parent: parentID, name: "a" });
      await client.groups.create({ parent: parentID, name: "b" });
      const outside = await createTopLevel();
      const groups = await client.groups.retrieve({
        parents: [parentID],
        key: [a.key, outside.key],
      });
      expect(groups.map((g) => g.key)).toEqual([a.key]);
    });
    it("should retrieve the children of a parent", async () => {
      const parent = await createTopLevel();
      const parentID = group.resourceID(parent.key);
      const a = await client.groups.create({ parent: parentID, name: "a" });
      const b = await client.groups.create({ parent: parentID, name: "b" });
      await createTopLevel();
      const groups = await client.groups.retrieve({ parents: [parentID] });
      expect(groups.map((g) => g.key).sort()).toEqual([a.key, b.key].sort());
    });
    it("should return no children of a parent for a zero limit", async () => {
      const parent = await createTopLevel();
      const parentID = group.resourceID(parent.key);
      await client.groups.create({ parent: parentID, name: "a" });
      expect(await client.groups.retrieve({ parents: [parentID], limit: 0 })).toEqual(
        [],
      );
    });
  });
  describe("getCachedAncestors", () => {
    it("should return each cached group above the groups once", async () => {
      const [root, mid, leaf] = await createChain();
      expect(client.groups.getCachedAncestors([leaf.key, mid.key])).toEqual([root]);
    });
    it("should give the same groups as the Core", async () => {
      const [, mid, leaf] = await createChain();
      expect(client.groups.getCachedAncestors([leaf.key, mid.key])).toEqual(
        await client.groups.retrieveAncestors([leaf.key, mid.key]),
      );
    });
    it("should return undefined when a group above is not cached", async () => {
      const [, , leaf] = await createChain();
      const other = createTestClient();
      await other.groups.retrieve({ key: leaf.key });
      expect(other.groups.getCachedAncestors(leaf.key)).toBeUndefined();
    });
  });
  describe("onAncestorsChange", () => {
    it("should call the handler after a rename or a move above the group", async () => {
      const [root, mid, leaf] = await createChain();
      const handler = vi.fn();
      const stop = client.groups.onAncestorsChange(leaf.key, handler);
      await client.groups.rename(root.key, "renamed");
      await expect
        .poll(() => handler.mock.lastCall?.[0])
        .toEqual([
          expect.objectContaining({ key: mid.key }),
          expect.objectContaining({ key: root.key, name: "renamed" }),
        ]);
      await client.groups.setParent(mid.key, null);
      await expect
        .poll(() => handler.mock.lastCall?.[0])
        .toEqual([expect.objectContaining({ key: mid.key })]);
      stop();
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
  describe("dispatch", () => {
    it("should apply the action to the group", async () => {
      const g = await createTopLevel();
      const name = `group-${id.create()}`;
      await client.groups.dispatch(g.key, group.set({ path: ["name"], value: name }));
      const stored = await createTestClient().groups.retrieve({ key: g.key });
      expect(stored.name).toEqual(name);
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
    it("should move several groups in one call", async () => {
      const parent = await createTopLevel();
      const a = await createTopLevel();
      const b = await createTopLevel();
      const parentID = group.resourceID(parent.key);
      await client.groups.setParent([a.key, b.key], parentID);
      expect(
        (await client.groups.retrieve({ parents: [parentID] }))
          .map((c) => c.key)
          .sort(),
      ).toEqual([a.key, b.key].sort());
    });
    it("should reject a move that makes a cycle and restore the cached parent", async () => {
      const parent = await createTopLevel();
      const child = await client.groups.create({
        name: "child",
        parent: group.resourceID(parent.key),
      });
      await expect(
        client.groups.setParent(parent.key, group.resourceID(child.key)),
      ).rejects.toThrow("would form a cycle");
      expect(client.groups.watch(parent.key).getCached()).toEqual(
        expect.objectContaining({ parent: undefined }),
      );
    });
  });
  describe("moveRequests", () => {
    it("should move records and groups under a group", async () => {
      const [parent, child, ch] = await Promise.all([
        createTopLevel(),
        createTopLevel(),
        newVirtualChannel(client),
      ]);
      await client.batch.send(
        client.groups.moveRequests(
          [channel.resourceID(ch.key), group.resourceID(child.key)],
          group.resourceID(parent.key),
        ),
      );
      expect((await client.channels.retrieve(ch.key)).group).toEqual(parent.key);
      expect((await client.groups.retrieve({ key: child.key })).parent).toEqual(
        group.resourceID(parent.key),
      );
    });
    it("should move records out of their group when the parent is not a group", async () => {
      const [g, ch] = await Promise.all([createTopLevel(), newVirtualChannel(client)]);
      await client.batch.send(client.channels.setGroupRequest(ch.key, g.key));
      await client.batch.send(client.groups.moveRequests([channel.resourceID(ch.key)]));
      expect((await client.channels.retrieve(ch.key)).group).toBeUndefined();
    });
    it("should throw for a type that cannot sit in a group", () => {
      expect(() =>
        client.groups.moveRequests([{ type: "label", key: id.create() }]),
      ).toThrow("a label cannot sit in a group");
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
      await expect(client.groups.delete(parent.key)).rejects.toThrow(
        "because it is not empty",
      );
    });
  });
  describe("batch", () => {
    it("should create a group and move a channel into it at once", async () => {
      const ch = await client.channels.create({
        name: `batch_${id.create()}`,
        dataType: "float32",
        virtual: true,
      });
      const key = uuid.create();
      await client.batch.send([
        client.groups.createRequest({ key, name: "Batched", scope: "channel" }),
        ...client.channels.setGroupRequest(ch.key, key),
      ]);
      expect(
        (await client.channels.retrieve({ groups: [key] })).map((c) => c.key),
      ).toEqual([ch.key]);
    });
    it("should commit nothing when a request fails", async () => {
      const key = uuid.create();
      await expect(
        client.batch.send([
          client.groups.createRequest({ key, name: "Rolled back", scope: "channel" }),
          ...client.groups.setParentRequest(key, group.resourceID(key)),
        ]),
      ).rejects.toThrow("would form a cycle");
      await expect(client.groups.retrieve({ key })).rejects.toThrow(NotFoundError);
    });
    it("should move cached records before the Core answers", async () => {
      const g = await createTopLevel();
      const ch = await client.channels.create({
        name: `batch_${id.create()}`,
        dataType: "float32",
        virtual: true,
      });
      const sent = client.batch.send(client.channels.setGroupRequest(ch.key, g.key));
      expect(client.channels.watch(ch.key).getCached()).toEqual(
        expect.objectContaining({ group: g.key }),
      );
      await sent;
    });
    it("should apply the cache writes again when the batch commits", async () => {
      const g = await createTopLevel();
      const apply = vi.fn(() => destructor.NOOP);
      await client.batch.send([{ ...client.groups.deleteRequest(g.key), apply }]);
      expect(apply).toHaveBeenCalledTimes(2);
    });
    it("should roll back the cache writes when the batch fails", async () => {
      const g = await createTopLevel();
      const ch = await client.channels.create({
        name: `batch_${id.create()}`,
        dataType: "float32",
        virtual: true,
      });
      await expect(
        client.batch.send([
          ...client.channels.setGroupRequest(ch.key, g.key),
          ...client.groups.setParentRequest(g.key, group.resourceID(g.key)),
        ]),
      ).rejects.toThrow("would form a cycle");
      expect(client.channels.watch(ch.key).getCached()).toEqual(
        expect.objectContaining({ group: undefined }),
      );
    });
  });
});
