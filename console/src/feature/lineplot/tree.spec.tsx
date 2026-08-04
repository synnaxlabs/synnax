// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  lineplot as clientLineplot,
  type lineplot,
  NotFoundError,
  ontology,
  project as clientProject,
} from "@synnaxlabs/client";
import { List, Text } from "@synnaxlabs/pluto";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { client, createPreloadedState, project } from "@/feature/lineplot/testutil";
import { Modals } from "@/platform/modals";
import { findButton, findLastButton } from "@/platform/modals/testutil";
import { type Tree } from "@/platform/tree";
import {
  createBaseProps,
  createResource,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { findTreeRow, renderOntologyTree } from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import { createCluster, createClusterState } from "@/session/cluster/testutil";
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

const Item = LinePlot.TREE_ITEMS.lineplot;

const createLinePlot = async (): Promise<lineplot.LinePlot> =>
  await client.lineplots.create(await project(), { name: uniqueName("plot") });

interface SetupParams {
  plots: lineplot.LinePlot[];
  overrides?: Partial<Tree.BaseProps>;
  withCluster?: boolean;
}

const renderMenu = async ({ plots, overrides, withCluster = false }: SetupParams) => {
  const store = await createTestStore({
    preloadedState: {
      ...createPreloadedState(plots[0].key),
      ...(withCluster ? createClusterState([createCluster("test")], "test") : {}),
    },
  });
  const Menu = Item.ContextMenu;
  if (Menu == null) throw new Error("TreeContextMenu not defined");
  const buildUI = (current: lineplot.LinePlot[]) => {
    const ids = current.map((p) => clientLineplot.ontologyID(p.key));
    const props: Tree.ContextMenuProps = {
      ...createBaseProps({ client, store, overrides }),
      selection: createSelection({ ids }),
      state: createState(current.map((p, i) => createResource(ids[i], p.name))),
    };
    const itemID = List.itemNameID(ontology.idToString(ids[0]));
    const element = (
      <>
        <Menu {...props} />
        <Text.MaybeEditable id={itemID} value={current[0].name} onChange={() => {}} />
        <Modals.Stack />
      </>
    );
    return { props, itemID, element };
  };
  const { props, itemID, element } = buildUI(plots);
  const { wrapper } = await createConsoleWrapper({ client, store });
  const utils = render(element, { wrapper });
  const rerenderWith = (current: lineplot.LinePlot[]): void => {
    utils.rerender(buildUI(current).element);
  };
  return { store, props, itemID, rerenderWith };
};

describe("lineplot/ontology", () => {
  describe("TreeContextMenu", () => {
    it("copies the plot's properties to the clipboard", async () => {
      const writeText = stubClipboardWriteText();
      const plot = await createLinePlot();
      await renderMenu({ plots: [plot] });
      fireEvent.click(await screen.findByText("Copy properties"));
      await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    });

    it("hides single-selection items for multi-selections", async () => {
      const [a, b] = [await createLinePlot(), await createLinePlot()];
      await renderMenu({ plots: [a, b] });
      expect(await screen.findByText("Delete")).toBeDefined();
      expect(screen.queryByText("Rename")).toBeNull();
      expect(screen.queryByText("Export")).toBeNull();
      expect(screen.queryByText("Copy link")).toBeNull();
    });

    it("deletes the plot and its session state after confirmation", async () => {
      const plot = await createLinePlot();
      const { store } = await renderMenu({ plots: [plot] });
      fireEvent.click(await screen.findByText("Delete"));
      await waitFor(() =>
        expect(
          screen.getByText(`Are you sure you want to delete ${plot.name}?`),
        ).toBeTruthy(),
      );
      fireEvent.click(findLastButton("Delete"));
      await waitFor(async () => {
        await expect(client.lineplots.retrieve(plot.key)).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
      expect(
        Session.LinePlot.selectSliceState(store.getState()).plots[plot.key],
      ).toBeUndefined();
    });

    it("renames the plot on the cluster", async () => {
      const plot = await createLinePlot();
      const { itemID } = await renderMenu({ plots: [plot] });
      fireEvent.click(await screen.findByText("Rename"));
      const el = await awaitTextEditing(itemID);
      const newName = uniqueName("renamed");
      await act(async () => {
        commitTextEdit(el, newName);
      });
      await waitFor(async () => {
        const renamed = await client.lineplots.retrieve(plot.key);
        expect(renamed.name).toBe(newName);
      });
    });

    it("exports the plot as a JSON download", async () => {
      const downloads = captureBrowserDownloads();
      const plot = await createLinePlot();
      await renderMenu({ plots: [plot] });
      fireEvent.click(await screen.findByText("Export"));
      await waitFor(() => expect(downloads.anchors).toHaveLength(1));
      expect(downloads.anchors[0].download).toBe(`${plot.name}.json`);
    });

    it("copies a deep link to the clipboard", async () => {
      const writeText = stubClipboardWriteText();
      const plot = await createLinePlot();
      await renderMenu({ plots: [plot], withCluster: true });
      fireEvent.click(await screen.findByText("Copy link"));
      await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
      expect(writeText.mock.calls[0][0]).toContain(`lineplot/${plot.key}`);
    });

    it("leaves the plot in place when deletion is canceled", async () => {
      const plot = await createLinePlot();
      const control = await createLinePlot();
      const { rerenderWith } = await renderMenu({ plots: [plot] });
      fireEvent.click(await screen.findByText("Delete"));
      await waitFor(() =>
        expect(
          screen.getByText(`Are you sure you want to delete ${plot.name}?`),
        ).toBeTruthy(),
      );
      fireEvent.click(findButton("Cancel"));
      await waitFor(() =>
        expect(
          screen.queryByText(`Are you sure you want to delete ${plot.name}?`),
        ).toBeNull(),
      );
      rerenderWith([control]);
      fireEvent.click(await screen.findByText("Delete"));
      await waitFor(() =>
        expect(
          screen.getByText(`Are you sure you want to delete ${control.name}?`),
        ).toBeTruthy(),
      );
      fireEvent.click(findLastButton("Delete"));
      await waitFor(async () => {
        await expect(client.lineplots.retrieve(control.key)).rejects.toSatisfy((e) =>
          NotFoundError.matches(e),
        );
      });
      await expect(client.lineplots.retrieve(plot.key)).resolves.toBeDefined();
    });
  });

  describe("onSelect", () => {
    it("retrieves the plot and opens it as a tab when double-clicked", async () => {
      const plot = await createLinePlot();
      const { store } = await renderOntologyTree({
        client,
        root: clientProject.ontologyID(await project()),
        items: LinePlot.TREE_ITEMS,
      });
      fireEvent.doubleClick(await findTreeRow(plot.name));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.key).toBe(plot.key);
    });
  });

  describe("haulItems", () => {
    it("returns a mosaic tab haul item for the resource", () => {
      const id = clientLineplot.ontologyID("11111111-1111-1111-1111-111111111111");
      const items = Item.haulItems(createResource(id, "My Plot"));
      expect(items).toHaveLength(1);
      expect(items[0].key).toContain("lineplot:11111111-1111-1111-1111-111111111111");
    });
  });
});
