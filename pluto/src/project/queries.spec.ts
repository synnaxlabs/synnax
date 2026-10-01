// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, NotFoundError, project, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Project } from "@/project";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should return a list of project keys", async () => {
      const p1 = await client.projects.create({
        name: "project1",
      });
      const p2 = await client.projects.create({
        name: "project2",
      });

      const { result } = renderHook(() => Project.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(p1.key);
      expect(result.current.data).toContain(p2.key);
    });

    it("should get individual projects using getItem", async () => {
      const testProject = await client.projects.create({
        name: "testProject",
      });

      const { result } = renderHook(() => Project.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedProject = result.current.getItem(testProject.key);
      expect(retrievedProject?.key).toEqual(testProject.key);
      expect(retrievedProject?.name).toEqual("testProject");
    });

    it("should handle pagination with limit and offset", async () => {
      // Scope the page to this test's own projects: the list is live, so
      // unscoped pagination would absorb projects created by concurrently
      // running spec files.
      const keys: string[] = [];
      for (let i = 0; i < 5; i++) {
        const created = await client.projects.create({
          name: `paginationProject${i}`,
        });
        keys.push(created.key);
      }

      const { result } = renderHook(
        () =>
          Project.useList({
            initialQuery: { keys },
            filter: (p) => keys.includes(p.key),
          }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({ limit: 2, offset: 1, keys });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should return all projects when no pagination params provided", async () => {
      const p1 = await client.projects.create({
        name: "allProjects1",
      });
      const p2 = await client.projects.create({
        name: "allProjects2",
      });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(p1.key);
      expect(result.current.data).toContain(p2.key);
    });

    it("should update the list when a project is created", async () => {
      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialLength = result.current.data.length;

      const newProject = await client.projects.create({
        name: "newProject",
      });

      await waitFor(() => {
        expect(result.current.data.length).toBeGreaterThan(initialLength);
        expect(result.current.data).toContain(newProject.key);
      });
    });

    it("should update the list when a project is renamed", async () => {
      const testProject = await client.projects.create({
        name: "originalName",
      });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.getItem(testProject.key)?.name).toEqual("originalName");

      await client.projects.rename(testProject.key, "renamedProject");

      await waitFor(() => {
        expect(result.current.getItem(testProject.key)?.name).toEqual("renamedProject");
      });
    });

    it("should remove project from list when deleted", async () => {
      const testProject = await client.projects.create({
        name: "toDeleteProject",
      });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(testProject.key);

      await client.projects.delete(testProject.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testProject.key);
      });
    });

    it("should handle multiple project updates simultaneously", async () => {
      const p1 = await client.projects.create({
        name: "multiUpdate1",
      });
      const p2 = await client.projects.create({
        name: "multiUpdate2",
      });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await Promise.all([
        client.projects.rename(p1.key, "updated1"),
        client.projects.rename(p2.key, "updated2"),
      ]);

      await waitFor(() => {
        expect(result.current.getItem(p1.key)?.name).toEqual("updated1");
        expect(result.current.getItem(p2.key)?.name).toEqual("updated2");
      });
    });

    it("should maintain list consistency during rapid changes", async () => {
      const testProject = await client.projects.create({
        name: "rapidChanges",
      });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      await act(async () => {
        for (let i = 1; i <= 3; i++)
          await client.projects.rename(testProject.key, `rapidChanges${i}`);
      });

      await waitFor(() => {
        const project = result.current.getItem(testProject.key);
        expect(project?.name).toEqual("rapidChanges3");
      });
    });
  });

  describe("use", () => {
    it("should retrieve a single project by key", async () => {
      const testProject = await client.projects.create({
        name: "singleProject",
      });

      const { result } = await renderHookSuspended(
        () => Project.use({ key: testProject.key }),
        { wrapper },
      );
      await waitFor(() => expect(result.current).not.toBeNull());

      expect(result.current?.key).toEqual(testProject.key);
      expect(result.current?.name).toEqual("singleProject");
    });

    it("should handle retrieve with valid project key", async () => {
      const project = await client.projects.create({
        name: "validProject",
      });

      const { result } = await renderHookSuspended(
        () => Project.use({ key: project.key }),
        {
          wrapper,
        },
      );
      await waitFor(() => expect(result.current).not.toBeNull());

      expect(result.current).not.toBeNull();
      expect(result.current?.key).toEqual(project.key);
    });
  });

  describe("useRename", () => {
    it("should correctly rename a project", async () => {
      const proj = await client.projects.create({
        name: `testProject-${id.create()}`,
      });

      const newName = `newName-${id.create()}`;
      const { result } = await renderHookSuspended(
        () => ({
          retrieve: Project.use({ key: proj.key }),
          rename: Project.useRename(),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.rename.updateAsync({ key: proj.key, name: newName });
      });
      await waitFor(() => expect(result.current.retrieve?.name).toEqual(newName));
    });

    it("should apply the rename optimistically", async () => {
      const proj = await client.projects.create({
        name: `testProject-${id.create()}`,
      });
      const afterOptimistic = vi.fn();
      const { result } = renderHook(() => Project.useRename({ afterOptimistic }), {
        wrapper,
      });
      await act(async () => {
        await result.current.updateAsync({
          key: proj.key,
          name: `newName-${id.create()}`,
        });
      });
      expect(afterOptimistic).toHaveBeenCalledOnce();
    });
  });

  describe("useGroupID", () => {
    it("should correctly retrieve group ID", async () => {
      const { result } = await renderHookSuspended(() => Project.useGroupID({}), {
        wrapper,
      });
      await waitFor(() => {
        expect(result.current?.type).toEqual("group");
        expect(result.current?.key).not.toBeFalsy();
      });
    });
  });

  describe("useDelete", () => {
    it("should correctly delete a project", async () => {
      const proj = await client.projects.create({
        name: "testProject",
      });

      const { result } = renderHook(() => Project.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(proj.key);
      });
      await waitFor(async () => {
        await expect(client.projects.retrieve(proj.key)).rejects.toThrow(NotFoundError);
      });
    });
  });

  describe("useChildren", () => {
    it("should return children filtered by a single type", async () => {
      const proj = await client.projects.create({ name: "single_type_ws" });
      const s1 = await client.schematics.create(proj.key, {
        name: "A Schematic",
      });
      const l1 = await client.logs.create(proj.key, { name: "My Log" });
      await client.lineplots.create(proj.key, { name: "My Plot" });

      const children = await Project.retrieveChildren(client, {
        resourceID: schematic.ontologyID(s1.key),
        types: ["log"],
      });
      const keys = children.map((p) => p.key);
      expect(keys).toContain(l1.key);
      expect(keys).toHaveLength(1);
    });

    it("should return children filtered by multiple types", async () => {
      const proj = await client.projects.create({ name: "multi_type_ws" });
      const s1 = await client.schematics.create(proj.key, {
        name: "Source Schematic",
      });
      const lp = await client.lineplots.create(proj.key, { name: "A Plot" });
      const t1 = await client.tables.create(proj.key, { name: "A Table" });
      const l1 = await client.logs.create(proj.key, { name: "A Log" });

      const children = await Project.retrieveChildren(client, {
        resourceID: schematic.ontologyID(s1.key),
        types: ["lineplot", "table"],
      });
      const keys = children.map((p) => p.key);
      expect(keys).toContain(lp.key);
      expect(keys).toContain(t1.key);
      expect(keys).not.toContain(l1.key);
      expect(keys).not.toContain(s1.key);
    });

    it("should return all visualization types except the source type", async () => {
      const proj = await client.projects.create({
        name: "all_but_schematic_ws",
      });
      const s1 = await client.schematics.create(proj.key, {
        name: "Current Schematic",
      });
      const s2 = await client.schematics.create(proj.key, {
        name: "Other Schematic",
      });
      const lp = await client.lineplots.create(proj.key, { name: "Plot" });
      const t1 = await client.tables.create(proj.key, { name: "Table" });
      const l1 = await client.logs.create(proj.key, { name: "Log" });

      const children = await Project.retrieveChildren(client, {
        resourceID: schematic.ontologyID(s1.key),
        types: ["lineplot", "table", "log"],
      });
      const keys = children.map((p) => p.key);
      expect(keys).toContain(lp.key);
      expect(keys).toContain(t1.key);
      expect(keys).toContain(l1.key);
      expect(keys).not.toContain(s1.key);
      expect(keys).not.toContain(s2.key);
    });

    it("should exclude the source resource from results", async () => {
      const proj = await client.projects.create({ name: "exclude_ws" });
      const s1 = await client.schematics.create(proj.key, {
        name: "Self",
      });
      const s2 = await client.schematics.create(proj.key, {
        name: "Other",
      });

      const children = await Project.retrieveChildren(client, {
        resourceID: schematic.ontologyID(s1.key),
        types: ["schematic"],
      });
      const keys = children.map((p) => p.key);
      expect(keys).not.toContain(s1.key);
      expect(keys).toContain(s2.key);
    });

    it("should return empty when resourceID is not provided", async () =>
      expect(await Project.retrieveChildren(client, { types: ["schematic"] })).toEqual(
        [],
      ));

    it("should find children inside groups", async () => {
      const proj = await client.projects.create({ name: "grouped_ws" });
      const s1 = await client.schematics.create(proj.key, {
        name: "Top Level",
      });
      const s2 = await client.schematics.create(proj.key, {
        name: "In Group",
      });
      const g = await client.groups.create({
        parent: project.ontologyID(proj.key),
        name: "My Group",
      });
      await client.ontology.moveChildren(
        project.ontologyID(proj.key),
        group.ontologyID(g.key),
        schematic.ontologyID(s2.key),
      );

      const children = await Project.retrieveChildren(client, {
        resourceID: schematic.ontologyID(s1.key),
        types: ["schematic"],
      });
      const keys = children.map((p) => p.key);
      expect(keys).toContain(s2.key);
      expect(keys).not.toContain(s1.key);
    });

    it("should find children in deeply nested groups", async () => {
      const proj = await client.projects.create({ name: "deep_nested_ws" });
      const s1 = await client.schematics.create(proj.key, {
        name: "Top Level",
      });
      const s2 = await client.schematics.create(proj.key, {
        name: "Deeply Nested",
      });
      const outerGroup = await client.groups.create({
        parent: project.ontologyID(proj.key),
        name: "Outer Group",
      });
      const innerGroup = await client.groups.create({
        parent: group.ontologyID(outerGroup.key),
        name: "Inner Group",
      });
      await client.ontology.moveChildren(
        project.ontologyID(proj.key),
        group.ontologyID(innerGroup.key),
        schematic.ontologyID(s2.key),
      );

      const children = await Project.retrieveChildren(client, {
        resourceID: schematic.ontologyID(s1.key),
        types: ["schematic"],
      });
      const keys = children.map((p) => p.key);
      expect(keys).toContain(s2.key);
      expect(keys).not.toContain(s1.key);
    });

    it("should scope results to the source resource's project", async () => {
      const p1 = await client.projects.create({ name: "scope_p_1" });
      const p2 = await client.projects.create({ name: "scope_p_2" });
      const s1 = await client.schematics.create(p1.key, {
        name: "P1 Schematic",
      });
      await client.schematics.create(p2.key, {
        name: "P2 Schematic",
      });
      const lp1 = await client.lineplots.create(p1.key, { name: "P1 Plot" });
      await client.lineplots.create(p2.key, { name: "P2 Plot" });

      const children = await Project.retrieveChildren(client, {
        resourceID: schematic.ontologyID(s1.key),
        types: ["lineplot"],
      });
      const keys = children.map((p) => p.key);
      expect(keys).toContain(lp1.key);
      expect(keys).toHaveLength(1);
    });

    describe("nested group visibility", () => {
      // TestSpace structure:
      //   Schematic A (top level)
      //   Group 1
      //     Schematic B
      //     Group 2
      //       Schematic C
      //       Schematic D
      //       Group E
      //         Schematic E
      //
      // Mirrored TestSpace (separate project, same structure):
      //   Schematic A Mirrored (top level)
      //   Group 1 Mirrored
      //     Schematic B Mirrored
      //     Group 2 Mirrored
      //       Schematic C Mirrored
      //       Schematic D Mirrored
      //       Group E Mirrored
      //         Schematic E Mirrored

      let sA: schematic.Schematic,
        sB: schematic.Schematic,
        sC: schematic.Schematic,
        sD: schematic.Schematic,
        sE: schematic.Schematic;
      let sAm: schematic.Schematic,
        sBm: schematic.Schematic,
        sCm: schematic.Schematic,
        sDm: schematic.Schematic,
        sEm: schematic.Schematic;

      beforeEach(async () => {
        const proj = await client.projects.create({ name: "TestSpace" });
        sA = await client.schematics.create(proj.key, {
          name: "Schematic A",
        });
        sB = await client.schematics.create(proj.key, {
          name: "Schematic B",
        });
        sC = await client.schematics.create(proj.key, {
          name: "Schematic C",
        });
        sD = await client.schematics.create(proj.key, {
          name: "Schematic D",
        });
        sE = await client.schematics.create(proj.key, {
          name: "Schematic E",
        });

        const g1 = await client.groups.create({
          parent: project.ontologyID(proj.key),
          name: "Group 1",
        });
        await client.ontology.moveChildren(
          project.ontologyID(proj.key),
          group.ontologyID(g1.key),
          schematic.ontologyID(sB.key),
        );

        const g2 = await client.groups.create({
          parent: group.ontologyID(g1.key),
          name: "Group 2",
        });
        await client.ontology.moveChildren(
          project.ontologyID(proj.key),
          group.ontologyID(g2.key),
          schematic.ontologyID(sC.key),
        );
        await client.ontology.moveChildren(
          project.ontologyID(proj.key),
          group.ontologyID(g2.key),
          schematic.ontologyID(sD.key),
        );

        const gE = await client.groups.create({
          parent: group.ontologyID(g2.key),
          name: "Group E",
        });
        await client.ontology.moveChildren(
          project.ontologyID(proj.key),
          group.ontologyID(gE.key),
          schematic.ontologyID(sE.key),
        );

        const mproj = await client.projects.create({
          name: "Mirrored TestSpace",
        });
        sAm = await client.schematics.create(mproj.key, {
          name: "Schematic A Mirrored",
        });
        sBm = await client.schematics.create(mproj.key, {
          name: "Schematic B Mirrored",
        });
        sCm = await client.schematics.create(mproj.key, {
          name: "Schematic C Mirrored",
        });
        sDm = await client.schematics.create(mproj.key, {
          name: "Schematic D Mirrored",
        });
        sEm = await client.schematics.create(mproj.key, {
          name: "Schematic E Mirrored",
        });

        const mg1 = await client.groups.create({
          parent: project.ontologyID(mproj.key),
          name: "Group 1 Mirrored",
        });
        await client.ontology.moveChildren(
          project.ontologyID(mproj.key),
          group.ontologyID(mg1.key),
          schematic.ontologyID(sBm.key),
        );

        const mg2 = await client.groups.create({
          parent: group.ontologyID(mg1.key),
          name: "Group 2 Mirrored",
        });
        await client.ontology.moveChildren(
          project.ontologyID(mproj.key),
          group.ontologyID(mg2.key),
          schematic.ontologyID(sCm.key),
        );
        await client.ontology.moveChildren(
          project.ontologyID(mproj.key),
          group.ontologyID(mg2.key),
          schematic.ontologyID(sDm.key),
        );

        const mgE = await client.groups.create({
          parent: group.ontologyID(mg2.key),
          name: "Group E Mirrored",
        });
        await client.ontology.moveChildren(
          project.ontologyID(mproj.key),
          group.ontologyID(mgE.key),
          schematic.ontologyID(sEm.key),
        );
      });

      const expectSiblingsFromSource = async (
        source: schematic.Schematic,
        expectedSiblings: schematic.Schematic[],
        unexpectedKeys: string[],
      ): Promise<void> => {
        const children = await Project.retrieveChildren(client, {
          resourceID: schematic.ontologyID(source.key),
          types: ["schematic"],
        });
        const keys = children.map((p) => p.key);
        for (const s of expectedSiblings) expect(keys).toContain(s.key);
        expect(keys).not.toContain(source.key);
        for (const k of unexpectedKeys) expect(keys).not.toContain(k);
      };

      it("top-level schematic A sees all project schematics", async () => {
        await expectSiblingsFromSource(
          sA,
          [sB, sC, sD, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("grouped schematic B sees all project schematics", async () => {
        await expectSiblingsFromSource(
          sB,
          [sA, sC, sD, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("deeply nested schematic C sees all project schematics", async () => {
        await expectSiblingsFromSource(
          sC,
          [sA, sB, sD, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("deeply nested schematic D sees all project schematics", async () => {
        await expectSiblingsFromSource(
          sD,
          [sA, sB, sC, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("most deeply nested schematic E sees all project schematics", async () => {
        await expectSiblingsFromSource(
          sE,
          [sA, sB, sC, sD],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("mirrored schematic A sees only mirrored schematics", async () => {
        await expectSiblingsFromSource(
          sAm,
          [sBm, sCm, sDm, sEm],
          [sA.key, sB.key, sC.key, sD.key, sE.key],
        );
      });

      it("mirrored deeply nested schematic E sees only mirrored schematics", async () => {
        await expectSiblingsFromSource(
          sEm,
          [sAm, sBm, sCm, sDm],
          [sA.key, sB.key, sC.key, sD.key, sE.key],
        );
      });
    });
  });
});
