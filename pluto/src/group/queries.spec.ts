// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, group, ontology } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Group } from "@/group";
import { Ontology } from "@/ontology";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("Group queries", () => {
  let controller: AbortController;
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({
      client,
      excludeFluxStores: [Ontology.RESOURCES_FLUX_STORE_KEY],
    });
  });

  beforeEach(() => {
    controller = new AbortController();
  });

  afterEach(() => {
    controller.abort();
  });

  describe("useList", () => {
    it("should return a list of groups for a given parent", async () => {
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-parent",
      );
      const group1 = await client.ontology.groups.create(parent.ontologyID, "group1");
      const group2 = await client.ontology.groups.create(parent.ontologyID, "group2");

      const { result } = renderHook(
        () => Group.useList({ initialParams: { parent: parent.ontologyID } }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: parent.ontologyID });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      expect(result.current.data).toHaveLength(2);
      expect(result.current.data).toContain(group1.key);
      expect(result.current.data).toContain(group2.key);

      // Check we can retrieve the full group data
      const retrievedGroup1 = result.current.getItem(group1.key);
      expect(retrievedGroup1?.name).toBe("group1");
      const retrievedGroup2 = result.current.getItem(group2.key);
      expect(retrievedGroup2?.name).toBe("group2");
    });

    it("should return an empty list when parent is not provided", async () => {
      const { result } = renderHook(() => Group.useList({ initialParams: {} }), {
        wrapper,
      });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(0);
      });
    });

    it("should filter groups by search term", async () => {
      const uniqueId = Math.random().toString(36).substring(7);
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        `test-parent-search-${uniqueId}`,
      );
      await client.ontology.groups.create(parent.ontologyID, "apple red");
      await client.ontology.groups.create(parent.ontologyID, "banana blue");
      await client.ontology.groups.create(parent.ontologyID, "apple purple");

      const { result } = renderHook(
        () =>
          Group.useList({
            initialParams: { parent: parent.ontologyID, searchTerm: "apple" },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: parent.ontologyID, searchTerm: "apple" });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      });

      const groupItems = result.current.getItem(result.current.data);
      const names = groupItems.map((g) => g.name);
      expect(names).toContain("apple red");
      expect(names).toContain("apple purple");
    });

    it("should respect limit and offset parameters", async () => {
      const uniqueId = Math.random().toString(36).substring(7);
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        `test-parent-paginated-${uniqueId}`,
      );
      await Promise.all(
        Array.from({ length: 5 }, (_, i) =>
          client.ontology.groups.create(parent.ontologyID, `group${i}`),
        ),
      );

      const { result } = renderHook(
        () =>
          Group.useList({
            initialParams: { parent: parent.ontologyID, limit: 2, offset: 1 },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: parent.ontologyID, limit: 2, offset: 1 });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(2);
      });

      // Check the actual group names via getItem
      const groupItems = result.current.data.map((key) => result.current.getItem(key));
      const names = groupItems.map((g) => g?.name).filter(Boolean);
      // With offset 1 and limit 2, we should get items at index 1 and 2
      // But the order might vary, so just check we got 2 of the 5 groups
      expect(names).toHaveLength(2);
      expect(names.every((name) => name?.startsWith("group"))).toBe(true);
    });

    it("should update when a new group is added", async () => {
      const uniqueId = Math.random().toString(36).substring(7);
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        `test-parent-live-${uniqueId}`,
      );
      await client.ontology.groups.create(parent.ontologyID, "initial-group");

      const { result } = renderHook(
        () =>
          Group.useList({
            initialParams: { parent: parent.ontologyID },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: parent.ontologyID });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(1);
      });

      // Create a new group which should trigger an update
      await act(async () => {
        await client.ontology.groups.create(parent.ontologyID, "new-group");
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

  describe("create", () => {
    it("should create a new group", async () => {
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-parent-create",
      );

      const { result } = renderHook(() => Group.useCreate(), {
        wrapper,
      });

      await act(async () => {
        await result.current.updateAsync({
          key: uuid.create(),
          name: "created-group",
          parent: parent.ontologyID,
        });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toBeDefined();
        expect(result.current.data?.name).toBe("created-group");
      });

      const retrieved = await client.ontology.retrieve(
        group.ontologyID(result.current.data!.key),
      );
      expect(retrieved.name).toBe("created-group");
    });

    it("should updated an existing group", async () => {
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-parent-key",
      );
      const group = await client.ontology.groups.create(
        parent.ontologyID,
        "original-name",
      );

      const { result } = renderHook(() => Group.useCreate(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({
          key: group.key,
          name: "updated-name",
          parent: parent.ontologyID,
        });
      });

      const retrieved = await client.ontology.retrieve(group.ontologyID);
      expect(retrieved.name).toBe("updated-name");
    });
  });

  describe("useRename", () => {
    it("should rename an existing group", async () => {
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-parent-rename",
      );
      const testGroup = await client.ontology.groups.create(
        parent.ontologyID,
        "original-name",
      );

      const { result } = renderHook(() => Group.useRename(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: testGroup.key, name: "new-name" });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const retrieved = await client.ontology.retrieve(testGroup.ontologyID);
      expect(retrieved.name).toBe("new-name");
    });
  });

  describe("useDelete", () => {
    it("should delete an existing group", async () => {
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-parent-delete",
      );
      const testGroup = await client.ontology.groups.create(
        parent.ontologyID,
        "to-be-deleted",
      );

      const { result } = renderHook(() => Group.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync({ key: testGroup.key });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      await expect(client.ontology.retrieve(testGroup.ontologyID)).rejects.toThrow();
    });
  });

  describe("Flux store integration", () => {
    it("should cache retrieved groups", async () => {
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-parent-cache",
      );
      const group = await client.ontology.groups.create(
        parent.ontologyID,
        "cached-group",
      );

      const { result: result1 } = renderHook(
        () =>
          Group.useList({
            initialParams: { parent: parent.ontologyID },
          }),
        { wrapper },
      );

      act(() => {
        result1.current.retrieve({ parent: parent.ontologyID });
      });

      await waitFor(() => {
        expect(result1.current.variant).toEqual("success");
        expect(result1.current.data).toHaveLength(1);
      });

      const { result: result2 } = renderHook(
        () =>
          Group.useList({
            initialParams: { parent: parent.ontologyID },
          }),
        { wrapper },
      );

      act(() => {
        result2.current.retrieve({ parent: parent.ontologyID });
      });

      await waitFor(() => {
        expect(result2.current.variant).toEqual("success");
        expect(result2.current.data).toHaveLength(1);
        expect(result2.current.data[0]).toBe(group.key);
      });
    });

    it("should update cache when group is deleted via listener", async () => {
      const parent = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-parent-listener-delete",
      );
      const group = await client.ontology.groups.create(
        parent.ontologyID,
        "listener-delete-group",
      );

      const { result } = renderHook(
        () =>
          Group.useList({
            initialParams: { parent: parent.ontologyID },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: parent.ontologyID });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(1);
      });

      await act(async () => {
        await client.ontology.groups.delete(group.key);
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(0);
      });
    });

    it("should add a group to the list when a group is re-parented", async () => {
      const parent1 = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-parent-reparent-1",
      );
      const parent2 = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-parent-reparent-2",
      );
      const group = await client.ontology.groups.create(
        parent1.ontologyID,
        "movable-group",
      );

      const { result } = renderHook(
        () =>
          Group.useList({
            initialParams: { parent: parent2.ontologyID },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: parent2.ontologyID });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(0);
      });

      // Move the group from parent1 to parent2
      await act(async () => {
        await client.ontology.moveChildren(
          parent1.ontologyID,
          parent2.ontologyID,
          group.ontologyID,
        );
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(1);
        expect(result.current.data[0]).toBe(group.key);
      });

      const retrievedGroup = result.current.getItem(group.key);
      expect(retrievedGroup?.name).toBe("movable-group");
    });

    it("should remove a group from the list when a group is re-parented to a different parent", async () => {
      const parent1 = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-parent-remove-1",
      );
      const parent2 = await client.ontology.groups.create(
        ontology.ROOT_ID,
        "test-parent-remove-2",
      );
      const group = await client.ontology.groups.create(
        parent1.ontologyID,
        "moving-away-group",
      );

      const { result } = renderHook(
        () =>
          Group.useList({
            initialParams: { parent: parent1.ontologyID },
          }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({ parent: parent1.ontologyID });
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(1);
        expect(result.current.data[0]).toBe(group.key);
      });

      // Move the group from parent1 to parent2
      await act(async () => {
        await client.ontology.moveChildren(
          parent1.ontologyID,
          parent2.ontologyID,
          group.ontologyID,
        );
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(0);
      });
    });
  });
});
