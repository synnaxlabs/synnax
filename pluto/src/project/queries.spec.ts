// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createTestClient,
  group,
  NotFoundError,
  project,
  schematic,
} from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Project } from "@/project";
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
        layout: { type: "dashboard", panels: [] },
      });
      const p2 = await client.projects.create({
        name: "project2",
        layout: { type: "schematic", nodes: [] },
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
        layout: {
          type: "dashboard",
          settings: { theme: "dark" },
          panels: [{ id: "panel1", type: "chart" }],
        },
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
      expect((retrievedProject?.layout as any).type).toEqual("dashboard");
      expect((retrievedProject?.layout as any).settings.theme).toEqual("dark");
    });

    it("should handle pagination with limit and offset", async () => {
      for (let i = 0; i < 5; i++)
        await client.projects.create({
          name: `paginationProject${i}`,
          layout: { type: "dashboard", index: i },
        });

      const { result } = renderHook(() => Project.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ limit: 2, offset: 1 });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should return all projects when no pagination params provided", async () => {
      const p1 = await client.projects.create({
        name: "allProjects1",
        layout: { type: "dashboard" },
      });
      const p2 = await client.projects.create({
        name: "allProjects2",
        layout: { type: "schematic" },
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
        layout: { type: "dashboard", created: Date.now() },
      });

      await waitFor(() => {
        expect(result.current.data.length).toBeGreaterThan(initialLength);
        expect(result.current.data).toContain(newProject.key);
      });
    });

    it("should update the list when a project is renamed", async () => {
      const testProject = await client.projects.create({
        name: "originalName",
        layout: { type: "dashboard" },
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

    it("should update the list when a project layout is changed", async () => {
      const testProject = await client.projects.create({
        name: "layoutProject",
        layout: { type: "dashboard", version: 1 },
      });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect((result.current.getItem(testProject.key)?.layout as any).version).toEqual(
        1,
      );

      const newLayout = { type: "schematic", version: 2, nodes: [] };
      await client.projects.setLayout(testProject.key, newLayout);

      await waitFor(() => {
        const updatedProject = result.current.getItem(testProject.key);
        expect((updatedProject?.layout as any).type).toEqual("schematic");
        expect((updatedProject?.layout as any).version).toEqual(2);
      });
    });

    it("should remove project from list when deleted", async () => {
      const testProject = await client.projects.create({
        name: "toDeleteProject",
        layout: { type: "dashboard" },
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
        layout: { type: "dashboard" },
      });
      const p2 = await client.projects.create({
        name: "multiUpdate2",
        layout: { type: "dashboard" },
      });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      // Update both projects simultaneously
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
        layout: { counter: 0 },
      });

      const { result } = renderHook(() => Project.useList(), { wrapper });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      // Perform rapid layout updates
      await act(async () => {
        for (let i = 1; i <= 3; i++)
          await client.projects.setLayout(testProject.key, { counter: i });
      });

      await waitFor(() => {
        const project = result.current.getItem(testProject.key);
        expect((project?.layout as any).counter).toEqual(3);
      });
    });
  });

  describe("useRetrieve", () => {
    it("should retrieve a single project by key", async () => {
      const testProject = await client.projects.create({
        name: "singleProject",
        layout: {
          type: "dashboard",
          title: "My Dashboard",
          widgets: [
            { id: "widget1", type: "chart", position: { x: 0, y: 0 } },
            { id: "widget2", type: "table", position: { x: 1, y: 0 } },
          ],
        },
      });

      const { result } = renderHook(
        () => Project.useRetrieve({ key: testProject.key }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data?.key).toEqual(testProject.key);
      expect(result.current.data?.name).toEqual("singleProject");
      expect(result.current.data?.layout.title).toEqual("My Dashboard");
      expect(result.current.data?.layout.widgets).toHaveLength(2);
    });

    it("should handle retrieve with valid project key", async () => {
      const project = await client.projects.create({
        name: "validProject",
        layout: { config: { setting1: "value1" } },
      });

      const { result } = renderHook(() => Project.useRetrieve({ key: project.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toBeDefined();
      expect(result.current.data?.key).toEqual(project.key);
      expect((result.current.data?.layout as any).config.setting1).toEqual("value1");
    });
  });

  describe("useRename", () => {
    it("should correctly rename a project", async () => {
      const proj = await client.projects.create({
        name: `testProject-${id.create()}`,
        layout: { config: { setting1: "value1" } },
      });

      const newName = `newName-${id.create()}`;
      const { result } = renderHook(
        () => ({
          retrieve: Project.useRetrieve({ key: proj.key }),
          rename: Project.useRename(),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.rename.updateAsync({ key: proj.key, name: newName });
      });
      await waitFor(() => expect(result.current.retrieve.data?.name).toEqual(newName));
    });
  });

  describe("useRetrieveGroupID", () => {
    it("should correctly retrieve group ID", async () => {
      const { result } = renderHook(() => Project.useRetrieveGroupID({}), {
        wrapper,
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
        expect(result.current.data?.type).toEqual("group");
        expect(result.current.data?.key).not.toBeFalsy();
      });
    });
  });

  describe("useDelete", () => {
    it("should correctly delete a project", async () => {
      const proj = await client.projects.create({
        name: "testProject",
        layout: { config: { setting1: "value1" } },
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

  describe("useSaveLayout", () => {
    it("should correctly save a project layout", async () => {
      const proj = await client.projects.create({
        name: "testProject",
        layout: { config: { setting1: "value1" } },
      });

      const { result } = renderHook(
        () => ({
          saveLayout: Project.useSaveLayout(),
          retrieve: Project.useRetrieve({ key: proj.key }),
        }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.retrieve.variant).toEqual("success");
        expect(result.current.retrieve.data?.key).toEqual(proj.key);
        expect(result.current.retrieve.data?.layout).toEqual({
          config: { setting1: "value1" },
        });
      });
      await act(async () => {
        await result.current.saveLayout.updateAsync({
          key: proj.key,
          layout: { config: { setting1: "value2" } },
        });
      });

      await waitFor(() => {
        expect(result.current.saveLayout.variant).toEqual("success");
        expect(result.current.retrieve.data?.key).toEqual(proj.key);
        expect(result.current.retrieve.data?.layout).toEqual({
          config: { setting1: "value2" },
        });
      });
    });
  });

  describe("useRetrieveChildren", () => {
    it("should return children filtered by a single type", async () => {
      const proj = await client.projects.create({ name: "single_type_ws", layout: {} });
      const s1 = await client.schematics.create(proj.key, {
        name: "A Schematic",
      });
      const l1 = await client.logs.create(proj.key, { name: "My Log" });
      await client.lineplots.create(proj.key, { name: "My Plot" });

      const { result } = renderHook(
        () =>
          Project.useRetrieveChildren({
            resourceID: schematic.ontologyID(s1.key),
            types: ["log"],
          }),
        { wrapper },
      );
      await waitFor(() => {
        expect((result.current.data ?? []).length).toBeGreaterThanOrEqual(1);
      });
      const keys = (result.current.data ?? []).map((p) => p.key);
      expect(keys).toContain(l1.key);
      expect(keys).toHaveLength(1);
    });

    it("should return children filtered by multiple types", async () => {
      const proj = await client.projects.create({ name: "multi_type_ws", layout: {} });
      const s1 = await client.schematics.create(proj.key, {
        name: "Source Schematic",
      });
      const lp = await client.lineplots.create(proj.key, { name: "A Plot" });
      const t1 = await client.tables.create(proj.key, { name: "A Table" });
      const l1 = await client.logs.create(proj.key, { name: "A Log" });

      const { result } = renderHook(
        () =>
          Project.useRetrieveChildren({
            resourceID: schematic.ontologyID(s1.key),
            types: ["lineplot", "table"],
          }),
        { wrapper },
      );
      await waitFor(() => {
        expect((result.current.data ?? []).length).toBeGreaterThanOrEqual(2);
      });
      const keys = (result.current.data ?? []).map((p) => p.key);
      expect(keys).toContain(lp.key);
      expect(keys).toContain(t1.key);
      expect(keys).not.toContain(l1.key);
      expect(keys).not.toContain(s1.key);
    });

    it("should return all visualization types except the source type", async () => {
      const proj = await client.projects.create({
        name: "all_but_schematic_ws",
        layout: {},
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

      const { result } = renderHook(
        () =>
          Project.useRetrieveChildren({
            resourceID: schematic.ontologyID(s1.key),
            types: ["lineplot", "table", "log"],
          }),
        { wrapper },
      );
      await waitFor(() => {
        expect((result.current.data ?? []).length).toBeGreaterThanOrEqual(3);
      });
      const keys = (result.current.data ?? []).map((p) => p.key);
      expect(keys).toContain(lp.key);
      expect(keys).toContain(t1.key);
      expect(keys).toContain(l1.key);
      expect(keys).not.toContain(s1.key);
      expect(keys).not.toContain(s2.key);
    });

    it("should exclude the source resource from results", async () => {
      const proj = await client.projects.create({ name: "exclude_ws", layout: {} });
      const s1 = await client.schematics.create(proj.key, {
        name: "Self",
      });
      const s2 = await client.schematics.create(proj.key, {
        name: "Other",
      });

      const { result } = renderHook(
        () =>
          Project.useRetrieveChildren({
            resourceID: schematic.ontologyID(s1.key),
            types: ["schematic"],
          }),
        { wrapper },
      );
      await waitFor(() => {
        expect((result.current.data ?? []).length).toBeGreaterThanOrEqual(1);
      });
      const keys = (result.current.data ?? []).map((p) => p.key);
      expect(keys).not.toContain(s1.key);
      expect(keys).toContain(s2.key);
    });

    it("should return empty when resourceID is not provided", async () => {
      const { result } = renderHook(
        () => Project.useRetrieveChildren({ types: ["schematic"] }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.data ?? []).toEqual([]);
      });
    });

    it("should return empty when no client is available", async () => {
      const noClientWrapper = await createAsyncSynnaxWrapper({ client: null });
      const { result } = renderHook(
        () =>
          Project.useRetrieveChildren({
            resourceID: schematic.ontologyID("some-key"),
            types: ["schematic"],
          }),
        { wrapper: noClientWrapper },
      );
      expect(result.current.data).toBeUndefined();
    });

    it("should find children inside groups", async () => {
      const proj = await client.projects.create({ name: "grouped_ws", layout: {} });
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

      const { result } = renderHook(
        () =>
          Project.useRetrieveChildren({
            resourceID: schematic.ontologyID(s1.key),
            types: ["schematic"],
          }),
        { wrapper },
      );
      await waitFor(() => {
        expect((result.current.data ?? []).length).toBeGreaterThanOrEqual(1);
      });
      const keys = (result.current.data ?? []).map((p) => p.key);
      expect(keys).toContain(s2.key);
      expect(keys).not.toContain(s1.key);
    });

    it("should find children in deeply nested groups", async () => {
      const proj = await client.projects.create({ name: "deep_nested_ws", layout: {} });
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

      const { result } = renderHook(
        () =>
          Project.useRetrieveChildren({
            resourceID: schematic.ontologyID(s1.key),
            types: ["schematic"],
          }),
        { wrapper },
      );
      await waitFor(() => {
        expect((result.current.data ?? []).length).toBeGreaterThanOrEqual(1);
      });
      const keys = (result.current.data ?? []).map((p) => p.key);
      expect(keys).toContain(s2.key);
      expect(keys).not.toContain(s1.key);
    });

    it("should scope results to the source resource's project", async () => {
      const p1 = await client.projects.create({ name: "scope_p_1", layout: {} });
      const p2 = await client.projects.create({ name: "scope_p_2", layout: {} });
      const s1 = await client.schematics.create(p1.key, {
        name: "P1 Schematic",
      });
      await client.schematics.create(p2.key, {
        name: "P2 Schematic",
      });
      const lp1 = await client.lineplots.create(p1.key, { name: "P1 Plot" });
      await client.lineplots.create(p2.key, { name: "P2 Plot" });

      const { result } = renderHook(
        () =>
          Project.useRetrieveChildren({
            resourceID: schematic.ontologyID(s1.key),
            types: ["lineplot"],
          }),
        { wrapper },
      );
      await waitFor(() => {
        expect((result.current.data ?? []).length).toBeGreaterThanOrEqual(1);
      });
      const keys = (result.current.data ?? []).map((p) => p.key);
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
        // --- TestSpace ---
        const proj = await client.projects.create({ name: "TestSpace", layout: {} });
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

        // --- Mirrored TestSpace ---
        const mproj = await client.projects.create({
          name: "Mirrored TestSpace",
          layout: {},
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
        const { result } = renderHook(
          () =>
            Project.useRetrieveChildren({
              resourceID: schematic.ontologyID(source.key),
              types: ["schematic"],
            }),
          { wrapper },
        );
        await waitFor(() => {
          expect((result.current.data ?? []).length).toBeGreaterThanOrEqual(
            expectedSiblings.length,
          );
        });
        const keys = (result.current.data ?? []).map((p) => p.key);
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
