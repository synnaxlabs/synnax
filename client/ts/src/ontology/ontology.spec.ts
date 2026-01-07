// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { ontology } from "@/ontology";
import { group } from "@/group";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

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
      const g = await client.groups.create({ parent: ontology.ROOT_ID, name });
      const g2 = await client.ontology.retrieve(group.ontologyID(g.key));
      expect(g2.name).toEqual(name);
    });
    test("retrieve children", async () => {
      const name = randomName();
      const g = await client.groups.create({ parent: ontology.ROOT_ID, name });
      const name2 = randomName();
      await client.groups.create({
        parent: group.ontologyID(g.key),
        name: name2,
      });
      const children = await client.ontology.retrieveChildren(group.ontologyID(g.key));
      expect(children.length).toEqual(1);
      expect(children[0].name).toEqual(name2);
    });
    test("retrieve parents", async () => {
      const name = randomName();
      const g = await client.groups.create({ parent: ontology.ROOT_ID, name });
      const name2 = randomName();
      const g2 = await client.groups.create({
        parent: group.ontologyID(g.key),
        name: name2,
      });
      const parents = await client.ontology.retrieveParents(group.ontologyID(g2.key));
      expect(parents.length).toEqual(1);
      expect(parents[0].name).toEqual(name);
    });
  });

  describe("write", () => {
    test("add children", async () => {
      const name = randomName();
      const g = await client.groups.create({ parent: ontology.ROOT_ID, name });
      const name2 = randomName();
      const g2 = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: name2,
      });
      await client.ontology.addChildren(
        group.ontologyID(g.key),
        group.ontologyID(g2.key),
      );
      const children = await client.ontology.retrieveChildren(group.ontologyID(g.key));
      expect(children.length).toEqual(1);
      expect(children[0].name).toEqual(name2);
    });
    test("remove children", async () => {
      const name = randomName();
      const g = await client.groups.create({ parent: ontology.ROOT_ID, name });
      const name2 = randomName();
      const g2 = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: name2,
      });
      await client.ontology.addChildren(
        group.ontologyID(g.key),
        group.ontologyID(g2.key),
      );
      await client.ontology.removeChildren(
        group.ontologyID(g.key),
        group.ontologyID(g2.key),
      );
      const children = await client.ontology.retrieveChildren(group.ontologyID(g.key));
      expect(children.length).toEqual(0);
    });
    test("move children", async () => {
      const name = randomName();
      const g = await client.groups.create({ parent: ontology.ROOT_ID, name });
      const name2 = randomName();
      const g2 = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: name2,
      });
      const oldRootLength = (await client.ontology.retrieveChildren(ontology.ROOT_ID))
        .length;

      await client.ontology.moveChildren(
        ontology.ROOT_ID,
        group.ontologyID(g.key),
        group.ontologyID(g2.key),
      );

      const children = await client.ontology.retrieveChildren(group.ontologyID(g.key));
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
  });

  describe("matchRelationship", () => {
    const sampleRelationship: ontology.Relationship = {
      from: { type: "group", key: "test-group" },
      type: "parent",
      to: { type: "channel", key: "test-channel" },
    };

    describe("type matching", () => {
      it("should return true when types match", () => {
        const match: ontology.MatchRelationshipArgs = { type: "parent" };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return false when types don't match", () => {
        const match: ontology.MatchRelationshipArgs = { type: "child" };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });
    });

    describe("from ID matching", () => {
      it("should return true when from type matches", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { type: "group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when from key matches", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { key: "test-group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when both from type and key match", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { type: "group", key: "test-group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return false when from type doesn't match", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { type: "channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });

      it("should return false when from key doesn't match", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { key: "wrong-key" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });

      it("should return false when from type matches but key doesn't", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { type: "group", key: "wrong-key" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });
    });

    describe("to ID matching", () => {
      it("should return true when to type matches", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          to: { type: "channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when to key matches", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          to: { key: "test-channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when both to type and key match", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          to: { type: "channel", key: "test-channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return false when to type doesn't match", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          to: { type: "group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });

      it("should return false when to key doesn't match", () => {
        const match = { type: "parent", to: { key: "wrong-key" } };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });

      it("should return false when to type matches but key doesn't", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          to: { type: "channel", key: "wrong-key" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });
    });

    describe("combined matching", () => {
      it("should return true when all specified criteria match", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { type: "group", key: "test-group" },
          to: { type: "channel", key: "test-channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return false when type matches but from doesn't", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { type: "channel" },
          to: { type: "channel", key: "test-channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });

      it("should return false when type matches but to doesn't", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { type: "group", key: "test-group" },
          to: { type: "group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });

      it("should return false when from and to match but type doesn't", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "child",
          from: { type: "group", key: "test-group" },
          to: { type: "channel", key: "test-channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });
    });

    describe("partial matching", () => {
      it("should return true when only type is specified and matches", () => {
        const match: ontology.MatchRelationshipArgs = { type: "parent" };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when only from type is specified and matches", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { type: "group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when only to type is specified and matches", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          to: { type: "channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when only from key is specified and matches", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { key: "test-group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when only to key is specified and matches", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          to: { key: "test-channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });
    });

    describe("edge cases", () => {
      it("should handle empty from and to objects", () => {
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: {},
          to: {},
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should handle relationships with empty keys", () => {
        const relationship: ontology.Relationship = {
          from: { type: "group", key: "" },
          type: "parent",
          to: { type: "channel", key: "" },
        };
        const match: ontology.MatchRelationshipArgs = {
          type: "parent",
          from: { key: "" },
          to: { key: "" },
        };
        const result = ontology.matchRelationship(relationship, match);
        expect(result).toBe(true);
      });
    });
  });

  describe("idToString", () => {
    it("should convert an ID to a string", () => {
      const result = ontology.idToString({ type: "group", key: "one" });
      expect(result).toEqual("group:one");
    });

    it("should convert an array of IDs to strings", () => {
      const result = ontology.idToString([
        { type: "group", key: "one" },
        { type: "channel", key: "two" },
      ]);
      expect(result).toEqual(["group:one", "channel:two"]);
    });

    it("should pass through string IDs", () => {
      const result = ontology.idToString("group:one");
      expect(result).toEqual("group:one");
    });

    it("should validate string IDs that get passed", () => {
      expect(() => {
        ontology.idToString("dog");
      }).toThrow();
    });

    it("should pass through an array of string IDs", () => {
      const result = ontology.idToString(["group:one", "channel:two"]);
      expect(result).toEqual(["group:one", "channel:two"]);
    });

    it("should validate an array of string IDs that get passed", () => {
      expect(() => {
        ontology.idToString(["group:one", "channel:two", "dog"]);
      }).toThrow();
    });
  });
});
