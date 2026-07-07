// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ontology,
  project as clientProject,
  table as clientTable,
  type table,
} from "@synnaxlabs/client";
import { List, Text } from "@synnaxlabs/pluto";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Table } from "@/feature/table";
import { client, createPreloadedState, project } from "@/feature/table/testutil";
import { createCluster, createClusterState } from "@/platform/cluster/testutil";
import { Modals } from "@/platform/modals";
import { findLastButton } from "@/platform/modals/testutil";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createResource,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { findTreeRow, renderOntologyTree } from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import {
  awaitTextEditing,
  captureBrowserDownloads,
  commitTextEdit,
  createConsoleWrapper,
  createTestStore,
  stubClipboardWriteText,
  uniqueName,
} from "@/testutil";

const createTable = async (): Promise<table.Table> =>
  await client.tables.create(await project(), { name: uniqueName("table") });

const tableExists = async (key: string): Promise<boolean> => {
  try {
    await client.tables.retrieve({ key });
    return true;
  } catch {
    return false;
  }
};

interface SetupArgs {
  tables: table.Table[];
  overrides?: Partial<Tree.BaseProps>;
  withCluster?: boolean;
}

const renderMenu = async ({ tables, overrides, withCluster = false }: SetupArgs) => {
  const ids = tables.map((t) => clientTable.ontologyID(t.key));
  const store = await createTestStore({
    preloadedState: {
      ...createPreloadedState(tables[0].key, tables[0].name),
      ...(withCluster ? createClusterState([createCluster("test")], "test") : {}),
    },
  });
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store, overrides }),
    selection: createSelection({ ids }),
    state: createState(tables.map((t, i) => createResource(ids[i], t.name))),
  };
  const { wrapper } = await createConsoleWrapper({ client, store });
  const Menu = Table.TREE_ITEM.ContextMenu;
  if (Menu == null) throw new Error("TreeContextMenu not defined");
  const itemID = List.itemNameID(ontology.idToString(ids[0]));
  render(
    <>
      <Menu {...props} />
      <Text.MaybeEditable id={itemID} value={tables[0].name} onChange={() => {}} />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  return { store, props, itemID };
};

describe("table/ontology", () => {
  describe("TreeContextMenu", () => {
    it("renders single-selection items once permissions resolve", async () => {
      const t = await createTable();
      await renderMenu({ tables: [t] });
      expect(await screen.findByText("Rename")).toBeDefined();
      expect(screen.getByText("Delete")).toBeDefined();
      expect(screen.getByText("Export")).toBeDefined();
      expect(screen.getByText("Copy link")).toBeDefined();
      expect(screen.getByText("Copy properties")).toBeDefined();
    });

    it("hides single-selection items for multi-selections", async () => {
      const [a, b] = [await createTable(), await createTable()];
      await renderMenu({ tables: [a, b] });
      expect(await screen.findByText("Delete")).toBeDefined();
      expect(screen.queryByText("Export")).toBeNull();
      expect(screen.queryByText("Copy link")).toBeNull();
    });

    it("deletes the table, its layout, and its session state after confirmation", async () => {
      const t = await createTable();
      const removeLayout = vi.fn();
      const { store } = await renderMenu({
        tables: [t],
        overrides: { removeLayout },
      });
      fireEvent.click(await screen.findByText("Delete"));
      await waitFor(() =>
        expect(
          screen.getByText(`Are you sure you want to delete ${t.name}?`),
        ).toBeTruthy(),
      );
      fireEvent.click(findLastButton("Delete"));
      await waitFor(async () => expect(await tableExists(t.key)).toBe(false));
      expect(removeLayout).toHaveBeenCalledWith(t.key);
      expect(
        Session.Table.selectSliceState(store.getState()).tables[t.key],
      ).toBeUndefined();
    });

    it("renames the table on the cluster and in the layout store", async () => {
      const t = await createTable();
      const { store, itemID } = await renderMenu({ tables: [t] });
      fireEvent.click(await screen.findByText("Rename"));
      const el = await awaitTextEditing(itemID);
      const newName = uniqueName("renamed");
      await act(async () => {
        commitTextEdit(el, newName);
      });
      await waitFor(async () => {
        const renamed = await client.tables.retrieve({ key: t.key });
        expect(renamed.name).toBe(newName);
      });
      expect(Session.Layout.select(store.getState(), t.key)?.name).toBe(newName);
    });

    it("exports the table as a JSON download", async () => {
      const downloads = captureBrowserDownloads();
      const t = await createTable();
      await renderMenu({ tables: [t] });
      fireEvent.click(await screen.findByText("Export"));
      await waitFor(() => expect(downloads.anchors).toHaveLength(1));
      expect(downloads.anchors[0].download).toBe(`${t.name}.json`);
    });

    it("copies a deep link to the clipboard", async () => {
      const writeText = stubClipboardWriteText();
      const t = await createTable();
      await renderMenu({ tables: [t], withCluster: true });
      fireEvent.click(await screen.findByText("Copy link"));
      await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
      expect(writeText.mock.calls[0][0]).toContain(`table/${t.key}`);
    });
  });

  describe("onSelect", () => {
    it("places the table's layout when a tree row is double-clicked", async () => {
      const t = await createTable();
      const { store } = await renderOntologyTree({
        client,
        root: clientProject.ontologyID(await project()),
        items: { table: Table.TREE_ITEM },
      });
      fireEvent.doubleClick(await findTreeRow(t.name));
      await waitFor(() =>
        expect(Session.Layout.select(store.getState(), t.key)?.name).toBe(t.name),
      );
      expect(Session.Layout.select(store.getState(), t.key)?.type).toBe(
        Table.LAYOUT_TYPE,
      );
    });
  });

  describe("haulItems", () => {
    it("returns a mosaic tab haul item for the resource", () => {
      const id = clientTable.ontologyID("11111111-1111-1111-1111-111111111111");
      const items = Table.TREE_ITEM.haulItems(createResource(id, "My Table"));
      expect(items).toHaveLength(1);
      expect(items[0].key).toContain("table:11111111-1111-1111-1111-111111111111");
    });
  });
});
