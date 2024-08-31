// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { ontology } from "@/ontology";
import { newClient } from "@/setupspecs";

const client = newClient();

const randomName = (): string => `group-${Math.random()}`;

describe("Ontology", () => {
  describe("retrieve", () => {
    test("retrieve", async () => {
      const name = randomName();
      const g = await client.ontology.groups.create(ontology.Root, name);
      const g2 = await client.ontology.retrieve(g.ontologyID);
      expect(g2.name).toEqual(name);
    });
    test("retrieve children", async () => {
      const name = randomName();
      const g = await client.ontology.groups.create(ontology.Root, name);
      const name2 = randomName();
      await client.ontology.groups.create(g.ontologyID, name2);
      const children = await client.ontology.retrieveChildren(g.ontologyID);
      expect(children.length).toEqual(1);
      expect(children[0].name).toEqual(name2);
    });
    test("retrieve parents", async () => {
      const name = randomName();
      const g = await client.ontology.groups.create(ontology.Root, name);
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
        await client.ontology.groups.create(ontology.Root, randomName());
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
      const g = await client.ontology.groups.create(ontology.Root, name);
      const name2 = randomName();
      const g2 = await client.ontology.groups.create(ontology.Root, name2);
      await client.ontology.addChildren(g.ontologyID, g2.ontologyID);
      const children = await client.ontology.retrieveChildren(g.ontologyID);
      expect(children.length).toEqual(1);
      expect(children[0].name).toEqual(name2);
    });
    test("remove children", async () => {
      const name = randomName();
      const g = await client.ontology.groups.create(ontology.Root, name);
      const name2 = randomName();
      const g2 = await client.ontology.groups.create(ontology.Root, name2);
      await client.ontology.addChildren(g.ontologyID, g2.ontologyID);
      await client.ontology.removeChildren(g.ontologyID, g2.ontologyID);
      const children = await client.ontology.retrieveChildren(g.ontologyID);
      expect(children.length).toEqual(0);
    });
    test("move children", async () => {
      const name = randomName();
      const g = await client.ontology.groups.create(ontology.Root, name);
      const name2 = randomName();
      const g2 = await client.ontology.groups.create(ontology.Root, name2);
      const oldRootLength = (await client.ontology.retrieveChildren(ontology.Root))
        .length;
      await client.ontology.moveChildren(ontology.Root, g.ontologyID, g2.ontologyID);
      const children = await client.ontology.retrieveChildren(g.ontologyID);
      expect(children.length).toEqual(1);
      const newRootLength = (await client.ontology.retrieveChildren(ontology.Root))
        .length;
      expect(newRootLength).toEqual(oldRootLength - 1);
    });
  });
  describe("signals", async () => {
    it("should correctly decode a set of relationships from a string", () => {
      const rel = ontology.parseRelationship("typeA:keyA->parent->typeB:keyB");
      expect(rel.type).toEqual("parent");
      expect(rel.from.type).toEqual("typeA");
      expect(rel.from.key).toEqual("keyA");
      expect(rel.to.type).toEqual("typeB");
      expect(rel.to.key).toEqual("keyB");
    });
    it("should correctly propagate resource changes to the ontology", async () => {
      const change = await client.ontology.openChangeTracker();
      const p = new Promise<ontology.ResourceChange[]>((resolve) =>
        change.resources.onChange((changes) => resolve(changes)),
      );
      await client.ontology.groups.create(ontology.Root, randomName());
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
      await client.ontology.groups.create(ontology.Root, randomName());
      const c = await p;
      expect(c.length).toBeGreaterThan(0);
      await change.close();
    });
  });
});
