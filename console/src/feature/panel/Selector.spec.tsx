// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul, Mosaic, Panel as PPanel } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Selector } from "@/feature/panel/Selector";
import { Modals } from "@/platform/modals";
import { createPanelWrapper } from "@/platform/panel/testutil";
import { findModalButton } from "@/platform/tree/menuTestutil";
import { Session } from "@/session";
import { getIconButton, type TestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const createTab = (): panel.Tab => ({
  variant: "view",
  key: uuid.create(),
  type: "t",
  args: {},
});

const createProjectPanel = async (
  projectKey: project.Key,
  tab: panel.Tab = createTab(),
): Promise<panel.Panel> =>
  await client.panels.create({
    key: uuid.create(),
    name: uniqueName("panel"),
    parent: project.ontologyID(projectKey),
    root: { variant: "leaf", tabs: [tab] },
  });

// The strip's own order decides which panel is a neighbor, so the row is read back
// from the rendered pills instead of assumed from creation order.
const renderStrip = async (
  panels: panel.Panel[],
  projectKey: project.Key,
): Promise<{ store: TestStore; row: panel.Panel[] }> => {
  const { wrapper, store } = await createPanelWrapper({ client, project: projectKey });
  await act(async () => {
    render(
      <>
        <Selector />
        <Modals.Stack />
      </>,
      { wrapper },
    );
  });
  await waitFor(() =>
    panels.forEach(({ name }) => expect(screen.getByText(name)).toBeTruthy()),
  );
  const row = screen
    .getAllByRole("tab")
    .map((pill) => panels.find(({ name }) => pill.textContent?.includes(name)))
    .filter((pan) => pan != null);
  expect(row).toHaveLength(panels.length);
  return { store, row };
};

const deletePanel = async (pan: panel.Panel): Promise<void> => {
  fireEvent.contextMenu(screen.getByText(pan.name));
  fireEvent.click(await screen.findByText("Delete"));
  await screen.findByText(`Are you sure you want to delete ${pan.name}?`);
  await act(async () => {
    fireEvent.click(findModalButton("Delete"));
  });
};

// Stands in for the tab a mosaic drags: the same pluto drag source and payload the
// mosaic's own tab uses, without mounting a mosaic and its renderer registry.
const DRAG_HANDLE = "drag the tab";

interface TabDragSourceProps {
  source: panel.Key;
  tab: panel.Tab;
}

const TabDragSource = ({ source, tab }: TabDragSourceProps): ReactElement => {
  const { startDrag, onDragEnd } = Mosaic.useDragTab();
  return (
    <button
      id="drag-source"
      onDragStart={(e) =>
        startDrag(e, tab.key, PPanel.createTabDragPayload(source, tab))
      }
      onDragEnd={onDragEnd}
    >
      {DRAG_HANDLE}
    </button>
  );
};

// jsdom has no DragEvent, and testing-library's fallback drops the coordinate init the
// drop targets read. A MouseEvent carries it.
const fireDrop = (el: Element): boolean =>
  fireEvent(el, new MouseEvent("drop", { bubbles: true, clientX: 0, clientY: 0 }));

const fireDragOver = (el: Element): boolean =>
  fireEvent(el, new MouseEvent("dragover", { bubbles: true, clientX: 1, clientY: 1 }));

describe("Panel.Selector", () => {
  it("should select a newly created panel", async () => {
    const { wrapper, store } = await createPanelWrapper({ client });
    await act(async () => {
      render(<Selector />, { wrapper });
    });
    // The strip suspends on the project's panel list, so nothing renders until it
    // resolves.
    const create = await screen.findByRole("button");
    expect(Session.Panel.selectSelected(store.getState())).toBeUndefined();
    await act(async () => {
      fireEvent.click(create);
    });
    const selected = Session.Panel.selectSelected(store.getState());
    expect(selected).toBeDefined();
    await waitFor(() => expect(screen.getByText("New Panel")).toBeTruthy());
  });

  // The strip is the only destination a tab can reach without opening a second
  // window, so a pill has to accept a tab and the create button has to mint one.
  describe("tab drop", () => {
    interface Harness {
      store: TestStore;
      source: panel.Panel;
      tab: panel.Tab;
      others: panel.Panel[];
    }

    const createHarness = async (otherCount: number): Promise<Harness> => {
      const { key: projectKey } = await client.projects.create({
        name: uniqueName("project"),
        layout: {},
      });
      const tab = createTab();
      const source = await createProjectPanel(projectKey, tab);
      const others: panel.Panel[] = [];
      for (let i = 0; i < otherCount; i++)
        others.push(await createProjectPanel(projectKey));
      const { wrapper, store } = await createPanelWrapper({
        client,
        project: projectKey,
      });
      store.dispatch(Session.Panel.select({ key: source.key }));
      await act(async () => {
        render(
          <Haul.Provider>
            <TabDragSource source={source.key} tab={tab} />
            <Selector />
          </Haul.Provider>,
          { wrapper },
        );
      });
      await waitFor(() =>
        [source, ...others].forEach(({ name }) =>
          expect(screen.getByText(name)).toBeTruthy(),
        ),
      );
      return { store, source, tab, others };
    };

    const startDrag = async (): Promise<void> => {
      await act(async () => {
        fireEvent.dragStart(screen.getByText(DRAG_HANDLE));
      });
    };

    it("should move a tab dropped on a pill into that panel and follow it", async () => {
      const { store, source, tab, others } = await createHarness(1);
      const [destination] = others;
      await startDrag();
      await act(async () => {
        fireDrop(screen.getByText(destination.name));
      });
      await waitFor(async () => {
        const [srcDoc, dstDoc] = await Promise.all([
          client.panels.retrieve(source.key),
          client.panels.retrieve(destination.key),
        ]);
        expect(panel.findTab(dstDoc.root, tab.key)).toEqual(tab);
        expect(panel.findTab(srcDoc.root, tab.key)).toBeUndefined();
      });
      await waitFor(() =>
        expect(Session.Panel.selectSelected(store.getState())).toEqual(destination.key),
      );
    });

    it("should mint a panel for a tab dropped on the create button", async () => {
      const { store, source, tab } = await createHarness(0);
      await startDrag();
      await act(async () => {
        fireDrop(getIconButton(document.body, "add"));
      });
      await waitFor(() => {
        const selected = Session.Panel.selectSelected(store.getState());
        if (selected == null || selected === source.key)
          throw new Error("no minted panel selected");
      });
      const minted = Session.Panel.selectSelected(store.getState());
      await waitFor(async () => {
        const [srcDoc, mintedDoc] = await Promise.all([
          client.panels.retrieve(source.key),
          client.panels.retrieve(minted as panel.Key),
        ]);
        expect(panel.findTab(mintedDoc.root, tab.key)).toEqual(tab);
        expect(panel.findTab(srcDoc.root, tab.key)).toBeUndefined();
      });
    });

    it("should switch to a panel a dragged tab rests on", async () => {
      const { store, others } = await createHarness(1);
      const [destination] = others;
      await startDrag();
      fireDragOver(screen.getByText(destination.name));
      await waitFor(() =>
        expect(Session.Panel.selectSelected(store.getState())).toEqual(destination.key),
      );
    });

    // The primary flow: rest on a pill to reach a panel, then drop into its mosaic.
    // The mosaic drop selects a tab, never a panel, so only the absence of a snap
    // back keeps the strip on the destination.
    it("should keep the panel it switched to when the drag lands in it", async () => {
      const { key: projectKey } = await client.projects.create({
        name: uniqueName("project"),
        layout: {},
      });
      const tab = createTab();
      const source = await createProjectPanel(projectKey, tab);
      const destination = await createProjectPanel(projectKey);
      const { wrapper, store } = await createPanelWrapper({
        client,
        project: projectKey,
      });
      store.dispatch(Session.Panel.select({ key: source.key }));
      await act(async () => {
        render(
          <Haul.Provider>
            <TabDragSource source={source.key} tab={tab} />
            <Selector />
            <PPanel.Suspended panelKey={destination.key}>
              <PPanel.Mosaic>{() => <div />}</PPanel.Mosaic>
            </PPanel.Suspended>
          </Haul.Provider>,
          { wrapper },
        );
      });
      await waitFor(() => expect(screen.getByText(destination.name)).toBeTruthy());
      const leaf = await waitFor(() => {
        const found = document.querySelector(".pluto-mosaic__leaf");
        if (found == null) throw new Error("destination mosaic never rendered");
        return found;
      });

      await startDrag();
      fireDragOver(screen.getByText(destination.name));
      await waitFor(() =>
        expect(Session.Panel.selectSelected(store.getState())).toEqual(destination.key),
      );
      await act(async () => {
        fireDrop(leaf);
        fireEvent.dragEnd(screen.getByText(DRAG_HANDLE));
      });
      await waitFor(async () => {
        const dstDoc = await client.panels.retrieve(destination.key);
        expect(panel.findTab(dstDoc.root, tab.key)).toEqual(tab);
      });
      expect(Session.Panel.selectSelected(store.getState())).toEqual(destination.key);
    });

    it("should snap back to the source panel when the drag is cancelled", async () => {
      const { store, source, others } = await createHarness(1);
      const [destination] = others;
      await startDrag();
      fireDragOver(screen.getByText(destination.name));
      await waitFor(() =>
        expect(Session.Panel.selectSelected(store.getState())).toEqual(destination.key),
      );
      await act(async () => {
        fireEvent.dragEnd(screen.getByText(DRAG_HANDLE));
      });
      await waitFor(() =>
        expect(Session.Panel.selectSelected(store.getState())).toEqual(source.key),
      );
    });
  });

  // Deleting the selected panel hands the window to the panel beside it in the
  // strip, the way closing a browser tab does.
  describe("delete", () => {
    const createStrip = async (): Promise<{ store: TestStore; row: panel.Panel[] }> => {
      const { key: projectKey } = await client.projects.create({
        name: uniqueName("project"),
        layout: {},
      });
      const panels = [
        await createProjectPanel(projectKey),
        await createProjectPanel(projectKey),
        await createProjectPanel(projectKey),
      ];
      return await renderStrip(panels, projectKey);
    };

    const select = async (store: TestStore, pan: panel.Panel): Promise<void> => {
      await act(async () => {
        fireEvent.click(screen.getByText(pan.name));
      });
      expect(Session.Panel.selectSelected(store.getState())).toEqual(pan.key);
    };

    // Each test visits the far end of the strip first, so the neighbor and the
    // most recently used panel are different panels.
    it("should select the panel to the right of the deleted one", async () => {
      const { store, row } = await createStrip();
      await select(store, row[2]);
      await select(store, row[0]);
      await deletePanel(row[0]);
      await waitFor(() =>
        expect(Session.Panel.selectSelected(store.getState())).toEqual(row[1].key),
      );
    });

    it("should select the panel to the left when the deleted one was last", async () => {
      const { store, row } = await createStrip();
      await select(store, row[0]);
      await select(store, row[2]);
      await deletePanel(row[2]);
      await waitFor(() =>
        expect(Session.Panel.selectSelected(store.getState())).toEqual(row[1].key),
      );
    });
  });
});
