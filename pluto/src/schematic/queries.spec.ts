// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError, schematic } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Schematic } from "@/schematic";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const newSchematic = (overrides: Partial<schematic.New> = {}): schematic.New => ({
  name: "test",
  viewport: { position: { x: 0, y: 0 }, zoom: 1 },
  legend: { visible: true, position: { x: 50, y: 50 }, colors: {} },
  nodes: [],
  edges: [],
  props: {},
  ...overrides,
});

describe("schematic queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useRetrieve", () => {
    it("should retrieve a schematic by key", async () => {
      const workspace = await client.workspaces.create({
        name: "test_workspace",
        layout: {},
      });
      const schematic = await client.schematics.create(
        workspace.key,
        newSchematic({ name: "retrieve_test" }),
      );

      const { result } = renderHook(
        () => Schematic.useRetrieve({ key: schematic.key }),
        {
          wrapper,
        },
      );
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data?.key).toEqual(schematic.key);
      expect(result.current.data?.name).toEqual("retrieve_test");
    });

    it("should cache retrieved schematics", async () => {
      const workspace = await client.workspaces.create({
        name: "cache_workspace",
        layout: {},
      });
      const schematic = await client.schematics.create(
        workspace.key,
        newSchematic({ name: "cached_schematic" }),
      );

      const { result: result1 } = renderHook(
        () => Schematic.useRetrieve({ key: schematic.key }),
        { wrapper },
      );
      await waitFor(() => expect(result1.current.variant).toEqual("success"));

      const { result: result2 } = renderHook(
        () => Schematic.useRetrieve({ key: schematic.key }),
        { wrapper },
      );
      await waitFor(() => expect(result2.current.variant).toEqual("success"));
      expect(result2.current.data).toEqual(result1.current.data);
    });
  });

  describe("useCreate", () => {
    it("should create a new schematic", async () => {
      const workspace = await client.workspaces.create({
        name: "create_workspace",
        layout: {},
      });

      const { result } = renderHook(() => Schematic.useCreate(), { wrapper });

      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({
          ...newSchematic({ name: "created_schematic" }),
          key,
          workspace: workspace.key,
        });
      });

      expect(result.current.variant).toEqual("success");
      expect(result.current.data?.name).toEqual("created_schematic");
      expect(result.current.data?.workspace).toEqual(workspace.key);

      const retrieved = await client.schematics.retrieve({ key });
      expect(retrieved.name).toEqual("created_schematic");
    });

    it("should store created schematic in flux store", async () => {
      const workspace = await client.workspaces.create({
        name: "store_workspace",
        layout: {},
      });

      const { result: createResult } = renderHook(() => Schematic.useCreate(), {
        wrapper,
      });

      const key = uuid.create();
      await act(async () => {
        await createResult.current.updateAsync({
          ...newSchematic({ name: "stored_schematic" }),
          key,
          workspace: workspace.key,
        });
      });

      const { result: retrieveResult } = renderHook(
        () => Schematic.useRetrieve({ key }),
        { wrapper },
      );
      await waitFor(() => expect(retrieveResult.current.variant).toEqual("success"));
      expect(retrieveResult.current.data?.name).toEqual("stored_schematic");
    });
  });

  describe("useRename", () => {
    it("should rename a schematic", async () => {
      const workspace = await client.workspaces.create({
        name: "rename_workspace",
        layout: {},
      });
      const schematic = await client.schematics.create(
        workspace.key,
        newSchematic({ name: "original_name" }),
      );

      const { result } = renderHook(
        () => {
          const retrieve = Schematic.useRetrieve({ key: schematic.key });
          const rename = Schematic.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );

      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      expect(result.current.retrieve.data?.name).toEqual("original_name");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: schematic.key,
          name: "renamed_schematic",
        });
      });

      const retrieved = await client.schematics.retrieve({
        key: schematic.key,
      });
      expect(retrieved.name).toEqual("renamed_schematic");
    });

    it("should update cached schematic after rename", async () => {
      const workspace = await client.workspaces.create({
        name: "rename_cache_workspace",
        layout: {},
      });
      const schematic = await client.schematics.create(
        workspace.key,
        newSchematic({ name: "cache_original" }),
      );

      const { result } = renderHook(
        () => ({
          retrieve: Schematic.useRetrieve({ key: schematic.key }),
          rename: Schematic.useRename(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      await act(async () => {
        await result.current.rename.updateAsync({
          key: schematic.key,
          name: "cache_renamed",
        });
      });

      await waitFor(() => {
        expect(result.current.retrieve.data?.name).toEqual("cache_renamed");
      });
    });
  });

  describe("useDispatch", () => {
    it("should dispatch setNodePosition and update the flux store", async () => {
      const workspace = await client.workspaces.create({
        name: "dispatch_pos_workspace",
        layout: {},
      });
      const s = await client.schematics.create(
        workspace.key,
        newSchematic({
          name: "dispatch_pos",
          nodes: [{ key: "n1", position: { x: 0, y: 0 }, selected: false, zIndex: 0, type: "" }],
        }),
      );

      const { result } = renderHook(
        () => ({
          retrieve: Schematic.useRetrieve({ key: s.key }),
          dispatch: Schematic.useDispatch(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      await act(async () => {
        await result.current.dispatch.updateAsync({
          key: s.key,
          actions: schematic.setNodePosition({ key: "n1", position: { x: 100, y: 200 } }),
        });
      });

      await waitFor(() => {
        const node = result.current.retrieve.data?.nodes.find((n) => n.key === "n1");
        expect(node?.position).toEqual({ x: 100, y: 200 });
      });

      const retrieved = await client.schematics.retrieve({ key: s.key });
      expect(retrieved.nodes[0].position).toEqual({ x: 100, y: 200 });
    });

    it("should dispatch multiple actions in sequence", async () => {
      const workspace = await client.workspaces.create({
        name: "dispatch_multi_workspace",
        layout: {},
      });
      const s = await client.schematics.create(
        workspace.key,
        newSchematic({ name: "dispatch_multi" }),
      );

      const { result } = renderHook(
        () => ({
          retrieve: Schematic.useRetrieve({ key: s.key }),
          dispatch: Schematic.useDispatch(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      await act(async () => {
        await result.current.dispatch.updateAsync({
          key: s.key,
          actions: [
            schematic.addNode({
              node: { key: "a", position: { x: 10, y: 20 }, selected: false, zIndex: 0, type: "" },
            }),
            schematic.addNode({
              node: { key: "b", position: { x: 30, y: 40 }, selected: false, zIndex: 0, type: "" },
            }),
            schematic.removeNode({ key: "a" }),
          ],
        });
      });

      await waitFor(() => {
        expect(result.current.retrieve.data?.nodes).toHaveLength(1);
        expect(result.current.retrieve.data?.nodes[0].key).toEqual("b");
      });
    });

    it("should dispatch edge actions", async () => {
      const workspace = await client.workspaces.create({
        name: "dispatch_edge_workspace",
        layout: {},
      });
      const s = await client.schematics.create(
        workspace.key,
        newSchematic({ name: "dispatch_edges" }),
      );

      const { result } = renderHook(
        () => ({
          retrieve: Schematic.useRetrieve({ key: s.key }),
          dispatch: Schematic.useDispatch(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      await act(async () => {
        await result.current.dispatch.updateAsync({
          key: s.key,
          actions: [
            schematic.setEdge({
              edge: { key: "e1", source: "n1", target: "n2", id: "e1", selected: false },
            }),
            schematic.setEdge({
              edge: { key: "e2", source: "n2", target: "n3", id: "e2", selected: false },
            }),
            schematic.removeEdge({ key: "e1" }),
          ],
        });
      });

      await waitFor(() => {
        expect(result.current.retrieve.data?.edges).toHaveLength(1);
        expect(result.current.retrieve.data?.edges[0].key).toEqual("e2");
      });
    });

    it("should persist dispatched actions to the server", async () => {
      const workspace = await client.workspaces.create({
        name: "dispatch_persist_workspace",
        layout: {},
      });
      const s = await client.schematics.create(
        workspace.key,
        newSchematic({ name: "dispatch_persist" }),
      );

      const { result } = renderHook(
        () => ({
          retrieve: Schematic.useRetrieve({ key: s.key }),
          dispatch: Schematic.useDispatch(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      await act(async () => {
        await result.current.dispatch.updateAsync({
          key: s.key,
          actions: schematic.addNode({
            node: { key: "persisted", position: { x: 5, y: 10 }, selected: false, zIndex: 0, type: "" },
          }),
        });
      });

      const retrieved = await client.schematics.retrieve({ key: s.key });
      expect(retrieved.nodes).toHaveLength(1);
      expect(retrieved.nodes[0].key).toEqual("persisted");
    });
  });

  describe("useDelete", () => {
    it("should delete a single schematic", async () => {
      const workspace = await client.workspaces.create({
        name: "delete_workspace",
        layout: {},
      });
      const schematic = await client.schematics.create(
        workspace.key,
        newSchematic({ name: "delete_single" }),
      );

      const { result } = renderHook(() => Schematic.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(schematic.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(client.schematics.retrieve({ key: schematic.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should delete multiple schematics", async () => {
      const workspace = await client.workspaces.create({
        name: "delete_multi_workspace",
        layout: {},
      });
      const schematic1 = await client.schematics.create(
        workspace.key,
        newSchematic({ name: "delete_multi_1" }),
      );
      const schematic2 = await client.schematics.create(
        workspace.key,
        newSchematic({ name: "delete_multi_2" }),
      );

      const { result } = renderHook(() => Schematic.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([schematic1.key, schematic2.key]);
      });

      expect(result.current.variant).toEqual("success");

      await expect(client.schematics.retrieve({ key: schematic1.key })).rejects.toThrow(
        NotFoundError,
      );
      await expect(client.schematics.retrieve({ key: schematic2.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });
});
