// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type group, newTestClient, ontology } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it } from "vitest";

import { Ontology } from "@/ontology";
import { newSynnaxWrapper } from "@/testutil/Synnax";
const client = newTestClient();

describe("Ontology Queries", () => {
  describe("useChildren", async () => {
    it("should return children of a parent", async () => {
      const parent = await client.ontology.groups.create(ontology.ROOT_ID, "parent");
      const child1 = await client.ontology.groups.create(parent.ontologyID, "child1");
      const child2 = await client.ontology.groups.create(parent.ontologyID, "child2");
      await client.ontology.addChildren(
        parent.ontologyID,
        child1.ontologyID,
        child2.ontologyID,
      );

      const { result } = renderHook(
        () =>
          Ontology.useChildren({
            initialParams: { id: parent.ontologyID },
          }),
        { wrapper: newSynnaxWrapper(client) },
      );
      act(() => {
        result.current.retrieve({ id: parent.ontologyID });
      });
      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
      });
    });

    it("should update the query when a child is added to the parent", async () => {
      const parent = await client.ontology.groups.create(ontology.ROOT_ID, "parent");
      const child1 = await client.ontology.groups.create(parent.ontologyID, "child1");
      const child2 = await client.ontology.groups.create(parent.ontologyID, "child2");
      await client.ontology.addChildren(
        parent.ontologyID,
        child1.ontologyID,
        child2.ontologyID,
      );
      const { result } = renderHook(
        () =>
          Ontology.useChildren({
            initialParams: { id: parent.ontologyID },
          }),
        { wrapper: newSynnaxWrapper(client) },
      );
      act(() => {
        result.current.retrieve({ id: parent.ontologyID });
      });
      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
        expect(result.current.listenersMounted).toBe(true);
      });
      const child3 = await client.ontology.groups.create(parent.ontologyID, "child3");
      await client.ontology.addChildren(parent.ontologyID, child3.ontologyID);
      await waitFor(() => {
        expect(result.current.data).toHaveLength(3);
      });
    });

    it("should update the query when a child is removed from the parent", async () => {
      const parent = await client.ontology.groups.create(ontology.ROOT_ID, "parent");
      const child1 = await client.ontology.groups.create(parent.ontologyID, "child1");
      const child2 = await client.ontology.groups.create(parent.ontologyID, "child2");
      await client.ontology.addChildren(
        parent.ontologyID,
        child1.ontologyID,
        child2.ontologyID,
      );
      const { result } = renderHook(
        () =>
          Ontology.useChildren({
            initialParams: { id: parent.ontologyID },
          }),
        { wrapper: newSynnaxWrapper(client) },
      );
      act(() => {
        result.current.retrieve({ id: parent.ontologyID });
      });
      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
        expect(result.current.listenersMounted).toBe(true);
      });
      await client.ontology.removeChildren(parent.ontologyID, child1.ontologyID);
      await waitFor(() => {
        expect(result.current.data).toHaveLength(1);
      });
    });
  });

  describe("useResourceList", () => {
    it("should return all resources when no parameters are provided", async () => {
      await client.ontology.groups.create(ontology.ROOT_ID, "group1");
      await client.ontology.groups.create(ontology.ROOT_ID, "group2");

      const { result } = renderHook(() => Ontology.useResourceList({}), {
        wrapper: newSynnaxWrapper(client),
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
      const groups: group.Group[] = [];
      for (let i = 0; i < 5; i++)
        groups.push(await client.ontology.groups.create(ontology.ROOT_ID, `group${i}`));

      const groupIDStrings = groups.map((g) => ontology.idToString(g.ontologyID));

      const { result } = renderHook(
        () =>
          Ontology.useResourceList({
            filter: (r) => groupIDStrings.includes(ontology.idToString(r.id)),
          }),
        {
          wrapper: newSynnaxWrapper(client),
        },
      );

      act(() => {
        result.current.retrieve({
          limit: 2,
          offset: 1,
          types: ["group"],
          ids: groups.map((g) => g.ontologyID),
        });
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
      });
    });

    it("should filter resources by search term", async () => {
      await client.ontology.groups.create(ontology.ROOT_ID, "matching-group");
      await client.ontology.groups.create(ontology.ROOT_ID, "different-name");

      const { result } = renderHook(() => Ontology.useResourceList({}), {
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
      });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.listenersMounted).toBe(true);
      });

      const newGroupName = id.create();
      const newGroup = await client.ontology.groups.create(
        ontology.ROOT_ID,
        newGroupName,
      );

      await waitFor(() => {
        const item = result.current.getItem(newGroup.key);
        expect(item?.name).toBe(newGroupName);
      });
    });
  });
});
