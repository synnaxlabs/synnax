// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { ontology } from "@/ontology";
import { newTestClient } from "@/testutil/client";

const client = newTestClient();

const randomName = (): string => `group-${Math.random()}`;

describe("Ontology", () => {
  describe("parseIDs", () => {
    it("should parse a single ID object", () => {
      const id: ontology.ID = { type: "group", key: "test-key" };
      const result = ontology.parseIDs(id);
      expect(result).toEqual([id]);
    });

    it("should parse an array of ID objects", () => {
      const ids: ontology.ID[] = [
        { type: "group", key: "test-key-1" },
        { type: "channel", key: "test-key-2" },
      ];
      const result = ontology.parseIDs(ids);
      expect(result).toEqual(ids);
    });

    it("should parse a single string ID", () => {
      const stringId = "group:test-key";
      const result = ontology.parseIDs(stringId);
      expect(result).toEqual([{ type: "group", key: "test-key" }]);
    });

    it("should parse an array of string IDs", () => {
      const stringIds = ["group:test-key-1", "channel:test-key-2"];
      const result = ontology.parseIDs(stringIds);
      expect(result).toEqual([
        { type: "group", key: "test-key-1" },
        { type: "channel", key: "test-key-2" },
      ]);
    });

    it("should extract ID from a single Resource object", () => {
      const resource: ontology.Resource = {
        id: { type: "group", key: "test-key" },
        name: "Test Resource",
        key: "group:test-key",
      };
      const result = ontology.parseIDs(resource);
      expect(result).toEqual([{ type: "group", key: "test-key" }]);
    });

    it("should extract IDs from an array of Resource objects", () => {
      const resources: ontology.Resource[] = [
        {
          id: { type: "group", key: "test-key-1" },
          name: "Test Resource 1",
          key: "group:test-key-1",
        },
        {
          id: { type: "channel", key: "test-key-2" },
          name: "Test Resource 2",
          key: "channel:test-key-2",
        },
      ];
      const result = ontology.parseIDs(resources);
      expect(result).toEqual([
        { type: "group", key: "test-key-1" },
        { type: "channel", key: "test-key-2" },
      ]);
    });

    it("should return empty array for empty input", () => {
      const result = ontology.parseIDs([]);
      expect(result).toEqual([]);
    });

    it("should handle string IDs with colons in the key", () => {
      const stringId = "group:test:key:with:colons";
      const result = ontology.parseIDs(stringId);
      expect(result).toEqual([{ type: "group", key: "test" }]);
    });

    it("should handle mixed Resource objects with different data types", () => {
      const resources: ontology.Resource[] = [
        {
          id: { type: "group", key: "test-key-1" },
          name: "Test Resource 1",
          key: "group:test-key-1",
          data: { customField: "value" },
        },
        {
          id: { type: "channel", key: "test-key-2" },
          name: "Test Resource 2",
          key: "channel:test-key-2",
          data: null,
        },
      ];
      const result = ontology.parseIDs(resources);
      expect(result).toEqual([
        { type: "group", key: "test-key-1" },
        { type: "channel", key: "test-key-2" },
      ]);
    });

    it("should throw an error for invalid string ID format", () => {
      const invalidStringId = "invalid-format";
      expect(() => ontology.parseIDs(invalidStringId)).toThrow();
    });

    it("should throw an error for invalid resource type in string ID", () => {
      const invalidStringId = "invalid-type:test-key";
      expect(() => ontology.parseIDs(invalidStringId)).toThrow();
    });
  });

  describe("retrieve", () => {
    test("retrieve", async () => {
      const name = randomName();
      const g = await client.ontology.groups.create(ontology.ROOT_ID, name);
      const g2 = await client.ontology.retrieve(g.ontologyID);
      expect(g2.name).toEqual(name);
    });
    test("retrieve children", async () => {
      const name = randomName();
      const g = await client.ontology.groups.create(ontology.ROOT_ID, name);
      const name2 = randomName();
      await client.ontology.groups.create(g.ontologyID, name2);
      const children = await client.ontology.retrieveChildren(g.ontologyID);
      expect(children.length).toEqual(1);
      expect(children[0].name).toEqual(name2);
    });
    test("retrieve parents", async () => {
      const name = randomName();
      const g = await client.ontology.groups.create(ontology.ROOT_ID, name);
      const name2 = randomName();
      const g2 = await client.ontology.groups.create(g.ontologyID, name2);
      const parents = await client.ontology.retrieveParents(g2.ontologyID);
      expect(parents.length).toEqual(1);
      expect(parents[0].name).toEqual(name);
    });
  });
  describe("page", () => {
    it("should return a page of resources", async () => {
      for (let i = 0; i < 10; i++)
        await client.ontology.groups.create(ontology.ROOT_ID, randomName());
      const page = await client.ontology.page(0, 5);
      expect(page.length).toEqual(5);
      const page2 = await client.ontology.page(5, 5);
      expect(page2.length).toEqual(5);
      const page1Keys = page.map((r) => r.key);
      const page2Keys = page2.map((r) => r.key);
      const intersection = page1Keys.filter((key) => page2Keys.includes(key));
      expect(intersection.length).toEqual(0);
    });
  });
  describe("write", () => {
    test("add children", async () => {
      const name = randomName();
      const g = await client.ontology.groups.create(ontology.ROOT_ID, name);
      const name2 = randomName();
      const g2 = await client.ontology.groups.create(ontology.ROOT_ID, name2);
      await client.ontology.addChildren(g.ontologyID, g2.ontologyID);
      const children = await client.ontology.retrieveChildren(g.ontologyID);
      expect(children.length).toEqual(1);
      expect(children[0].name).toEqual(name2);
    });
    test("remove children", async () => {
      const name = randomName();
      const g = await client.ontology.groups.create(ontology.ROOT_ID, name);
      const name2 = randomName();
      const g2 = await client.ontology.groups.create(ontology.ROOT_ID, name2);
      await client.ontology.addChildren(g.ontologyID, g2.ontologyID);
      await client.ontology.removeChildren(g.ontologyID, g2.ontologyID);
      const children = await client.ontology.retrieveChildren(g.ontologyID);
      expect(children.length).toEqual(0);
    });
    test("move children", async () => {
      const name = randomName();
      const g = await client.ontology.groups.create(ontology.ROOT_ID, name);
      const name2 = randomName();
      const g2 = await client.ontology.groups.create(ontology.ROOT_ID, name2);
      const oldRootLength = (await client.ontology.retrieveChildren(ontology.ROOT_ID))
        .length;
      await client.ontology.moveChildren(ontology.ROOT_ID, g.ontologyID, g2.ontologyID);
      const children = await client.ontology.retrieveChildren(g.ontologyID);
      expect(children.length).toEqual(1);
      const newRootLength = (await client.ontology.retrieveChildren(ontology.ROOT_ID))
        .length;
      expect(newRootLength).toEqual(oldRootLength - 1);
    });
  });

  describe("signals", async () => {
    it("should correctly decode a set of relationships from a string", () => {
      const rel = ontology.relationshipZ.parse("table:keyA->parent->schematic:keyB");
      expect(rel.type).toEqual(ontology.PARENT_OF_RELATIONSHIP_TYPE);
      expect(rel.from.type).toEqual("table");
      expect(rel.from.key).toEqual("keyA");
      expect(rel.to.type).toEqual("schematic");
      expect(rel.to.key).toEqual("keyB");
    });
    it("should correctly propagate resource changes to the ontology", async () => {
      const change = await client.ontology.openChangeTracker();
      const p = new Promise<ontology.ResourceChange[]>((resolve) =>
        change.resources.onChange((changes) => resolve(changes)),
      );
      await client.ontology.groups.create(ontology.ROOT_ID, randomName());
      const c = await p;
      expect(c.length).toBeGreaterThan(0);
      await change.close();
    });
    it("should correctly propagate relationship changes to the ontology", async () => {
      const change = await client.ontology.openChangeTracker();
      const p = new Promise<ontology.RelationshipChange[]>((resolve) => {
        change.relationships.onChange((changes) => {
          resolve(changes);
        });
      });
      await client.ontology.groups.create(ontology.ROOT_ID, randomName());
      const c = await p;
      expect(c.length).toBeGreaterThan(0);
      await change.close();
    });
  });
});
