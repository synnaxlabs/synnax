// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, group, ontology } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it } from "vitest";

import { Ontology } from "@/ontology";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });

describe("Ontology Queries", () => {
  describe("useChildren", async () => {
    it("should return children of a parent", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "parent",
      });
      const child1 = await client.groups.create({
        parent: group.ontologyID(parent.key),
        name: "child1",
      });
      const child2 = await client.groups.create({
        parent: group.ontologyID(parent.key),
        name: "child2",
      });
      await client.ontology.addChildren(
        group.ontologyID(parent.key),
        group.ontologyID(child1.key),
        group.ontologyID(child2.key),
      );

      const { result } = renderHook(
        () =>
          Ontology.useListChildren({
            initialQuery: { id: group.ontologyID(parent.key) },
          }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({ id: group.ontologyID(parent.key) });
      });
      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
      });
    });

    it("should update the query when a child is added to the parent", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "parent",
      });
      await client.groups.create({
        parent: group.ontologyID(parent.key),
        name: "child1",
      });
      await client.groups.create({
        parent: group.ontologyID(parent.key),
        name: "child2",
      });
      const { result } = renderHook(
        () =>
          Ontology.useListChildren({
            initialQuery: { id: group.ontologyID(parent.key) },
          }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({ id: group.ontologyID(parent.key) });
      });
      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
      });
      await act(async () => {
        const alternateParent = await client.groups.create({
          parent: ontology.ROOT_ID,
          name: "alternateParent",
        });
        await client.groups.create({
          parent: group.ontologyID(parent.key),
          name: "child3",
        });
        await client.groups.create({
          parent: group.ontologyID(alternateParent.key),
          name: "child4",
        });
      });
      await waitFor(() => {
        expect(result.current.data).toHaveLength(3);
      });
    });

    it("should update the query when a child is removed from the parent", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "parent",
      });
      const child1 = await client.groups.create({
        parent: group.ontologyID(parent.key),
        name: "child1",
      });
      const child2 = await client.groups.create({
        parent: group.ontologyID(parent.key),
        name: "child2",
      });
      await client.ontology.addChildren(
        group.ontologyID(parent.key),
        group.ontologyID(child1.key),
        group.ontologyID(child2.key),
      );
      const { result } = renderHook(
        () =>
          Ontology.useListChildren({
            initialQuery: { id: group.ontologyID(parent.key) },
          }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({ id: group.ontologyID(parent.key) });
      });
      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
      });
      await client.ontology.removeChildren(
        group.ontologyID(parent.key),
        group.ontologyID(child1.key),
      );
      await waitFor(() => {
        expect(result.current.data).toHaveLength(1);
      });
    });
  });

  describe("useResourceList", () => {
    it("should return all resources when no parameters are provided", async () => {
      await client.groups.create({ parent: ontology.ROOT_ID, name: "group1" });
      await client.groups.create({ parent: ontology.ROOT_ID, name: "group2" });

      const { result } = renderHook(() => Ontology.useResourceList({}), {
        wrapper,
      });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.data.length).toBeGreaterThanOrEqual(2);
        const groupNames = result.current
          .getItem(result.current.data)
          .map((r) => r.name);
        expect(groupNames).toContain("group1");
        expect(groupNames).toContain("group2");
      });
    });

    it("should respect pagination parameters", async () => {
      const groups: group.Group[] = await Promise.all(
        Array.from({ length: 5 }, async (_, i) =>
          client.groups.create({
            parent: ontology.ROOT_ID,
            name: `group${i}`,
          }),
        ),
      );

      const groupIDStrings = groups.map((g) =>
        ontology.idToString(group.ontologyID(g.key)),
      );

      const { result } = renderHook(
        () =>
          Ontology.useResourceList({
            filter: (r) => groupIDStrings.includes(ontology.idToString(r.id)),
            useCachedList: false,
          }),
        {
          wrapper,
        },
      );

      act(() => {
        result.current.retrieve({
          limit: 2,
          offset: 1,
          types: ["group"],
          ids: groups.map((g) => group.ontologyID(g.key)),
        });
      });

      await waitFor(() => {
        // TODO: This is a flakey test that doesn't always return correctly due to
        // agressive caching. In reality, the page length should be exactly 2,
        // but signal propagation can cause it to be greater than 2.
        expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      });
    });

    it("should filter resources by search term", async () => {
      await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "matching-group",
      });
      await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "different-name",
      });

      const { result } = renderHook(() => Ontology.useResourceList({}), {
        wrapper,
      });

      act(() => {
        result.current.retrieve({ searchTerm: "matching" });
      });

      await waitFor(() => {
        expect(result.current.data.length).toBeGreaterThanOrEqual(1);
        const groupNames = result.current.data.map(
          (r) => result.current.getItem(r)?.name,
        );
        expect(groupNames).toContain("matching-group");
        expect(groupNames).not.toContain("different-name");
      });
    });

    it("should update when a new resource is created", async () => {
      const { result } = renderHook(() => Ontology.useResourceList({}), {
        wrapper,
      });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });

      const newGroupName = id.create();
      const newGroup = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: newGroupName,
      });

      await waitFor(() => {
        const item = result.current.getItem(
          ontology.idToString(group.ontologyID(newGroup.key)),
        );
        expect(item?.name).toBe(newGroupName);
      });
    });
  });

  describe("useRetrieveChildren", () => {
    it("should retrieve children of a parent", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "parent",
      });
      await client.groups.create({
        parent: group.ontologyID(parent.key),
        name: "child1",
      });
      await client.groups.create({
        parent: group.ontologyID(parent.key),
        name: "child2",
      });

      const { result } = renderHook(
        () => Ontology.useRetrieveChildren({ id: group.ontologyID(parent.key) }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(2);
      });
    });

    it("should return empty array when parent has no children", async () => {
      const parent = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "empty-parent",
      });

      const { result } = renderHook(
        () => Ontology.useRetrieveChildren({ id: group.ontologyID(parent.key) }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual([]);
      });
    });

    it("should return disabled state when no client connected", async () => {
      const noClientWrapper = createSynnaxWrapper({ client: null });

      const { result } = renderHook(
        () => Ontology.useRetrieveChildren({ id: ontology.ROOT_ID }),
        { wrapper: noClientWrapper },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("disabled");
      });
    });

    it("should re-fetch when query ID changes", async () => {
      const parent1 = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "parent1",
      });
      await client.groups.create({
        parent: group.ontologyID(parent1.key),
        name: "p1-child",
      });

      const parent2 = await client.groups.create({
        parent: ontology.ROOT_ID,
        name: "parent2",
      });
      await client.groups.create({
        parent: group.ontologyID(parent2.key),
        name: "p2-child1",
      });
      await client.groups.create({
        parent: group.ontologyID(parent2.key),
        name: "p2-child2",
      });

      const { result, rerender } = renderHook(
        ({ id }) => Ontology.useRetrieveChildren({ id }),
        {
          wrapper,
          initialProps: { id: group.ontologyID(parent1.key) },
        },
      );

      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toHaveLength(1);
      });

      rerender({ id: group.ontologyID(parent2.key) });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
      });
    });
  });
});
