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
        newSchematic("retrieve_test"),
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
        newSchematic("cached_schematic"),
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
          ...newSchematic("created_schematic"),
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
          ...newSchematic("stored_schematic"),
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
        newSchematic("original_name"),
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
        newSchematic("cache_original"),
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

  describe("useDelete", () => {
    it("should delete a single schematic", async () => {
      const workspace = await client.workspaces.create({
        name: "delete_workspace",
        layout: {},
      });
      const schematic = await client.schematics.create(
        workspace.key,
        newSchematic("delete_single"),
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
        newSchematic("delete_multi_1"),
      );
      const schematic2 = await client.schematics.create(
        workspace.key,
        newSchematic("delete_multi_2"),
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

  describe("off-page reference sibling schematics", () => {
    it("should return sibling schematics in the same workspace", async () => {
      const ws = await client.workspaces.create({
        name: "opr_sibling_ws",
        layout: {},
      });
      const s1 = await client.schematics.create(ws.key, newSchematic("Current"));
      const s2 = await client.schematics.create(ws.key, newSchematic("Sibling"));

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
  });
});
