// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { Workspace } from "@/workspace";

const client = createTestClient();

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
});
