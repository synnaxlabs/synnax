// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  NotFoundError,
  ontology,
  project as clientProject,
  table as clientTable,
  type table,
} from "@synnaxlabs/client";
import { List, Text } from "@synnaxlabs/pluto";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

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
  resolveFocusedTab,
  stubClipboardWriteText,
  uniqueName,
} from "@/testutil";

const Item = Table.TREE_ITEMS.table;

const createTable = async (): Promise<table.Table> =>
  await client.tables.create(await project(), { name: uniqueName("table") });

interface SetupParams {
  tables: table.Table[];
  overrides?: Partial<Tree.BaseProps>;
  withCluster?: boolean;
}

const renderMenu = async ({ tables, overrides, withCluster = false }: SetupParams) => {
  const ids = tables.map((t) => clientTable.ontologyID(t.key));
  const store = await createTestStore({
    preloadedState: {
      ...createPreloadedState(tables[0].key),
      ...(withCluster ? createClusterState([createCluster("test")], "test") : {}),
    },
  });
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store, overrides }),
    selection: createSelection({ ids }),
    state: createState(tables.map((t, i) => createResource(ids[i], t.name))),
  };
  const { wrapper } = await createConsoleWrapper({ client, store });
  const Menu = Item.ContextMenu;
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

    it("deletes the table and its session state after confirmation", async () => {
      const t = await createTable();
      const { store } = await renderMenu({ tables: [t] });
      fireEvent.click(await screen.findByText("Delete"));
      await waitFor(() =>
        expect(
          screen.getByText(`Are you sure you want to delete ${t.name}?`),
        ).toBeTruthy(),
      );
      fireEvent.click(findLastButton("Delete"));
      await waitFor(async () => {
        await expect(client.tables.retrieve({ key: t.key })).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
      expect(
        Session.Table.selectSliceState(store.getState()).tables[t.key],
      ).toBeUndefined();
    });

    it("renames the table on the cluster", async () => {
      const t = await createTable();
      const { itemID } = await renderMenu({ tables: [t] });
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
    it("retrieves the table and opens it as a tab when double-clicked", async () => {
      const t = await createTable();
      const { store } = await renderOntologyTree({
        client,
        root: clientProject.ontologyID(await project()),
        items: Table.TREE_ITEMS,
      });
      fireEvent.doubleClick(await findTreeRow(t.name));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.key).toBe(t.key);
    });
  });

  describe("haulItems", () => {
    it("returns a mosaic tab haul item for the resource", () => {
      const id = clientTable.ontologyID("11111111-1111-1111-1111-111111111111");
      const items = Item.haulItems(createResource(id, "My Table"));
      expect(items).toHaveLength(1);
      expect(items[0].key).toContain("table:11111111-1111-1111-1111-111111111111");
    });
  });
});
