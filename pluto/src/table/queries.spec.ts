// Copyright 2025 Synnax Labs, Inc.
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

import { Table } from "@/table";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

describe("table queries", () => {
  let wrapper: React.FC<PropsWithChildren>;
  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  describe("useRetrieve", () => {
    it("should retrieve a table by key", async () => {
      const workspace = await client.workspaces.create({
        name: "test_workspace",
        layout: {},
      });
      const table = await client.workspaces.tables.create(workspace.key, {
        name: "retrieve_test",
        data: {},
      });

      const { result } = renderHook(() => Table.useRetrieve({ key: table.key }), {
        wrapper,
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data?.key).toEqual(table.key);
      expect(result.current.data?.name).toEqual("retrieve_test");
    });

    it("should cache retrieved tables", async () => {
      const workspace = await client.workspaces.create({
        name: "cache_workspace",
        layout: {},
      });
      const table = await client.workspaces.tables.create(workspace.key, {
        name: "cached_table",
        data: {},
      });

      const { result: result1 } = renderHook(
        () => Table.useRetrieve({ key: table.key }),
        { wrapper },
      );
      await waitFor(() => expect(result1.current.variant).toEqual("success"));

      const { result: result2 } = renderHook(
        () => Table.useRetrieve({ key: table.key }),
        { wrapper },
      );
      await waitFor(() => expect(result2.current.variant).toEqual("success"));
      expect(result2.current.data).toEqual(result1.current.data);
    });
  });

  describe("useCreate", () => {
    it("should create a new table", async () => {
      const workspace = await client.workspaces.create({
        name: "create_workspace",
        layout: {},
      });

      const { result } = renderHook(() => Table.useCreate(), { wrapper });

      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({
          key,
          workspace: workspace.key,
          name: "created_table",
          data: {},
        });
      });

      expect(result.current.variant).toEqual("success");
      expect(result.current.data?.name).toEqual("created_table");
      expect(result.current.data?.workspace).toEqual(workspace.key);

      const retrieved = await client.workspaces.tables.retrieve({ key });
      expect(retrieved.name).toEqual("created_table");
    });

    it("should store created table in flux store", async () => {
      const workspace = await client.workspaces.create({
        name: "store_workspace",
        layout: {},
      });

      const { result: createResult } = renderHook(() => Table.useCreate(), {
        wrapper,
      });

      const key = uuid.create();
      await act(async () => {
        await createResult.current.updateAsync({
          key,
          workspace: workspace.key,
          name: "stored_table",
          data: {},
        });
      });

      const { result: retrieveResult } = renderHook(() => Table.useRetrieve({ key }), {
        wrapper,
      });
      await waitFor(() => expect(retrieveResult.current.variant).toEqual("success"));
      expect(retrieveResult.current.data?.name).toEqual("stored_table");
    });
  });

  describe("useRename", () => {
    it("should rename a table", async () => {
      const workspace = await client.workspaces.create({
        name: "rename_workspace",
        layout: {},
      });
      const table = await client.workspaces.tables.create(workspace.key, {
        name: "original_name",
        data: {},
      });

      const { result } = renderHook(
        () => {
          const retrieve = Table.useRetrieve({ key: table.key });
          const rename = Table.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );

      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      expect(result.current.retrieve.data?.name).toEqual("original_name");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: table.key,
          name: "renamed_table",
        });
      });

      const retrieved = await client.workspaces.tables.retrieve({ key: table.key });
      expect(retrieved.name).toEqual("renamed_table");
    });

    it("should update cached table after rename", async () => {
      const workspace = await client.workspaces.create({
        name: "rename_cache_workspace",
        layout: {},
      });
      const table = await client.workspaces.tables.create(workspace.key, {
        name: "cache_original",
        data: {},
      });

      const { result } = renderHook(
        () => ({
          retrieve: Table.useRetrieve({ key: table.key }),
          rename: Table.useRename(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      await act(async () => {
        await result.current.rename.updateAsync({
          key: table.key,
          name: "cache_renamed",
        });
      });

      await waitFor(() => {
        expect(result.current.retrieve.data?.name).toEqual("cache_renamed");
      });
    });
  });

  describe("useDelete", () => {
    it("should delete a single table", async () => {
      const workspace = await client.workspaces.create({
        name: "delete_workspace",
        layout: {},
      });
      const table = await client.workspaces.tables.create(workspace.key, {
        name: "delete_single",
        data: {},
      });

      const { result } = renderHook(() => Table.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(table.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(
        client.workspaces.tables.retrieve({ key: table.key }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should delete multiple tables", async () => {
      const workspace = await client.workspaces.create({
        name: "delete_multi_workspace",
        layout: {},
      });
      const table1 = await client.workspaces.tables.create(workspace.key, {
        name: "delete_multi_1",
        data: {},
      });
      const table2 = await client.workspaces.tables.create(workspace.key, {
        name: "delete_multi_2",
        data: {},
      });

      const { result } = renderHook(() => Table.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([table1.key, table2.key]);
      });

      expect(result.current.variant).toEqual("success");

      await expect(
        client.workspaces.tables.retrieve({ key: table1.key }),
      ).rejects.toThrow(NotFoundError);
      await expect(
        client.workspaces.tables.retrieve({ key: table2.key }),
      ).rejects.toThrow(NotFoundError);
    });
  });
});
