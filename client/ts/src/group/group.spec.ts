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

import { group } from "@/group";
import { NotFoundError } from "@/index";
import { ontology } from "@/ontology";
import { createTestClient } from "@/testutil";

const client = createTestClient();

describe("Group", () => {
  describe("create", () => {
    it("should correctly create a group", async () => {
      const name = `group-${Math.random()}`;
      const g = await client.groups.create({ parent: ontology.ROOT_ID, name });
      expect(g.name).toEqual(name);
    });
    it("should update an existing group", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `test-parent-key${id.create()}`,
      });
      const g = await client.groups.create({
        parent: group.ontologyID(parent.key),
        name: `original-name-${id.create()}`,
      });
      await client.groups.create({
        parent: group.ontologyID(parent.key),
        key: g.key,
        name: "updated-name",
      });
      const g2 = await client.ontology.retrieve(group.ontologyID(g.key));
      expect(g2.name).toEqual("updated-name");
    });
  });
  describe("retrieve", () => {
    it("should retrieve the groups that match where", async () => {
      const a = await client.groups.create({ parent: ontology.ROOT_ID, name: "a" });
      const b = await client.groups.create({ parent: ontology.ROOT_ID, name: "b" });
      const groups = await client.groups.retrieve({
        where: { key: { in: [a.key, b.key] } },
      });
      expect(groups.map((g) => g.key).sort()).toEqual([a.key, b.key].sort());
    });
    it("should retrieve the groups that match a condition", async () => {
      const a = await client.groups.create({ parent: ontology.ROOT_ID, name: "a" });
      await client.groups.create({ parent: ontology.ROOT_ID, name: "b" });
      const groups = await client.groups.retrieve(group.key.in([a.key]));
      expect(groups.map((g) => g.key)).toEqual([a.key]);
    });
    it("should leave out a key that no group has", async () => {
      const a = await client.groups.create({ parent: ontology.ROOT_ID, name: "a" });
      const groups = await client.groups.retrieve(group.key.in([a.key, uuid.create()]));
      expect(groups.map((g) => g.key)).toEqual([a.key]);
    });
    it("should retrieve the children of a parent that match where", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `parent-${id.create()}`,
      });
      const parentID = group.ontologyID(parent.key);
      const a = await client.groups.create({ parent: parentID, name: "a" });
      await client.groups.create({ parent: parentID, name: "b" });
      const outside = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "outside",
      });
      const groups = await client.groups.retrieve({
        parent: parentID,
        where: { key: { in: [a.key, outside.key] } },
      });
      expect(groups.map((g) => g.key)).toEqual([a.key]);
    });
    it("should retrieve the children of a parent", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `parent-${id.create()}`,
      });
      const parentID = group.ontologyID(parent.key);
      const a = await client.groups.create({ parent: parentID, name: "a" });
      const b = await client.groups.create({ parent: parentID, name: "b" });
      await client.groups.create({ parent: ontology.ROOT_ID, name: "outside" });
      const groups = await client.groups.retrieve({ parent: parentID });
      expect(groups.map((g) => g.key).sort()).toEqual([a.key, b.key].sort());
    });
    it("should return no children of a parent for a zero limit", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: `parent-${id.create()}`,
      });
      const parentID = group.ontologyID(parent.key);
      await client.groups.create({ parent: parentID, name: "a" });
      expect(await client.groups.retrieve({ parent: parentID, limit: 0 })).toEqual([]);
    });
  });
  describe("rename", () => {
    it("should correctly rename a group", async () => {
      const name = `group-${Math.random()}`;
      const g = await client.groups.create({ parent: ontology.ROOT_ID, name });
      const newName = `group-${Math.random()}`;
      await client.groups.rename(g.key, newName);
      const g2 = await client.ontology.retrieve(group.ontologyID(g.key));
      expect(g2.name).toEqual(newName);
    });
  });
  describe("delete", () => {
    it("should correctly delete the group", async () => {
      const name = `group-${Math.random()}`;
      const g = await client.groups.create({ parent: ontology.ROOT_ID, name });
      await client.groups.delete(g.key);
      await expect(
        async () => await client.ontology.retrieve(group.ontologyID(g.key)),
      ).rejects.toThrow(NotFoundError);
    });
  });
});
