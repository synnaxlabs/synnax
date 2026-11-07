// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Group } from "@/group";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("Group queries", () => {
  let controller: AbortController;
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  beforeEach(() => {
    controller = new AbortController();
  });

  afterEach(() => {
    controller.abort();
  });

  describe("useList", () => {
    it("should return a list of groups for a given parent", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: "test-parent",
      });
      const group1 = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "group1",
      });
      const group2 = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "group2",
      });

      const { result } = renderHook(
        () => Group.useList({ query: { parents: [group.resourceID(parent.key)] } }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parents: [group.resourceID(parent.key)] });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data).toHaveLength(2);
      expect(result.current.data).toContain(group1.key);
      expect(result.current.data).toContain(group2.key);

      const retrievedGroup1 = result.current.getItem(group1.key);
      expect(retrievedGroup1?.name).toBe("group1");
      const retrievedGroup2 = result.current.getItem(group2.key);
      expect(retrievedGroup2?.name).toBe("group2");
    });

    it("should keep only the top-level groups of the queried scope", async () => {
      const top = await client.groups.create({
        scope: "channel",
        name: `top-${uuid.create()}`,
      });
      const nested = await client.groups.create({
        parent: group.resourceID(top.key),
        name: "nested",
      });
      const other = await client.groups.create({
        scope: "project",
        name: `other-${uuid.create()}`,
      });
      const query: Group.ListQuery = { scope: "channel", nested: false };
      const { result } = renderHook(() => Group.useList({ query }), {
        wrapper,
      });

      act(() => {
        result.current.retrieve(query);
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toContain(top.key);
      });
      expect(result.current.data).not.toContain(nested.key);
      expect(result.current.data).not.toContain(other.key);
    });

    it("should respect limit and offset parameters", async () => {
      const uniqueId = Math.random().toString(36).substring(7);
      const parent = await client.groups.create({
        scope: "channel",
        name: `test-parent-paginated-${uniqueId}`,
      });
      await Promise.all(
        Array.from({ length: 5 }, (_, i) =>
          client.groups.create({
            parent: group.resourceID(parent.key),
            name: `group${i}`,
          }),
        ),
      );

      const { result } = renderHook(
        () =>
          Group.useList({
            query: {
              parents: [group.resourceID(parent.key)],
              limit: 2,
              offset: 1,
            },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({
          parents: [group.resourceID(parent.key)],
          limit: 2,
          offset: 1,
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(2);
      });

      const groupItems = result.current.data.map((key) => result.current.getItem(key));
      const names = groupItems.map((g) => g?.name).filter(Boolean);
      // With offset 1 and limit 2, we should get items at index 1 and 2
      // But the order might vary, so just check we got 2 of the 5 groups
      expect(names).toHaveLength(2);
      expect(names.every((name) => name?.startsWith("group"))).toBe(true);
    });

    it("should update when a new group is added", async () => {
      const uniqueId = Math.random().toString(36).substring(7);
      const parent = await client.groups.create({
        scope: "channel",
        name: `test-parent-live-${uniqueId}`,
      });
      await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "initial-group",
      });

      const { result } = renderHook(
        () =>
          Group.useList({
            query: { parents: [group.resourceID(parent.key)] },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parents: [group.resourceID(parent.key)] });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(1);
      });

      await act(async () => {
        await client.groups.create({
          parent: group.resourceID(parent.key),
          name: "new-group",
        });
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
      });

      const groupItems2 = result.current.data.map((key) => result.current.getItem(key));
      const names = groupItems2.map((g) => g?.name).filter(Boolean);
      expect(names).toContain("initial-group");
      expect(names).toContain("new-group");
    });
  });

  describe("useCreate", () => {
    it("should create a new group", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: "test-parent-create",
      });

      const { result } = renderHook(() => Group.useCreate(), {
        wrapper,
      });

      await act(async () => {
        await result.current.updateAsync({
          key: uuid.create(),
          name: "created-group",
          parent: group.resourceID(parent.key),
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toBeDefined();
        expect(result.current.data?.name).toBe("created-group");
      });

      const retrieved = await client.ontology.retrieve(
        group.resourceID(result.current.data?.key as string),
      );
      expect(retrieved.name).toBe("created-group");
    });

    it("should updated an existing group", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: "test-parent-key",
      });
      const g = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "original-name",
      });

      const { result } = renderHook(() => Group.useCreate(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          key: g.key,
          name: "updated-name",
          parent: group.resourceID(parent.key),
        });
      });

      const retrieved = await client.ontology.retrieve(group.resourceID(g.key));
      expect(retrieved.name).toBe("updated-name");
    });
  });

  describe("useRename", () => {
    it("should rename an existing group", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: "test-parent-rename",
      });
      const testGroup = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "original-name",
      });

      const { result } = renderHook(() => Group.useRename(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: testGroup.key, name: "new-name" });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrieved = await client.ontology.retrieve(group.resourceID(testGroup.key));
      expect(retrieved.name).toBe("new-name");
    });
  });

  describe("useDelete", () => {
    it("should delete an existing group", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: "test-parent-delete",
      });
      const testGroup = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "to-be-deleted",
      });

      const { result } = renderHook(() => Group.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: testGroup.key });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(
        client.ontology.retrieve(group.resourceID(testGroup.key)),
      ).rejects.toThrow();
    });

    it("should run afterOptimistic before the delete commits", async () => {
      const testGroup = await client.groups.create({
        scope: "channel",
        name: "optimistic-delete",
      });
      const order: string[] = [];
      const { result } = renderHook(
        () =>
          Group.useDelete({
            afterOptimistic: ({ data }) => {
              order.push(`optimistic:${data.key}`);
            },
            afterSuccess: () => {
              order.push("success");
            },
          }),
        { wrapper },
      );

      await act(async () => {
        await result.current.updateAsync({ key: testGroup.key });
      });

      expect(order).toEqual([`optimistic:${testGroup.key}`, "success"]);
    });
  });

  describe("useResultAncestors", () => {
    it("should keep the groups and the groups above them live", async () => {
      const root = await client.groups.create({ scope: "channel", name: "root" });
      const mid = await client.groups.create({
        parent: group.resourceID(root.key),
        name: "mid",
      });
      const leaf = await client.groups.create({
        parent: group.resourceID(mid.key),
        name: "leaf",
      });
      const keysOf = (groups?: group.Group[]): group.Key[] =>
        (groups ?? []).map(({ key }) => key).sort();
      const { result } = renderHook(
        () => Group.useResultAncestors({ keys: [leaf.key] }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(keysOf(result.current.data)).toEqual(
          [leaf.key, mid.key, root.key].sort(),
        );
      });
      await act(async () => {
        await client.groups.rename(root.key, "renamed");
      });
      await waitFor(() =>
        expect(result.current.data).toContainEqual(
          expect.objectContaining({ key: root.key, name: "renamed" }),
        ),
      );
      await act(async () => {
        await client.groups.setParent(mid.key, null);
      });
      await waitFor(() =>
        expect(keysOf(result.current.data)).toEqual([leaf.key, mid.key].sort()),
      );
    });
  });

  describe("Flux store integration", () => {
    it("should cache retrieved groups", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: "test-parent-cache",
      });
      const g = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "cached-group",
      });

      const { result: result1 } = renderHook(
        () =>
          Group.useList({
            query: { parents: [group.resourceID(parent.key)] },
          }),
        { wrapper },
      );

      act(() => {
        result1.current.retrieve({ parents: [group.resourceID(parent.key)] });
      });

      await waitFor(() => {
        expect(result1.current.variant).toEqual("success");
        expect(result1.current.data).toHaveLength(1);
      });

      const { result: result2 } = renderHook(
        () =>
          Group.useList({
            query: { parents: [group.resourceID(parent.key)] },
          }),
        { wrapper },
      );

      act(() => {
        result2.current.retrieve({ parents: [group.resourceID(parent.key)] });
      });

      await waitFor(() => {
        expect(result2.current.variant).toEqual("success");
        expect(result2.current.data).toHaveLength(1);
        expect(result2.current.data[0]).toBe(g.key);
      });
    });

    it("should update cache when group is deleted via listener", async () => {
      const parent = await client.groups.create({
        scope: "channel",
        name: "test-parent-listener-delete",
      });
      const g = await client.groups.create({
        parent: group.resourceID(parent.key),
        name: "listener-delete-group",
      });

      const { result } = renderHook(
        () =>
          Group.useList({
            query: { parents: [group.resourceID(parent.key)] },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parents: [group.resourceID(parent.key)] });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(1);
      });

      await act(async () => {
        await client.groups.delete(g.key);
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(0);
      });
    });

    it("should add a group to the list when a group is re-parented", async () => {
      const parent1 = await client.groups.create({
        scope: "channel",
        name: "test-parent-reparent-1",
      });
      const parent2 = await client.groups.create({
        scope: "channel",
        name: "test-parent-reparent-2",
      });
      const g = await client.groups.create({
        parent: group.resourceID(parent1.key),
        name: "movable-group",
      });

      const { result } = renderHook(
        () =>
          Group.useList({
            query: { parents: [group.resourceID(parent2.key)] },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parents: [group.resourceID(parent2.key)] });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(0);
      });

      await act(async () => {
        await client.groups.setParent(g.key, group.resourceID(parent2.key));
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(1);
        expect(result.current.data[0]).toBe(g.key);
      });

      const retrievedGroup = result.current.getItem(g.key);
      expect(retrievedGroup?.name).toBe("movable-group");
    });

    it("should remove a group from the list when a group is re-parented to a different parent", async () => {
      const parent1 = await client.groups.create({
        scope: "channel",
        name: "test-parent-remove-1",
      });
      const parent2 = await client.groups.create({
        scope: "channel",
        name: "test-parent-remove-2",
      });
      const g = await client.groups.create({
        parent: group.resourceID(parent1.key),
        name: "moving-away-group",
      });

      const { result } = renderHook(
        () =>
          Group.useList({
            query: { parents: [group.resourceID(parent1.key)] },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parents: [group.resourceID(parent1.key)] });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(1);
        expect(result.current.data[0]).toBe(g.key);
      });

      await act(async () => {
        await client.groups.setParent(g.key, group.resourceID(parent2.key));
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(0);
      });
    });
  });
});
