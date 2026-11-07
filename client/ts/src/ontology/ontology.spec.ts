// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { id } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, onTestFinished, test, vi } from "vitest";

import { group } from "@/group";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { resource } from "@/resource";
import { createTestClient, expectLive } from "@/testutil";

const client = createTestClient();
const remote = createTestClient();

const randomName = (): string => `group-${Math.random()}`;

// Creates a group, linked under the parent by an ontology edge when one is given.
const createGroup = async (
  parent?: resource.ID,
  name: string = randomName(),
): Promise<resource.ID> => {
  const g = await client.groups.create({ scope: "channel", name });
  const gID = group.resourceID(g.key);
  if (parent != null) await client.ontology.addChildren(parent, gID);
  return gID;
};

describe("Ontology", () => {
  describe("parseIDs", () => {
    it("should parse a single ID object", () => {
      const id: resource.ID = { type: "group", key: "test-key" };
      const result = ontology.parseIDs(id);
      expect(result).toEqual([id]);
    });

    it("should parse an array of ID objects", () => {
      const ids: resource.ID[] = [
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
      const g = await client.groups.create({ scope: "channel", name });
      const g2 = await client.ontology.retrieve(group.resourceID(g.key));
      expect(g2.name).toEqual(name);
    });
    test("retrieve by string ID", async () => {
      const name = randomName();
      const g = await client.groups.create({ scope: "channel", name });
      const g2 = await client.ontology.retrieve(`group:${g.key}`);
      expect(g2.name).toEqual(name);
    });
    test("retrieve children", async () => {
      const parent = await createGroup();
      const name = randomName();
      await createGroup(parent, name);
      const children = await client.ontology.children.retrieve({ ids: parent });
      expect(children.map((c) => c.name)).toEqual([name]);
    });
    test("retrieve parents", async () => {
      const name = randomName();
      const parent = await createGroup(undefined, name);
      const child = await createGroup(parent);
      const parents = await client.ontology.parents.retrieve({ ids: child });
      expect(parents.map((p) => p.name)).toEqual([name]);
    });
    test("retrieve by search term", async () => {
      const name = randomName();
      const g = await client.groups.create({ scope: "channel", name });
      await expect
        .poll(async () => {
          const results = await client.ontology.retrieve({ searchTerm: name });
          return results.find((r) => r.id.key === g.key);
        })
        .toMatchObject({ name, id: { type: "group", key: g.key } });
    });
    test("retrieve by search term under a root", async () => {
      const name = randomName();
      const root = await createGroup();
      const inside = await createGroup(root, name);
      await createGroup(await createGroup(), name);
      await expect
        .poll(async () =>
          (
            await client.ontology.retrieve({ searchTerm: name, root, types: ["group"] })
          ).map((r) => r.id.key),
        )
        .toEqual([inside.key]);
    });
    test("refetch a subscribed search under a root when a parent link changes", async () => {
      const match: ontology.Resource = {
        key: "group:match",
        id: { type: "group", key: "match" },
        name: "match",
      };
      let answer = [match];
      const unary: UnaryClient = {
        use: () => {},
        send: async (_target, _req, _reqZ, resZ) => resZ.parse({ resources: answer }),
      };
      const cache = new query.Cache({ openStreamer: null, onError: () => {} });
      onTestFinished(async () => await cache.close());
      const ont = new ontology.Client({ unary, cache });
      const root: resource.ID = { type: "group", key: "root" };
      const params = { searchTerm: "match", root, types: ["group" as const] };
      vi.useFakeTimers();
      onTestFinished(() => {
        vi.useRealTimers();
      });
      const off = ont.onChange(params, () => {});
      try {
        expect(await ont.retrieve(params)).toEqual([match]);
        await vi.runAllTimersAsync();
        answer = [];
        const moved: ontology.Relationship = {
          from: { type: "group", key: "other" },
          type: "parent",
          to: match.id,
        };
        ont.cache.relationships.set(ontology.relationshipToString(moved), moved);
        await vi.runAllTimersAsync();
        expect(ont.getCached(params)).toEqual([]);
      } finally {
        off();
      }
    });
    test("retrieve by search term for the given types only", async () => {
      const name = randomName();
      const g = await client.groups.create({ scope: "channel", name });
      const found = async (types: resource.Type[]): Promise<boolean> =>
        (await client.ontology.retrieve({ searchTerm: name, types })).some(
          (r) => r.id.key === g.key,
        );
      await expect.poll(async () => await found(["group"])).toBe(true);
      expect(await found(["channel"])).toBe(false);
    });
    test("page through search results with a limit and an offset", async () => {
      const token = `page${Math.random().toString(36).slice(2)}`;
      const root = await createGroup();
      for (const suffix of ["a", "b", "c"])
        await createGroup(root, `${token} ${suffix}`);
      const search = async (offset: number): Promise<number> =>
        (
          await client.ontology.retrieve({
            searchTerm: token,
            root,
            limit: 2,
            offset,
          })
        ).length;
      await expect.poll(async () => await search(0)).toEqual(2);
      expect(await search(2)).toEqual(1);
    });
  });

  describe("ancestors", () => {
    const createChain = async (): Promise<{
      top: resource.ID;
      mid: resource.ID;
      leaf: resource.ID;
    }> => {
      const top = await createGroup(ontology.ROOT_ID);
      const mid = await createGroup(top);
      const leaf = await createGroup(mid);
      return { top, mid, leaf };
    };

    // Resolves once the stream delivers a change made after every earlier write, so no
    // late echo of those writes lands during a spec.
    const drainStream = async (): Promise<void> => {
      await client.connect();
      const probe = await remote.groups.create({
        scope: "channel",
        name: randomName(),
      });
      const probeID = group.resourceID(probe.key);
      await remote.ontology.addChildren(ontology.ROOT_ID, probeID);
      const link = ontology.relationshipToString({
        from: ontology.ROOT_ID,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: probeID,
      });
      await expect
        .poll(() => client.ontology.cache.relationships.get(link))
        .toBeDefined();
      await expect
        .poll(() => client.ontology.cache.resources.get(resource.idToString(probeID)))
        .toBeDefined();
    };

    it("should return the resources and the path above each ID", async () => {
      const { top, mid, leaf } = await createChain();
      const { resources, paths } = await client.ontology.ancestors.retrieve({
        ids: leaf,
      });
      expect(resources.map((r) => r.key).sort()).toEqual(
        resource.idToString([top, mid]).sort(),
      );
      expect(paths[resource.idToString(leaf)]).toEqual([[top, mid]]);
    });

    it("should cache the resources of the IDs", async () => {
      const { leaf } = await createChain();
      await drainStream();
      const key = resource.idToString(leaf);
      client.ontology.cache.resources.evict(key);
      await client.ontology.ancestors.retrieve({ ids: leaf });
      expect(client.ontology.cache.resources.get(key)).toMatchObject({ id: leaf });
    });

    it("should stop the walk at the root", async () => {
      const { top, mid, leaf } = await createChain();
      const { resources, paths } = await client.ontology.ancestors.retrieve({
        ids: leaf,
        root: top,
      });
      expect(resources.map((r) => r.key)).toEqual([resource.idToString(mid)]);
      expect(paths[resource.idToString(leaf)]).toEqual([[mid]]);
    });

    it("should return a path for each parent", async () => {
      const { top, mid, leaf } = await createChain();
      const sideID = await createGroup(top);
      await client.ontology.addChildren(sideID, leaf);
      const { paths } = await client.ontology.ancestors.retrieve({
        ids: leaf,
        root: top,
      });
      expect(paths[resource.idToString(leaf)]).toEqual(
        expect.arrayContaining([[mid], [sideID]]),
      );
      expect(paths[resource.idToString(leaf)]).toHaveLength(2);
    });

    it("should drop a cached parent link that the Core no longer has", async () => {
      const { top, mid, leaf } = await createChain();
      const side = await createGroup(top);
      // A link whose delete the cache missed, as across a reconnect.
      const stale: ontology.Relationship = {
        from: side,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: leaf,
      };
      client.ontology.cache.relationships.set(
        ontology.relationshipToString(stale),
        stale,
      );
      const { paths } = await client.ontology.ancestors.retrieve({
        ids: leaf,
        root: top,
      });
      expect(paths[resource.idToString(leaf)]).toEqual([[mid]]);
    });

    it("should take back a dropped parent link that another client adds", async () => {
      const { top, leaf } = await createChain();
      const side = await client.groups.create({ parent: top, name: randomName() });
      await drainStream();
      const stale: ontology.Relationship = {
        from: group.resourceID(side.key),
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: leaf,
      };
      const key = ontology.relationshipToString(stale);
      const { relationships } = client.ontology.cache;
      relationships.set(key, stale);
      await client.ontology.ancestors.retrieve({ ids: leaf, root: top });
      expect(relationships.status(key)).toBe("tombstoned");
      await remote.ontology.addChildren(stale.from, leaf);
      await expect.poll(() => relationships.status(key)).toBe("present");
    });

    it("should keep a link deleted while the request is in flight", async () => {
      const { top, mid, leaf } = await createChain();
      const key = ontology.relationshipToString({
        from: mid,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: leaf,
      });
      await drainStream();
      const request = client.ontology.ancestors.retrieve({ ids: leaf, root: top });
      client.ontology.cache.relationships.delete(key);
      await request;
      expect(client.ontology.cache.relationships.get(key)).toBeUndefined();
    });

    it("should keep a rename written while the request is in flight", async () => {
      const { top, mid, leaf } = await createChain();
      const resource = await client.ontology.retrieve(mid);
      await drainStream();
      const request = client.ontology.ancestors.retrieve({ ids: leaf, root: top });
      client.ontology.cache.resources.set({ ...resource, name: "renamed" });
      await request;
      expect(client.ontology.cache.resources.get(resource.key)?.name).toBe("renamed");
    });

    describe("onChange", () => {
      beforeAll(async () => await client.connect());

      it("should follow a remote move", async () => {
        const { top, mid, leaf } = await createChain();
        const sideID = await createGroup(top);
        const params = { ids: leaf, root: top };
        const handler = vi.fn();
        const off = client.ontology.ancestors.onChange(params, handler);
        try {
          await client.ontology.ancestors.retrieve(params);
          await remote.ontology.moveChildren(mid, sideID, leaf);
          await expect
            .poll(() => {
              const cached = client.ontology.ancestors.getCached(params);
              return query.isLive(cached)
                ? cached.paths[resource.idToString(leaf)]
                : null;
            })
            .toEqual([[sideID]]);
          expect(handler).toHaveBeenCalled();
        } finally {
          off();
        }
      });

      it("should follow a remote move to the root", async () => {
        const { top, mid, leaf } = await createChain();
        const params = { ids: leaf, root: top };
        const off = client.ontology.ancestors.onChange(params, () => {});
        try {
          await client.ontology.ancestors.retrieve(params);
          await remote.ontology.moveChildren(mid, top, leaf);
          await expect
            .poll(() => {
              const cached = client.ontology.ancestors.getCached(params);
              return query.isLive(cached)
                ? cached.paths[resource.idToString(leaf)]
                : null;
            })
            .toEqual([[]]);
        } finally {
          off();
        }
      });

      it("should follow a remote group rename", async () => {
        const { top, mid, leaf } = await createChain();
        const params = { ids: leaf, root: top };
        const off = client.ontology.ancestors.onChange(params, () => {});
        try {
          await client.ontology.ancestors.retrieve(params);
          const name = randomName();
          await remote.groups.rename(mid.key, name);
          await expect
            .poll(() => {
              const cached = client.ontology.ancestors.getCached(params);
              return query.isLive(cached) ? cached.resources[0]?.name : null;
            })
            .toEqual(name);
        } finally {
          off();
        }
      });
    });
  });

  describe("write", () => {
    test("add children", async () => {
      const name = randomName();
      const g = await client.groups.create({ scope: "channel", name });
      const name2 = randomName();
      const g2 = await client.groups.create({
        scope: "channel",
        name: name2,
      });
      await client.ontology.addChildren(
        group.resourceID(g.key),
        group.resourceID(g2.key),
      );
      const children = await client.ontology.children.retrieve({
        ids: group.resourceID(g.key),
      });
      expect(children.length).toEqual(1);
      expect(children[0].name).toEqual(name2);
    });
    test("remove children", async () => {
      const name = randomName();
      const g = await client.groups.create({ scope: "channel", name });
      const name2 = randomName();
      const g2 = await client.groups.create({
        scope: "channel",
        name: name2,
      });
      await client.ontology.addChildren(
        group.resourceID(g.key),
        group.resourceID(g2.key),
      );
      await client.ontology.removeChildren(
        group.resourceID(g.key),
        group.resourceID(g2.key),
      );
      const children = await client.ontology.children.retrieve({
        ids: group.resourceID(g.key),
      });
      expect(children.length).toEqual(0);
    });
    test("move children", async () => {
      const from = await createGroup();
      const to = await createGroup();
      const child = await createGroup(from);
      await client.ontology.moveChildren(from, to, child);
      const childKeys = async (ids: resource.ID): Promise<string[]> =>
        (await client.ontology.children.retrieve({ ids })).map(({ key }) => key);
      expect(await childKeys(to)).toEqual([resource.idToString(child)]);
      expect(await childKeys(from)).toEqual([]);
    });
  });

  describe("cached reads", () => {
    // Changes made while the stream is still opening are lost (epoch
    // reconciliation repairs them in production); open it up front so remote
    // writes in these specs are always observed.
    beforeAll(async () => await client.connect());

    describe("children", () => {
      it("reflects remote child changes on an unsubscribed repeat retrieve", async () => {
        const parent = await client.groups.create({
          scope: "channel",
          name: randomName(),
        });
        const parentID = group.resourceID(parent.key);
        const child = await client.groups.create({
          scope: "channel",
          name: randomName(),
        });
        const childID = group.resourceID(child.key);
        expect(await client.ontology.children.retrieve({ ids: parentID })).toHaveLength(
          0,
        );
        await remote.ontology.addChildren(parentID, childID);
        // Unsubscribed queries hold no frozen answer: repeat retrieves refetch
        // and converge on the remote change once it streams in.
        await expect
          .poll(async () =>
            (await client.ontology.children.retrieve({ ids: parentID })).some(
              (r) => r.id.key === child.key,
            ),
          )
          .toBe(true);
        await remote.ontology.removeChildren(parentID, childID);
        await expect
          .poll(async () =>
            (await client.ontology.children.retrieve({ ids: parentID })).some(
              (r) => r.id.key === child.key,
            ),
          )
          .toBe(false);
      });
    });

    describe("getCached", () => {
      it("returns undefined before a children retrieve and the answer after", async () => {
        const parentID = await createGroup();
        const child = await createGroup(parentID);
        const params = { ids: parentID };
        expect(client.ontology.children.getCached(params)).toBeUndefined();
        await client.ontology.children.retrieve(params);
        const cached = expectLive(client.ontology.children.getCached(params));
        expect(cached.some((r) => r.id.key === child.key)).toBe(true);
      });
    });

    describe("onChange", () => {
      it("delivers remotely added children through relationship composition", async () => {
        const parentID = await createGroup();
        const params = { ids: parentID };
        const handler = vi.fn();
        const off = client.ontology.children.onChange(params, handler);
        try {
          expect(await client.ontology.children.retrieve(params)).toHaveLength(0);
          const child = await remote.groups.create({
            scope: "channel",
            name: randomName(),
          });
          await remote.ontology.addChildren(parentID, group.resourceID(child.key));
          await expect
            .poll(() => {
              const cached = client.ontology.children.getCached(params);
              return query.isLive(cached) && cached.some((r) => r.id.key === child.key);
            })
            .toBe(true);
          expect(handler).toHaveBeenCalled();
        } finally {
          off();
        }
      });
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
        const match: ontology.MatchRelationshipParams = { type: "parent" };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return false when types don't match", () => {
        const match: ontology.MatchRelationshipParams = { type: "child" };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });
    });

    describe("from ID matching", () => {
      it("should return true when from type matches", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { type: "group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when from key matches", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { key: "test-group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when both from type and key match", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { type: "group", key: "test-group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return false when from type doesn't match", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { type: "channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });

      it("should return false when from key doesn't match", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { key: "wrong-key" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });

      it("should return false when from type matches but key doesn't", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { type: "group", key: "wrong-key" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });
    });

    describe("to ID matching", () => {
      it("should return true when to type matches", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          to: { type: "channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when to key matches", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          to: { key: "test-channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when both to type and key match", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          to: { type: "channel", key: "test-channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return false when to type doesn't match", () => {
        const match: ontology.MatchRelationshipParams = {
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
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          to: { type: "channel", key: "wrong-key" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });
    });

    describe("combined matching", () => {
      it("should return true when all specified criteria match", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { type: "group", key: "test-group" },
          to: { type: "channel", key: "test-channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return false when type matches but from doesn't", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { type: "channel" },
          to: { type: "channel", key: "test-channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });

      it("should return false when type matches but to doesn't", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { type: "group", key: "test-group" },
          to: { type: "group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(false);
      });

      it("should return false when from and to match but type doesn't", () => {
        const match: ontology.MatchRelationshipParams = {
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
        const match: ontology.MatchRelationshipParams = { type: "parent" };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when only from type is specified and matches", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { type: "group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when only to type is specified and matches", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          to: { type: "channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when only from key is specified and matches", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { key: "test-group" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });

      it("should return true when only to key is specified and matches", () => {
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          to: { key: "test-channel" },
        };
        const result = ontology.matchRelationship(sampleRelationship, match);
        expect(result).toBe(true);
      });
    });

    describe("edge cases", () => {
      it("should handle empty from and to objects", () => {
        const match: ontology.MatchRelationshipParams = {
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
        const match: ontology.MatchRelationshipParams = {
          type: "parent",
          from: { key: "" },
          to: { key: "" },
        };
        const result = ontology.matchRelationship(relationship, match);
        expect(result).toBe(true);
      });
    });
  });
});

describe("store", () => {
  it("caches resource sets and corpses deletes from live signals", async () => {
    await client.connect();
    const label = await client.labels.create({
      name: `cache-test-${id.create()}`,
      color: "#E774D0",
    });
    const resourceKey = resource.idToString({ type: "label", key: label.key });
    await expect
      .poll(() => client.ontology.cache.resources.get(resourceKey))
      .toBeDefined();
    expect(client.ontology.cache.resources.status(resourceKey)).toBe("present");
    await client.labels.delete(label.key);
    await expect
      .poll(() => client.ontology.cache.resources.status(resourceKey))
      .toBe("tombstoned");
    const tombstone = client.ontology.cache.resources.getTombstone(resourceKey);
    expect(tombstone?.corpse.name).toEqual(label.name);
  });

  it("stays a detached, local-only cache when caching is disabled", async () => {
    const disabled = createTestClient({ cache: false });
    expect(() => disabled.ontology.cache.resources).not.toThrow();
    expect(disabled.connection.status.details.epoch).toBe(0);
    // Detached: connecting succeeds without a change stream, so the epoch
    // never advances past 0.
    await disabled.connect();
    expect(disabled.connection.status.details.epoch).toBe(0);
    await disabled.close();
  });
});

describe("deleteRelationships", () => {
  const groupID = (key: string): resource.ID => ({ type: "group", key });
  const rel = (from: resource.ID, to: resource.ID): ontology.Relationship => ({
    from,
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to,
  });
  const createCache = (rels: ontology.Relationship[]): ontology.Cache => {
    const root = new query.Cache({ openStreamer: null });
    const indexes = {
      byTo: new query.LookupIndex<string, ontology.Relationship>((r) =>
        resource.idToString(r.to),
      ),
      byFrom: new query.LookupIndex<string, ontology.Relationship>((r) =>
        resource.idToString(r.from),
      ),
    };
    const relationships = root.createTable<string, ontology.Relationship>({
      name: "relationships",
      indexes: [indexes.byTo, indexes.byFrom],
    });
    const resources = root.createTable<string, ontology.Resource>({
      name: "resources",
    });
    rels.forEach((r) => relationships.set(ontology.relationshipToString(r), r));
    return new ontology.Cache(relationships, resources, indexes);
  };

  const [a, b, c, d] = ["a", "b", "c", "d"].map(groupID);

  it("deletes every cached relationship touching the given IDs", () => {
    const cache = createCache([rel(a, b), rel(b, c), rel(c, d)]);
    cache.deleteRelationships(b);
    expect(cache.relationshipsTo(b)).toHaveLength(0);
    expect(cache.relationshipsFrom(b)).toHaveLength(0);
    expect(cache.relationshipsFrom(a)).toHaveLength(0);
    expect(cache.relationshipsFrom(c)).toHaveLength(1);
  });

  it("returns a rollback restoring them, a shared relationship included once", () => {
    const cache = createCache([rel(a, b), rel(b, c)]);
    const undo = cache.deleteRelationships([a, b]);
    expect(cache.relationshipsFrom(a)).toHaveLength(0);
    expect(cache.relationshipsTo(c)).toHaveLength(0);
    undo();
    expect(cache.relationshipsFrom(a)).toHaveLength(1);
    expect(cache.relationshipsTo(b)).toHaveLength(1);
    expect(cache.relationshipsTo(c)).toHaveLength(1);
  });

  it("deletes by collected keys, never by predicate", () => {
    const cache = createCache([rel(a, b), rel(b, c)]);
    const spy = vi.spyOn(cache.relationships, "delete");
    cache.deleteRelationships(b);
    for (const [arg] of spy.mock.calls) expect(typeof arg).not.toBe("function");
    spy.mockRestore();
  });
});
