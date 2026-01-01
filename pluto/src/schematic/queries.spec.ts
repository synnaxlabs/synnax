// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Schematic } from "@/schematic";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

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
      const schematic = await client.workspaces.schematics.create(workspace.key, {
        name: "retrieve_test",
        data: {},
      });

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
      const schematic = await client.workspaces.schematics.create(workspace.key, {
        name: "cached_schematic",
        data: {},
      });

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
          key,
          workspace: workspace.key,
          name: "created_schematic",
          data: {},
        });
      });

      expect(result.current.variant).toEqual("success");
      expect(result.current.data?.name).toEqual("created_schematic");
      expect(result.current.data?.workspace).toEqual(workspace.key);

      const retrieved = await client.workspaces.schematics.retrieve({ key });
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
          key,
          workspace: workspace.key,
          name: "stored_schematic",
          data: {},
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
      const schematic = await client.workspaces.schematics.create(workspace.key, {
        name: "original_name",
        data: {},
      });

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

      const retrieved = await client.workspaces.schematics.retrieve({
        key: schematic.key,
      });
      expect(retrieved.name).toEqual("renamed_schematic");
    });

    it("should update cached schematic after rename", async () => {
      const workspace = await client.workspaces.create({
        name: "rename_cache_workspace",
        layout: {},
      });
      const schematic = await client.workspaces.schematics.create(workspace.key, {
        name: "cache_original",
        data: {},
      });

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
      const schematic = await client.workspaces.schematics.create(workspace.key, {
        name: "delete_single",
        data: {},
      });

      const { result } = renderHook(() => Schematic.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(schematic.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(
        client.workspaces.schematics.retrieve({ key: schematic.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should delete multiple schematics", async () => {
      const workspace = await client.workspaces.create({
        name: "delete_multi_workspace",
        layout: {},
      });
      const schematic1 = await client.workspaces.schematics.create(workspace.key, {
        name: "delete_multi_1",
        data: {},
      });
      const schematic2 = await client.workspaces.schematics.create(workspace.key, {
        name: "delete_multi_2",
        data: {},
      });

      const { result } = renderHook(() => Schematic.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([schematic1.key, schematic2.key]);
      });

      expect(result.current.variant).toEqual("success");

      await expect(
        client.workspaces.schematics.retrieve({ key: schematic1.key }),
      ).rejects.toThrow(NotFoundError);
      await expect(
        client.workspaces.schematics.retrieve({ key: schematic2.key }),
      ).rejects.toThrow(NotFoundError);
    });
  });
});
