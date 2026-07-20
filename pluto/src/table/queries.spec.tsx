// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, table } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Errors } from "@/errors";
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
      const project = await client.projects.create({
        name: "test_project",
        layout: {},
      });
      const created = await client.tables.create(project.key, {
        name: "retrieve_test",
      });

      const { result } = renderHook(() => Table.useRetrieve({ key: created.key }), {
        wrapper,
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data?.key).toEqual(created.key);
      expect(result.current.data?.name).toEqual("retrieve_test");
    });

    it("should cache retrieved tables", async () => {
      const project = await client.projects.create({
        name: "cache_project",
        layout: {},
      });
      const created = await client.tables.create(project.key, {
        name: "cached_table",
      });

      const { result: result1 } = renderHook(
        () => Table.useRetrieve({ key: created.key }),
        { wrapper },
      );
      await waitFor(() => expect(result1.current.variant).toEqual("success"));

      const { result: result2 } = renderHook(
        () => Table.useRetrieve({ key: created.key }),
        { wrapper },
      );
      await waitFor(() => expect(result2.current.variant).toEqual("success"));
      expect(result2.current.data).toEqual(result1.current.data);
    });
  });

  describe("useCreate", () => {
    it("should create a new table", async () => {
      const project = await client.projects.create({
        name: "create_project",
        layout: {},
      });

      const { result } = renderHook(() => Table.useCreate(), { wrapper });

      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({
          key,
          project: project.key,
          name: "created_table",
        });
      });

      expect(result.current.variant).toEqual("success");
      expect(result.current.data?.name).toEqual("created_table");
      expect(result.current.data?.project).toEqual(project.key);

      const retrieved = await client.tables.retrieve({ key });
      expect(retrieved.name).toEqual("created_table");
    });

    it("should cache the created table for subsequent retrieves", async () => {
      const project = await client.projects.create({
        name: "store_project",
        layout: {},
      });

      const { result: createResult } = renderHook(() => Table.useCreate(), {
        wrapper,
      });

      const key = uuid.create();
      await act(async () => {
        await createResult.current.updateAsync({
          key,
          project: project.key,
          name: "stored_table",
        });
      });

      const { result: retrieveResult } = renderHook(() => Table.useRetrieve({ key }), {
        wrapper,
      });
      await waitFor(() => expect(retrieveResult.current.variant).toEqual("success"));
      expect(retrieveResult.current.data?.name).toEqual("stored_table");
    });

    it("should initialize a 2x2 layout of empty text cells when rows and columns are empty", async () => {
      const project = await client.projects.create({
        name: "default_layout_project",
        layout: {},
      });

      const { result } = renderHook(() => Table.useCreate(), { wrapper });

      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({
          key,
          project: project.key,
          name: "default_layout_table",
        });
      });
      expect(result.current.variant).toEqual("success");

      const retrieved = await client.tables.retrieve({ key });
      expect(retrieved.rows).toHaveLength(2);
      expect(retrieved.columns).toHaveLength(2);
      expect(retrieved.rows[0].cells).toHaveLength(2);
      expect(retrieved.rows[1].cells).toHaveLength(2);
      expect(Object.keys(retrieved.cells)).toHaveLength(4);
      for (const cell of Object.values(retrieved.cells)) {
        expect(cell.variant).toEqual("text");
        expect(cell.props.value).toEqual("");
        expect(cell.props.level).toEqual("h5");
      }
    });

    it("should not apply defaults when rows are provided", async () => {
      const project = await client.projects.create({
        name: "explicit_layout_project",
        layout: {},
      });

      const { result } = renderHook(() => Table.useCreate(), { wrapper });

      const key = uuid.create();
      await act(async () => {
        await result.current.updateAsync({
          key,
          project: project.key,
          name: "explicit_layout",
          rows: [{ size: 40, cells: ["x"] }],
          columns: [{ size: 80 }],
          cells: { x: { key: "x", variant: "value", props: { units: "psi" } } },
        });
      });
      expect(result.current.variant).toEqual("success");

      const retrieved = await client.tables.retrieve({ key });
      expect(retrieved.rows).toHaveLength(1);
      expect(retrieved.columns).toHaveLength(1);
      expect(retrieved.rows[0].cells).toEqual(["x"]);
      expect(retrieved.cells.x.variant).toEqual("value");
    });

    it("should not commit the table when the server rejects the create", async () => {
      const key = uuid.create();
      const { result } = renderHook(() => Table.useCreate(), { wrapper });

      // Use a fake project key so the server-side ontology insert fails.
      await act(async () => {
        await result.current.updateAsync({
          key,
          project: uuid.create(),
          name: "rollback_table",
        });
      });
      expect(result.current.variant).toEqual("error");
      await expect(client.tables.retrieve({ key })).rejects.toThrow(NotFoundError);
    });
  });

  describe("useRename", () => {
    it("should rename a table", async () => {
      const project = await client.projects.create({
        name: "rename_project",
        layout: {},
      });
      const created = await client.tables.create(project.key, {
        name: "original_name",
      });

      const { result } = renderHook(
        () => {
          const retrieve = Table.useRetrieve({ key: created.key });
          const rename = Table.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );

      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      expect(result.current.retrieve.data?.name).toEqual("original_name");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: created.key,
          name: "renamed_table",
        });
      });

      const retrieved = await client.tables.retrieve({ key: created.key });
      expect(retrieved.name).toEqual("renamed_table");
    });

    it("should update cached table after rename", async () => {
      const project = await client.projects.create({
        name: "rename_cache_project",
        layout: {},
      });
      const created = await client.tables.create(project.key, {
        name: "cache_original",
      });

      const { result } = renderHook(
        () => ({
          retrieve: Table.useRetrieve({ key: created.key }),
          rename: Table.useRename(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      await act(async () => {
        await result.current.rename.updateAsync({
          key: created.key,
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
      const project = await client.projects.create({
        name: "delete_project",
        layout: {},
      });
      const created = await client.tables.create(project.key, {
        name: "delete_single",
      });

      const { result } = renderHook(() => Table.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync(created.key);
      });
      expect(result.current.variant).toEqual("success");
      await expect(client.tables.retrieve({ key: created.key })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should delete multiple tables", async () => {
      const project = await client.projects.create({
        name: "delete_multi_project",
        layout: {},
      });
      const created1 = await client.tables.create(project.key, {
        name: "delete_multi_1",
      });
      const created2 = await client.tables.create(project.key, {
        name: "delete_multi_2",
      });

      const { result } = renderHook(() => Table.useDelete(), { wrapper });

      await act(async () => {
        await result.current.updateAsync([created1.key, created2.key]);
      });

      expect(result.current.variant).toEqual("success");

      await expect(client.tables.retrieve({ key: created1.key })).rejects.toThrow(
        NotFoundError,
      );
      await expect(client.tables.retrieve({ key: created2.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("useDispatch", () => {
    const createTable = async () => {
      const proj = await client.projects.create({
        name: `dispatch_ws_${uuid.create()}`,
        layout: {},
      });
      return await client.tables.create(proj.key, {
        name: "dispatch_test",
        rows: [{ size: 36, cells: ["a", "b"] }],
        columns: [{ size: 80 }, { size: 100 }],
        cells: {
          a: { key: "a", variant: "text", props: { value: "A" } },
          b: { key: "b", variant: "text", props: { value: "B" } },
        },
      });
    };

    it("should apply a dispatched action and update the cached table", async () => {
      const created = await createTable();
      const { result } = renderHook(
        () => ({
          retrieve: Table.useRetrieve({ key: created.key }),
          dispatch: Table.useDispatch(),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [table.rename({ name: "after_dispatch" })],
        });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.name).toEqual("after_dispatch"),
      );
    });

    it("should restore prior state after a set_cell dispatch is undone", async () => {
      const created = await createTable();
      const { result } = renderHook(
        () => ({
          retrieve: Table.useRetrieve({ key: created.key }),
          dispatch: Table.useDispatch(),
          undo: Table.useUndo({ key: created.key }),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            table.setCell({
              cell: { key: "a", variant: "value", props: { units: "psi" } },
            }),
          ],
        });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.cells.a.variant).toEqual("value"),
      );
      await act(async () => result.current.undo.undo());
      await waitFor(() => {
        expect(result.current.retrieve.data?.cells.a.variant).toEqual("text");
        expect(result.current.retrieve.data?.cells.a.props).toEqual({ value: "A" });
      });
    });

    it("should re-apply a set_cell dispatch after undo and redo", async () => {
      const created = await createTable();
      const { result } = renderHook(
        () => ({
          retrieve: Table.useRetrieve({ key: created.key }),
          dispatch: Table.useDispatch(),
          undo: Table.useUndo({ key: created.key }),
          redo: Table.useRedo({ key: created.key }),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            table.setCell({
              cell: { key: "a", variant: "value", props: { units: "psi" } },
            }),
          ],
        });
      });
      await waitFor(() =>
        expect(result.current.retrieve.data?.cells.a.variant).toEqual("value"),
      );
      await act(async () => result.current.undo.undo());
      await waitFor(() =>
        expect(result.current.retrieve.data?.cells.a.variant).toEqual("text"),
      );
      await act(async () => result.current.redo.redo());
      await waitFor(() => {
        expect(result.current.retrieve.data?.cells.a.variant).toEqual("value");
        expect(result.current.retrieve.data?.cells.a.props).toEqual({ units: "psi" });
      });
    });

    it("should coalesce successive set_cell dispatches on the same cell into one undo step", async () => {
      const created = await createTable();
      const { result } = renderHook(
        () => ({
          retrieve: Table.useRetrieve({ key: created.key }),
          dispatch: Table.useDispatch(),
          undo: Table.useUndo({ key: created.key }),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      for (const value of ["A1", "A2", "A3"])
        await act(async () => {
          await result.current.dispatch.dispatchAsync({
            key: created.key,
            actions: [
              table.setCell({
                cell: { key: "a", variant: "text", props: { value } },
              }),
            ],
          });
        });
      await waitFor(() =>
        expect(result.current.retrieve.data?.cells.a.props.value).toEqual("A3"),
      );
      await act(async () => result.current.undo.undo());
      await waitFor(() =>
        expect(result.current.retrieve.data?.cells.a.props.value).toEqual("A"),
      );
    });

    it("should not coalesce set_cell dispatches across different cells", async () => {
      const created = await createTable();
      const { result } = renderHook(
        () => ({
          retrieve: Table.useRetrieve({ key: created.key }),
          dispatch: Table.useDispatch(),
          undo: Table.useUndo({ key: created.key }),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            table.setCell({
              cell: { key: "a", variant: "text", props: { value: "A1" } },
            }),
          ],
        });
      });
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            table.setCell({
              cell: { key: "b", variant: "text", props: { value: "B1" } },
            }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.retrieve.data?.cells.a.props.value).toEqual("A1");
        expect(result.current.retrieve.data?.cells.b.props.value).toEqual("B1");
      });
      await act(async () => result.current.undo.undo());
      await waitFor(() => {
        expect(result.current.retrieve.data?.cells.a.props.value).toEqual("A1");
        expect(result.current.retrieve.data?.cells.b.props.value).toEqual("B");
      });
    });

    it("should coalesce successive resize_row dispatches under the resize kind", async () => {
      const created = await createTable();
      const { result } = renderHook(
        () => ({
          retrieve: Table.useRetrieve({ key: created.key }),
          dispatch: Table.useDispatch(),
          undo: Table.useUndo({ key: created.key }),
        }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      for (const size of [40, 50, 60])
        await act(async () => {
          await result.current.dispatch.dispatchAsync({
            key: created.key,
            actions: [table.resizeRow({ index: 0, size })],
          });
        });
      await waitFor(() =>
        expect(result.current.retrieve.data?.rows[0].size).toEqual(60),
      );
      await act(async () => result.current.undo.undo());
      await waitFor(() =>
        expect(result.current.retrieve.data?.rows[0].size).toEqual(36),
      );
    });
  });

  describe("useEnsureRetrieved", () => {
    // Single-hook bootstrap component so the suspending useEnsureRetrieved
    // is not followed by additional hooks, which trips a React 19 concurrent
    // replay warning (same pattern as schematic queries.spec.tsx).
    const loadTable = async (
      Wrapper: FC<PropsWithChildren>,
      key: table.Key,
    ): Promise<void> => {
      const Bootstrap = (): ReactElement => {
        Table.useEnsureRetrieved({ key });
        return <div data-testid="loaded" />;
      };
      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary loading={null}>
              <Bootstrap />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });
      await utils.findByTestId("loaded");
    };

    it("populates the cache so downstream selectors resolve", async () => {
      const proj = await client.projects.create({
        name: `ensure_ws_${uuid.create()}`,
        layout: {},
      });
      const created = await client.tables.create(proj.key, {
        name: "ensure_test",
        rows: [{ size: 30, cells: ["a"] }],
        columns: [{ size: 80 }],
        cells: { a: { key: "a", variant: "text", props: { value: "A" } } },
      });
      await loadTable(wrapper, created.key);
      const { result } = renderHook(() => Table.useSelectName({ key: created.key }), {
        wrapper,
      });
      expect(result.current).toEqual("ensure_test");
    });

    it("does not suspend when the table is already in the store", async () => {
      const proj = await client.projects.create({
        name: `fastpath_ws_${uuid.create()}`,
        layout: {},
      });
      const created = await client.tables.create(proj.key, {
        name: "fastpath_test",
        rows: [{ size: 30, cells: ["a"] }],
        columns: [{ size: 80 }],
        cells: { a: { key: "a", variant: "text", props: { value: "A" } } },
      });
      await loadTable(wrapper, created.key);

      const Wrapper = wrapper;
      // With the store warm, the fast-path resolves without suspending.
      const Probe = (): ReactElement => {
        Table.useEnsureRetrieved({ key: created.key });
        return <div data-testid="ready" />;
      };
      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary loading={<div data-testid="fallback" />}>
              <Probe />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });
      expect(utils.queryByTestId("fallback")).toBeNull();
      expect(utils.queryByTestId("ready")).toBeTruthy();
    });
  });

  describe("selectors", () => {
    const createTable = async () => {
      const proj = await client.projects.create({
        name: `selector_ws_${uuid.create()}`,
        layout: {},
      });
      return await client.tables.create(proj.key, {
        name: "selector_test",
        rows: [
          { size: 30, cells: ["a", "b"] },
          { size: 40, cells: ["c", "d"] },
        ],
        columns: [{ size: 80 }, { size: 100 }],
        cells: {
          a: { key: "a", variant: "text", props: { value: "A" } },
          b: { key: "b", variant: "text", props: { value: "B" } },
          c: { key: "c", variant: "text", props: { value: "C" } },
          d: { key: "d", variant: "text", props: { value: "D" } },
        },
      });
    };

    const loadAndSelect = async <T,>(
      key: table.Key,
      hook: () => T,
    ): Promise<ReturnType<typeof renderHook<T, unknown>>> => {
      const retrieve = renderHook(() => Table.useRetrieve({ key }), { wrapper });
      await waitFor(() => expect(retrieve.result.current.variant).toEqual("success"));
      return renderHook(hook, { wrapper });
    };

    it("useSelectName returns the table's name and updates after a rename", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () => ({
        name: Table.useSelectName({ key: created.key }),
        rename: Table.useRename(),
      }));
      expect(result.current.name).toEqual("selector_test");
      await act(async () => {
        await result.current.rename.updateAsync({
          key: created.key,
          name: "selector_renamed",
        });
      });
      await waitFor(() => expect(result.current.name).toEqual("selector_renamed"));
    });

    it("useSelectRows returns the table's rows and updates after addRow", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () => ({
        rows: Table.useSelectRows({ key: created.key }),
        dispatch: Table.useDispatch(),
      }));
      expect(result.current.rows).toHaveLength(2);
      expect(result.current.rows[0].cells).toEqual(["a", "b"]);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            table.addRow({
              index: 2,
              size: 50,
              cells: [
                { key: "e", variant: "text", props: { value: "E" } },
                { key: "f", variant: "text", props: { value: "F" } },
              ],
            }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.rows).toHaveLength(3);
        expect(result.current.rows[2].cells).toEqual(["e", "f"]);
      });
    });

    it("useSelectColumns returns the table's columns and updates after a resize", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () => ({
        columns: Table.useSelectColumns({ key: created.key }),
        dispatch: Table.useDispatch(),
      }));
      expect(result.current.columns).toHaveLength(2);
      expect(result.current.columns[0].size).toEqual(80);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [table.resizeCol({ index: 0, size: 200 })],
        });
      });
      await waitFor(() => expect(result.current.columns[0].size).toEqual(200));
    });

    it("useSelectCell returns the cell for a known key and updates after setCell", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () => ({
        cell: Table.useSelectCell({ key: created.key, cellKey: "a" }),
        dispatch: Table.useDispatch(),
      }));
      const initial = result.current.cell;
      expect(initial?.variant).toEqual("text");
      if (initial?.variant === "text") expect(initial.props.value).toEqual("A");
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            table.setCell({
              cell: { key: "a", variant: "value", props: { units: "psi" } },
            }),
          ],
        });
      });
      await waitFor(() => {
        const next = result.current.cell;
        expect(next?.variant).toEqual("value");
        if (next?.variant === "value") expect(next.props.units).toEqual("psi");
      });
    });

    it("useSelectCell returns undefined for an unknown cell key", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () =>
        Table.useSelectCell({ key: created.key, cellKey: "ghost" }),
      );
      expect(result.current).toBeUndefined();
    });

    it("useSelectCells returns the requested cells keyed by id", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () =>
        Table.useSelectCells({ key: created.key, cellKeys: ["a", "c"] }),
      );
      expect(Array.from(result.current.keys())).toEqual(["a", "c"]);
      const a = result.current.get("a");
      const c = result.current.get("c");
      if (a?.variant === "text") expect(a.props.value).toEqual("A");
      if (c?.variant === "text") expect(c.props.value).toEqual("C");
    });

    it("useSelectCells omits missing keys without throwing", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () =>
        Table.useSelectCells({ key: created.key, cellKeys: ["a", "ghost", "c"] }),
      );
      expect(Array.from(result.current.keys())).toEqual(["a", "c"]);
      expect(result.current.has("ghost")).toBe(false);
    });

    it("useSelectCells returns an empty map when cellKeys is empty", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () =>
        Table.useSelectCells({ key: created.key, cellKeys: [] }),
      );
      expect(result.current.size).toBe(0);
    });

    it("useSelectCells keeps its reference when an unrelated cell changes", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () => ({
        cells: Table.useSelectCells({ key: created.key, cellKeys: ["a", "b"] }),
        dispatch: Table.useDispatch(),
      }));
      const initial = result.current.cells;
      expect(initial.size).toBe(2);
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            table.setCell({
              cell: { key: "c", variant: "value", props: { units: "psi" } },
            }),
          ],
        });
      });
      expect(result.current.cells).toBe(initial);
    });

    it("useSelectCells returns a new map when one of the requested cells changes", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () => ({
        cells: Table.useSelectCells({ key: created.key, cellKeys: ["a", "b"] }),
        dispatch: Table.useDispatch(),
      }));
      const initial = result.current.cells;
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            table.setCell({
              cell: { key: "a", variant: "value", props: { units: "psi" } },
            }),
          ],
        });
      });
      await waitFor(() => {
        expect(result.current.cells).not.toBe(initial);
        expect(result.current.cells.get("a")?.variant).toEqual("value");
      });
    });

    it("useCellPosition returns the grid coordinates of a known cell and null for an unknown one", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () => ({
        a: Table.useCellPosition({ key: created.key, cellKey: "a" }),
        d: Table.useCellPosition({ key: created.key, cellKey: "d" }),
        ghost: Table.useCellPosition({ key: created.key, cellKey: "ghost" }),
      }));
      expect(result.current.a).toEqual({ x: 0, y: 0 });
      expect(result.current.d).toEqual({ x: 1, y: 1 });
      expect(result.current.ghost).toBeNull();
    });

    it("useCellPosition updates when a row is inserted above the cell", async () => {
      const created = await createTable();
      const { result } = await loadAndSelect(created.key, () => ({
        a: Table.useCellPosition({ key: created.key, cellKey: "a" }),
        dispatch: Table.useDispatch(),
      }));
      expect(result.current.a).toEqual({ x: 0, y: 0 });
      await act(async () => {
        await result.current.dispatch.dispatchAsync({
          key: created.key,
          actions: [
            table.addRow({
              index: 0,
              size: 30,
              cells: [
                { key: "x", variant: "text", props: {} },
                { key: "y", variant: "text", props: {} },
              ],
            }),
          ],
        });
      });
      await waitFor(() => expect(result.current.a).toEqual({ x: 0, y: 1 }));
    });
  });
});

describe("findCellPosition", () => {
  const rows: table.Row[] = [
    { size: 30, cells: ["a", "b", "c"] },
    { size: 30, cells: ["d", "e", "f"] },
  ];

  it("returns the (x, y) coordinates of a known cell", () => {
    expect(Table.findCellPosition(rows, "a")).toEqual({ x: 0, y: 0 });
    expect(Table.findCellPosition(rows, "e")).toEqual({ x: 1, y: 1 });
    expect(Table.findCellPosition(rows, "f")).toEqual({ x: 2, y: 1 });
  });

  it("returns null when the cell is absent from every row", () => {
    expect(Table.findCellPosition(rows, "ghost")).toBeNull();
  });

  it("returns null for empty rows", () => {
    expect(Table.findCellPosition([], "a")).toBeNull();
  });
});

describe("nextCellPosition", () => {
  const rows: table.Row[] = [
    { size: 30, cells: ["a", "b"] },
    { size: 30, cells: ["c", "d"] },
  ];

  it("steps forward within a row", () => {
    expect(Table.nextCellPosition(rows, { x: 0, y: 0 }, 1)).toEqual({ x: 1, y: 0 });
  });

  it("wraps forward to the start of the next row at row-end", () => {
    expect(Table.nextCellPosition(rows, { x: 1, y: 0 }, 1)).toEqual({ x: 0, y: 1 });
  });

  it("returns null when stepping forward past the last cell", () => {
    expect(Table.nextCellPosition(rows, { x: 1, y: 1 }, 1)).toBeNull();
  });

  it("steps backward within a row", () => {
    expect(Table.nextCellPosition(rows, { x: 1, y: 1 }, -1)).toEqual({ x: 0, y: 1 });
  });

  it("wraps backward to the end of the previous row at row-start", () => {
    expect(Table.nextCellPosition(rows, { x: 0, y: 1 }, -1)).toEqual({ x: 1, y: 0 });
  });

  it("returns null when stepping backward past the first cell", () => {
    expect(Table.nextCellPosition(rows, { x: 0, y: 0 }, -1)).toBeNull();
  });

  it("returns null when pos's row is out of range", () => {
    expect(Table.nextCellPosition(rows, { x: 0, y: 5 }, 1)).toBeNull();
  });

  it("skips empty neighbor rows when wrapping", () => {
    const ragged: table.Row[] = [
      { size: 30, cells: ["a"] },
      { size: 30, cells: [] },
    ];
    expect(Table.nextCellPosition(ragged, { x: 0, y: 0 }, 1)).toBeNull();
  });
});

describe("cellsInRegion", () => {
  const rows: table.Row[] = [
    { size: 30, cells: ["a", "b", "c"] },
    { size: 30, cells: ["d", "e", "f"] },
    { size: 30, cells: ["g", "h", "i"] },
  ];

  it("returns the inclusive rectangle of cell keys", () => {
    expect(Table.cellsInRegion(rows, { x: 0, y: 0 }, { x: 1, y: 1 })).toEqual([
      "a",
      "b",
      "d",
      "e",
    ]);
  });

  it("normalizes start and end so order does not matter", () => {
    expect(Table.cellsInRegion(rows, { x: 2, y: 2 }, { x: 1, y: 1 })).toEqual([
      "e",
      "f",
      "h",
      "i",
    ]);
  });

  it("returns a single cell when start equals end", () => {
    expect(Table.cellsInRegion(rows, { x: 1, y: 1 }, { x: 1, y: 1 })).toEqual(["e"]);
  });

  it("skips rows past the bottom of the table", () => {
    expect(Table.cellsInRegion(rows, { x: 0, y: 0 }, { x: 0, y: 10 })).toEqual([
      "a",
      "d",
      "g",
    ]);
  });

  it("skips column slots past the right of a ragged row", () => {
    const ragged: table.Row[] = [
      { size: 30, cells: ["a", "b"] },
      { size: 30, cells: ["c"] },
    ];
    expect(Table.cellsInRegion(ragged, { x: 0, y: 0 }, { x: 1, y: 1 })).toEqual([
      "a",
      "b",
      "c",
    ]);
  });
});
