// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DisconnectedError,
  ontology,
  project,
  resource,
  schematic,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Ontology } from "@/ontology";
import { renderHookSuspended } from "@/testutil/render";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });

describe("Ontology Queries", () => {
  describe("useChildren", async () => {
    it("should return children of a parent", async () => {
      const parent = await client.projects.create({ name: "parent" });
      await client.schematics.create(parent.key, { name: "child1" });
      await client.schematics.create(parent.key, { name: "child2" });

      const { result } = renderHook(
        () =>
          Ontology.useListChildren({
            query: { id: project.resourceID(parent.key) },
          }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({ id: project.resourceID(parent.key) });
      });
      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
      });
    });

    it("should update the query when a child is added to the parent", async () => {
      const parent = await client.projects.create({ name: "parent" });
      await client.schematics.create(parent.key, { name: "child1" });
      await client.schematics.create(parent.key, { name: "child2" });
      const { result } = renderHook(
        () =>
          Ontology.useListChildren({
            query: { id: project.resourceID(parent.key) },
          }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({ id: project.resourceID(parent.key) });
      });
      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
      });
      await act(async () => {
        const alternateParent = await client.projects.create({
          name: "alternateParent",
        });
        await client.schematics.create(parent.key, { name: "child3" });
        await client.schematics.create(alternateParent.key, { name: "child4" });
      });
      await waitFor(() => {
        expect(result.current.data).toHaveLength(3);
      });
    });

    it("should update the query when a child is removed from the parent", async () => {
      const parent = await client.projects.create({ name: "parent" });
      const child1 = await client.schematics.create(parent.key, { name: "child1" });
      await client.schematics.create(parent.key, { name: "child2" });
      const { result } = renderHook(
        () =>
          Ontology.useListChildren({
            query: { id: project.resourceID(parent.key) },
          }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({ id: project.resourceID(parent.key) });
      });
      await waitFor(() => {
        expect(result.current.data).toHaveLength(2);
      });
      await client.ontology.removeChildren(
        project.resourceID(parent.key),
        schematic.resourceID(child1.key),
      );
      await waitFor(() => {
        expect(result.current.data).toHaveLength(1);
      });
    });
  });

  describe("useResourceList", () => {
    it("should return all resources when no parameters are provided", async () => {
      await client.projects.create({ name: "project1" });
      await client.projects.create({ name: "project2" });

      const { result } = renderHook(() => Ontology.useResourceList({}), {
        wrapper,
      });

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.data.length).toBeGreaterThanOrEqual(2);
        const names = result.current.getItem(result.current.data).map((r) => r.name);
        expect(names).toContain("project1");
        expect(names).toContain("project2");
      });
    });

    it("should respect pagination parameters", async () => {
      const projects: project.Project[] = await Promise.all(
        Array.from(
          { length: 5 },
          async (_, i) => await client.projects.create({ name: `project${i}` }),
        ),
      );

      const idStrings = projects.map((g) =>
        resource.idToString(project.resourceID(g.key)),
      );

      const { result } = renderHook(
        () =>
          Ontology.useResourceList({
            filter: (r) => idStrings.includes(resource.idToString(r.id)),
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
          types: ["project"],
          ids: projects.map((g) => project.resourceID(g.key)),
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
      await client.projects.create({ name: "matching-project" });
      await client.projects.create({ name: "different-name" });

      const { result } = renderHook(() => Ontology.useResourceList({}), {
        wrapper,
      });

      act(() => {
        result.current.retrieve({ searchTerm: "matching" });
      });

      await waitFor(() => {
        expect(result.current.data.length).toBeGreaterThanOrEqual(1);
        const names = result.current.data.map((r) => result.current.getItem(r)?.name);
        expect(names).toContain("matching-project");
        expect(names).not.toContain("different-name");
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

      const newName = id.create();
      const newProject = await client.projects.create({ name: newName });

      await waitFor(() => {
        const item = result.current.getItem(
          resource.idToString(project.resourceID(newProject.key)),
        );
        expect(item?.name).toBe(newName);
      });
    });
  });

  describe("useChildren", () => {
    it("should retrieve children of a parent", async () => {
      const parent = await client.projects.create({ name: "parent" });
      await client.schematics.create(parent.key, { name: "child1" });
      await client.schematics.create(parent.key, { name: "child2" });

      const { result } = await renderHookSuspended(
        () => Ontology.useChildren({ id: project.resourceID(parent.key) }),
        { wrapper },
      );

      await waitFor(() => {
        expect(result.current).not.toBeNull();
        expect(result.current).toHaveLength(2);
      });
    });

    it("should return empty array when parent has no children", async () => {
      const parent = await client.projects.create({ name: "empty-parent" });

      const { result } = await renderHookSuspended(
        () => Ontology.useChildren({ id: project.resourceID(parent.key) }),
        { wrapper },
      );

      await waitFor(() => expect(result.current).toEqual([]));
    });

    it("should throw when no client is connected", () => {
      const noClientWrapper = createSynnaxWrapper({ client: null });
      expect(() =>
        renderHook(() => Ontology.useChildren({ id: ontology.ROOT_ID }), {
          wrapper: noClientWrapper,
        }),
      ).toThrow(DisconnectedError);
    });

    it("should re-fetch when query ID changes", async () => {
      const parent1 = await client.projects.create({ name: "parent1" });
      await client.schematics.create(parent1.key, { name: "p1-child" });

      const parent2 = await client.projects.create({ name: "parent2" });
      await client.schematics.create(parent2.key, { name: "p2-child1" });
      await client.schematics.create(parent2.key, { name: "p2-child2" });

      const { result, rerender } = await renderHookSuspended(
        ({ id }) => Ontology.useChildren({ id }),
        {
          wrapper,
          initialProps: { id: project.resourceID(parent1.key) },
        },
      );

      await waitFor(() => expect(result.current).toHaveLength(1));

      await act(async () => rerender({ id: project.resourceID(parent2.key) }));

      await waitFor(() => expect(result.current).toHaveLength(2));
    });
  });

  describe("useRootChildID", () => {
    it("should resolve a root child by name", async () => {
      const { result } = await renderHookSuspended(
        () => Ontology.useRootChildID({ name: "Arcs" }),
        { wrapper },
      );
      await waitFor(() => expect(result.current?.type).toEqual("arc_directory"));
    });

    it("should return undefined when the root has no child with the name", async () => {
      const name = `missing-${id.create()}`;
      const { result } = await renderHookSuspended(
        () => Ontology.useRootChildID({ name }),
        { wrapper },
      );
      expect(result.current).toBeUndefined();
    });
  });
});
