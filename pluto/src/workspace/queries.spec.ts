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
  schematic,
  workspace,
} from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { Workspace } from "@/workspace";

const client = createTestClient();

const newSchematic = (name: string): schematic.New => ({
  name,
  legend: {
    visible: true,
    position: {
      x: 50,
      y: 50,
      root: { x: "left", y: "top" },
      units: { x: "px", y: "px" },
    },
    colors: {},
  },
  nodes: [],
  edges: [],
  props: {},
});

describe("queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useList", () => {
    it("should return a list of workspace keys", async () => {
      const ws1 = await client.workspaces.create({
        name: "workspace1",
        layout: { type: "dashboard", panels: [] },
      });
      const ws2 = await client.workspaces.create({
        name: "workspace2",
        layout: { type: "schematic", nodes: [] },
      });

      const { result } = renderHook(() => Workspace.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(2);
      expect(result.current.data).toContain(ws1.key);
      expect(result.current.data).toContain(ws2.key);
    });

    it("should get individual workspaces using getItem", async () => {
      const testWorkspace = await client.workspaces.create({
        name: "testWorkspace",
        layout: {
          type: "dashboard",
          settings: { theme: "dark" },
          panels: [{ id: "panel1", type: "chart" }],
        },
      });

      const { result } = renderHook(() => Workspace.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedWorkspace = result.current.getItem(testWorkspace.key);
      expect(retrievedWorkspace?.key).toEqual(testWorkspace.key);
      expect(retrievedWorkspace?.name).toEqual("testWorkspace");
      expect((retrievedWorkspace?.layout as any).type).toEqual("dashboard");
      expect((retrievedWorkspace?.layout as any).settings.theme).toEqual("dark");
    });

    it("should handle pagination with limit and offset", async () => {
      for (let i = 0; i < 5; i++)
        await client.workspaces.create({
          name: `paginationWorkspace${i}`,
          layout: { type: "dashboard", index: i },
        });

      const { result } = renderHook(() => Workspace.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ limit: 2, offset: 1 });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should return all workspaces when no pagination params provided", async () => {
      const ws1 = await client.workspaces.create({
        name: "allWorkspaces1",
        layout: { type: "dashboard" },
      });
      const ws2 = await client.workspaces.create({
        name: "allWorkspaces2",
        layout: { type: "schematic" },
      });

      const { result } = renderHook(() => Workspace.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(ws1.key);
      expect(result.current.data).toContain(ws2.key);
    });

    it("should update the list when a workspace is created", async () => {
      const { result } = renderHook(() => Workspace.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const initialLength = result.current.data.length;

      const newWorkspace = await client.workspaces.create({
        name: "newWorkspace",
        layout: { type: "dashboard", created: Date.now() },
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(initialLength + 1);
        expect(result.current.data).toContain(newWorkspace.key);
      });
    });

    it("should update the list when a workspace is renamed", async () => {
      const testWorkspace = await client.workspaces.create({
        name: "originalName",
        layout: { type: "dashboard" },
      });

      const { result } = renderHook(() => Workspace.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.getItem(testWorkspace.key)?.name).toEqual("originalName");

      await client.workspaces.rename(testWorkspace.key, "renamedWorkspace");

      await waitFor(() => {
        expect(result.current.getItem(testWorkspace.key)?.name).toEqual(
          "renamedWorkspace",
        );
      });
    });

    it("should update the list when a workspace layout is changed", async () => {
      const testWorkspace = await client.workspaces.create({
        name: "layoutWorkspace",
        layout: { type: "dashboard", version: 1 },
      });

      const { result } = renderHook(() => Workspace.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(
        (result.current.getItem(testWorkspace.key)?.layout as any).version,
      ).toEqual(1);

      const newLayout = { type: "schematic", version: 2, nodes: [] };
      await client.workspaces.setLayout(testWorkspace.key, newLayout);

      await waitFor(() => {
        const updatedWorkspace = result.current.getItem(testWorkspace.key);
        expect((updatedWorkspace?.layout as any).type).toEqual("schematic");
        expect((updatedWorkspace?.layout as any).version).toEqual(2);
      });
    });

    it("should remove workspace from list when deleted", async () => {
      const testWorkspace = await client.workspaces.create({
        name: "toDeleteWorkspace",
        layout: { type: "dashboard" },
      });

      const { result } = renderHook(() => Workspace.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toContain(testWorkspace.key);

      await client.workspaces.delete(testWorkspace.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testWorkspace.key);
      });
    });

    it("should handle multiple workspace updates simultaneously", async () => {
      const ws1 = await client.workspaces.create({
        name: "multiUpdate1",
        layout: { type: "dashboard" },
      });
      const ws2 = await client.workspaces.create({
        name: "multiUpdate2",
        layout: { type: "dashboard" },
      });

      const { result } = renderHook(() => Workspace.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      // Update both workspaces simultaneously
      await Promise.all([
        client.workspaces.rename(ws1.key, "updated1"),
        client.workspaces.rename(ws2.key, "updated2"),
      ]);

      await waitFor(() => {
        expect(result.current.getItem(ws1.key)?.name).toEqual("updated1");
        expect(result.current.getItem(ws2.key)?.name).toEqual("updated2");
      });
    });

    it("should maintain list consistency during rapid changes", async () => {
      const testWorkspace = await client.workspaces.create({
        name: "rapidChanges",
        layout: { counter: 0 },
      });

      const { result } = renderHook(() => Workspace.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      // Perform rapid layout updates
      await act(async () => {
        for (let i = 1; i <= 3; i++)
          await client.workspaces.setLayout(testWorkspace.key, { counter: i });
      });

      await waitFor(() => {
        const workspace = result.current.getItem(testWorkspace.key);
        expect((workspace?.layout as any).counter).toEqual(3);
      });
    });
  });

  describe("useRetrieve", () => {
    it("should retrieve a single workspace by key", async () => {
      const testWorkspace = await client.workspaces.create({
        name: "singleWorkspace",
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
        () => Workspace.useRetrieve({ key: testWorkspace.key }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data?.key).toEqual(testWorkspace.key);
      expect(result.current.data?.name).toEqual("singleWorkspace");
      expect(result.current.data?.layout.title).toEqual("My Dashboard");
      expect(result.current.data?.layout.widgets).toHaveLength(2);
    });

    it("should handle retrieve with valid workspace key", async () => {
      const workspace = await client.workspaces.create({
        name: "validWorkspace",
        layout: { config: { setting1: "value1" } },
      });

      const { result } = renderHook(
        () => Workspace.useRetrieve({ key: workspace.key }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.data).toBeDefined();
      expect(result.current.data?.key).toEqual(workspace.key);
      expect((result.current.data?.layout as any).config.setting1).toEqual("value1");
    });
  });

  describe("useRename", () => {
    it("should correctly rename a workspace", async () => {
      const ws = await client.workspaces.create({
        name: `testWorkspace-${id.create()}`,
        layout: { config: { setting1: "value1" } },
      });

      const newName = `newName-${id.create()}`;
      const { result } = renderHook(
        () => ({
          retrieve: Workspace.useRetrieve({ key: ws.key }),
          rename: Workspace.useRename(),
        }),
        { wrapper },
      );
      await act(async () => {
        await result.current.rename.updateAsync({ key: ws.key, name: newName });
      });
      await waitFor(() => expect(result.current.retrieve.data?.name).toEqual(newName));
    });
  });

  describe("useRetrieveGroupID", () => {
    it("should correctly retrieve group ID", async () => {
      const { result } = renderHook(() => Workspace.useRetrieveGroupID({}), {
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
    it("should correctly delete a workspace", async () => {
      const ws = await client.workspaces.create({
        name: "testWorkspace",
        layout: { config: { setting1: "value1" } },
      });

      const { result } = renderHook(() => Workspace.useDelete(), { wrapper });
      await act(async () => {
        await result.current.updateAsync(ws.key);
      });
      await waitFor(async () => {
        await expect(client.workspaces.retrieve(ws.key)).rejects.toThrow(NotFoundError);
      });
    });
  });

  describe("useSaveLayout", () => {
    it("should correctly save a workspace layout", async () => {
      const ws = await client.workspaces.create({
        name: "testWorkspace",
        layout: { config: { setting1: "value1" } },
      });

      const { result } = renderHook(
        () => ({
          saveLayout: Workspace.useSaveLayout(),
          retrieve: Workspace.useRetrieve({ key: ws.key }),
        }),
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.retrieve.variant).toEqual("success");
        expect(result.current.retrieve.data?.key).toEqual(ws.key);
        expect(result.current.retrieve.data?.layout).toEqual({
          config: { setting1: "value1" },
        });
      });
      await act(async () => {
        await result.current.saveLayout.updateAsync({
          key: ws.key,
          layout: {
            config: { setting1: "value2" },
          },
        });
      });

      await waitFor(() => {
        expect(result.current.saveLayout.variant).toEqual("success");
        expect(result.current.retrieve.data?.key).toEqual(ws.key);
        expect(result.current.retrieve.data?.layout).toEqual({
          config: { setting1: "value2" },
        });
      });
    });
  });

  describe("useRetrieveChildren", () => {
    it("should return children filtered by a single type", async () => {
      const ws = await client.workspaces.create({
        name: "single_type_ws",
        layout: {},
      });
      const s1 = await client.schematics.create(ws.key, newSchematic("A Schematic"));
      const l1 = await client.logs.create(ws.key, {
        name: "My Log",
        data: {},
      });
      await client.lineplots.create(ws.key, {
        name: "My Plot",
        data: {},
      });

      const { result } = renderHook(
        () =>
          Workspace.useRetrieveChildren({
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
      const ws = await client.workspaces.create({
        name: "multi_type_ws",
        layout: {},
      });
      const s1 = await client.schematics.create(
        ws.key,
        newSchematic("Source Schematic"),
      );
      const lp = await client.lineplots.create(ws.key, {
        name: "A Plot",
        data: {},
      });
      const t1 = await client.tables.create(ws.key, {
        name: "A Table",
        data: {},
      });
      const l1 = await client.logs.create(ws.key, {
        name: "A Log",
        data: {},
      });

      const { result } = renderHook(
        () =>
          Workspace.useRetrieveChildren({
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
      const ws = await client.workspaces.create({
        name: "all_but_schematic_ws",
        layout: {},
      });
      const s1 = await client.schematics.create(
        ws.key,
        newSchematic("Current Schematic"),
      );
      const s2 = await client.schematics.create(
        ws.key,
        newSchematic("Other Schematic"),
      );
      const lp = await client.lineplots.create(ws.key, {
        name: "Plot",
        data: {},
      });
      const t1 = await client.tables.create(ws.key, {
        name: "Table",
        data: {},
      });
      const l1 = await client.logs.create(ws.key, {
        name: "Log",
        data: {},
      });

      const { result } = renderHook(
        () =>
          Workspace.useRetrieveChildren({
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
      const ws = await client.workspaces.create({
        name: "exclude_ws",
        layout: {},
      });
      const s1 = await client.schematics.create(ws.key, newSchematic("Self"));
      const s2 = await client.schematics.create(ws.key, newSchematic("Other"));

      const { result } = renderHook(
        () =>
          Workspace.useRetrieveChildren({
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
        () => Workspace.useRetrieveChildren({ types: ["schematic"] }),
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
          Workspace.useRetrieveChildren({
            resourceID: schematic.ontologyID("some-key"),
            types: ["schematic"],
          }),
        { wrapper: noClientWrapper },
      );
      expect(result.current.data).toBeUndefined();
    });

    it("should find children inside groups", async () => {
      const ws = await client.workspaces.create({
        name: "grouped_ws",
        layout: {},
      });
      const s1 = await client.schematics.create(ws.key, newSchematic("Top Level"));
      const s2 = await client.schematics.create(ws.key, newSchematic("In Group"));
      const g = await client.groups.create({
        parent: workspace.ontologyID(ws.key),
        name: "My Group",
      });
      await client.ontology.moveChildren(
        workspace.ontologyID(ws.key),
        group.ontologyID(g.key),
        schematic.ontologyID(s2.key),
      );

      const { result } = renderHook(
        () =>
          Workspace.useRetrieveChildren({
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
      const ws = await client.workspaces.create({
        name: "deep_nested_ws",
        layout: {},
      });
      const s1 = await client.schematics.create(ws.key, newSchematic("Top Level"));
      const s2 = await client.schematics.create(ws.key, newSchematic("Deeply Nested"));
      const outerGroup = await client.groups.create({
        parent: workspace.ontologyID(ws.key),
        name: "Outer Group",
      });
      const innerGroup = await client.groups.create({
        parent: group.ontologyID(outerGroup.key),
        name: "Inner Group",
      });
      await client.ontology.moveChildren(
        workspace.ontologyID(ws.key),
        group.ontologyID(innerGroup.key),
        schematic.ontologyID(s2.key),
      );

      const { result } = renderHook(
        () =>
          Workspace.useRetrieveChildren({
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

    it("should scope results to the source resource's workspace", async () => {
      const ws1 = await client.workspaces.create({
        name: "scope_ws_1",
        layout: {},
      });
      const ws2 = await client.workspaces.create({
        name: "scope_ws_2",
        layout: {},
      });
      const s1 = await client.schematics.create(ws1.key, newSchematic("WS1 Schematic"));
      await client.schematics.create(ws2.key, newSchematic("WS2 Schematic"));
      const lp1 = await client.lineplots.create(ws1.key, {
        name: "WS1 Plot",
        data: {},
      });
      await client.lineplots.create(ws2.key, {
        name: "WS2 Plot",
        data: {},
      });

      const { result } = renderHook(
        () =>
          Workspace.useRetrieveChildren({
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
      // Mirrored TestSpace (separate workspace, same structure):
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
        const ws = await client.workspaces.create({
          name: "TestSpace",
          layout: {},
        });
        sA = await client.schematics.create(ws.key, newSchematic("Schematic A"));
        sB = await client.schematics.create(ws.key, newSchematic("Schematic B"));
        sC = await client.schematics.create(ws.key, newSchematic("Schematic C"));
        sD = await client.schematics.create(ws.key, newSchematic("Schematic D"));
        sE = await client.schematics.create(ws.key, newSchematic("Schematic E"));

        const g1 = await client.groups.create({
          parent: workspace.ontologyID(ws.key),
          name: "Group 1",
        });
        await client.ontology.moveChildren(
          workspace.ontologyID(ws.key),
          group.ontologyID(g1.key),
          schematic.ontologyID(sB.key),
        );

        const g2 = await client.groups.create({
          parent: group.ontologyID(g1.key),
          name: "Group 2",
        });
        await client.ontology.moveChildren(
          workspace.ontologyID(ws.key),
          group.ontologyID(g2.key),
          schematic.ontologyID(sC.key),
        );
        await client.ontology.moveChildren(
          workspace.ontologyID(ws.key),
          group.ontologyID(g2.key),
          schematic.ontologyID(sD.key),
        );

        const gE = await client.groups.create({
          parent: group.ontologyID(g2.key),
          name: "Group E",
        });
        await client.ontology.moveChildren(
          workspace.ontologyID(ws.key),
          group.ontologyID(gE.key),
          schematic.ontologyID(sE.key),
        );

        // --- Mirrored TestSpace ---
        const mws = await client.workspaces.create({
          name: "Mirrored TestSpace",
          layout: {},
        });
        sAm = await client.schematics.create(
          mws.key,
          newSchematic("Schematic A Mirrored"),
        );
        sBm = await client.schematics.create(
          mws.key,
          newSchematic("Schematic B Mirrored"),
        );
        sCm = await client.schematics.create(
          mws.key,
          newSchematic("Schematic C Mirrored"),
        );
        sDm = await client.schematics.create(
          mws.key,
          newSchematic("Schematic D Mirrored"),
        );
        sEm = await client.schematics.create(
          mws.key,
          newSchematic("Schematic E Mirrored"),
        );

        const mg1 = await client.groups.create({
          parent: workspace.ontologyID(mws.key),
          name: "Group 1 Mirrored",
        });
        await client.ontology.moveChildren(
          workspace.ontologyID(mws.key),
          group.ontologyID(mg1.key),
          schematic.ontologyID(sBm.key),
        );

        const mg2 = await client.groups.create({
          parent: group.ontologyID(mg1.key),
          name: "Group 2 Mirrored",
        });
        await client.ontology.moveChildren(
          workspace.ontologyID(mws.key),
          group.ontologyID(mg2.key),
          schematic.ontologyID(sCm.key),
        );
        await client.ontology.moveChildren(
          workspace.ontologyID(mws.key),
          group.ontologyID(mg2.key),
          schematic.ontologyID(sDm.key),
        );

        const mgE = await client.groups.create({
          parent: group.ontologyID(mg2.key),
          name: "Group E Mirrored",
        });
        await client.ontology.moveChildren(
          workspace.ontologyID(mws.key),
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
            Workspace.useRetrieveChildren({
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

      it("top-level schematic A sees all workspace schematics", async () => {
        await expectSiblingsFromSource(
          sA,
          [sB, sC, sD, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("grouped schematic B sees all workspace schematics", async () => {
        await expectSiblingsFromSource(
          sB,
          [sA, sC, sD, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("deeply nested schematic C sees all workspace schematics", async () => {
        await expectSiblingsFromSource(
          sC,
          [sA, sB, sD, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("deeply nested schematic D sees all workspace schematics", async () => {
        await expectSiblingsFromSource(
          sD,
          [sA, sB, sC, sE],
          [sAm.key, sBm.key, sCm.key, sDm.key, sEm.key],
        );
      });

      it("most deeply nested schematic E sees all workspace schematics", async () => {
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
