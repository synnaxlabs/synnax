// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, panel, project, user } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { Haul, Mosaic, Panel as PPanel } from "@synnaxlabs/pluto";
import { fireDragEvent } from "@synnaxlabs/pluto/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

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

interface CreateProjectPanelProps {
  name?: string;
  key?: panel.Key;
  tab?: panel.Tab;
}

const createProjectPanel = async (
  projectKey: project.Key,
  {
    name = "panel",
    key = uuid.create(),
    tab = createTab(),
  }: CreateProjectPanelProps = {},
): Promise<panel.Panel> =>
  await client.panels.create({
    key,
    name: uniqueName(name),
    parent: project.ontologyID(projectKey),
    root: { variant: "leaf", tabs: [tab] },
  });

/**
 * Nests the wrapper in the haul provider the app mounts through Pluto.Context, so a
 * pill drag resolves against the same redux-backed drag state as production.
 */
const withHaul = (Wrapper: FC<PropsWithChildren>): FC<PropsWithChildren> => {
  const HaulWrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Wrapper>
      <Haul.Provider {...Session.Haul.PROVIDER_PROPS}>{children}</Haul.Provider>
    </Wrapper>
  );
  HaulWrapper.displayName = "HaulWrapper";
  return HaulWrapper;
};

const pillNames = (): (string | null)[] =>
  screen.getAllByRole("tab").map((tab) => tab.textContent);

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

const CURSOR = { x: 1, y: 1 };

const fireDrop = (el: Element): void => fireDragEvent(el, "drop", CURSOR);

const fireDragOver = (el: Element): void => fireDragEvent(el, "dragOver", CURSOR);

describe("Panel.Selector", () => {
  beforeEach(() => {
    mocks.engine = "web";
  });

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
    await waitFor(() => expect(screen.getByText("New panel")).toBeTruthy());
  });

  // The membership query answers in an order the user never chose; the session's
  // strip order is what the pills must follow.
  it("should render the pills in the session's strip order", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const alpha = await createProjectPanel(proj.key, { name: "alpha" });
    const bravo = await createProjectPanel(proj.key, { name: "bravo" });
    const { wrapper, store } = await createPanelWrapper({
      client,
      project: proj.key,
    });
    act(() => {
      store.dispatch(
        Session.Panel.reconcileOrder({
          panels: [
            { key: alpha.key, name: alpha.name },
            { key: bravo.key, name: bravo.name },
          ],
        }),
      );
      store.dispatch(Session.Panel.reorder({ key: bravo.key, index: 0 }));
    });
    await act(async () => {
      render(<Selector />, { wrapper });
    });
    await screen.findByText(alpha.name);
    await screen.findByText(bravo.name);
    expect(pillNames()).toEqual([bravo.name, alpha.name]);
  });

  // A panel the session has not reconciled yet must not displace the panels the
  // user ordered, however early it lands in the membership answer.
  it("should render a panel missing from the strip order last", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    // The membership answer arrives in key order, so the unreconciled panel takes
    // the lower key: a correct sort cannot pass by luck.
    const [lowKey, highKey] = [uuid.create(), uuid.create()].sort();
    const pending = await createProjectPanel(proj.key, {
      name: "pending",
      key: lowKey,
    });
    const known = await createProjectPanel(proj.key, { name: "known", key: highKey });
    const { wrapper, store } = await createPanelWrapper({
      client,
      project: proj.key,
    });
    act(() => {
      store.dispatch(
        Session.Panel.reconcileOrder({
          panels: [{ key: known.key, name: known.name }],
        }),
      );
    });
    await act(async () => {
      render(<Selector />, { wrapper });
    });
    await screen.findByText(pending.name);
    await screen.findByText(known.name);
    expect(pillNames()).toEqual([known.name, pending.name]);
  });

  // Pluto resolves the insertion slot and the slice applies the move; nothing else
  // proves the strip hands one to the other, so a mis-wired drop is invisible.
  it("should reorder the strip when a pill is dragged past the last slot", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const alpha = await createProjectPanel(proj.key, { name: "alpha" });
    const bravo = await createProjectPanel(proj.key, { name: "bravo" });
    const { wrapper, store } = await createPanelWrapper({
      client,
      project: proj.key,
    });
    act(() => {
      store.dispatch(
        Session.Panel.reconcileOrder({
          panels: [
            { key: alpha.key, name: alpha.name },
            { key: bravo.key, name: bravo.name },
          ],
        }),
      );
    });
    await act(async () => {
      render(<Selector />, { wrapper: withHaul(wrapper) });
    });
    await screen.findByText(alpha.name);
    await screen.findByText(bravo.name);
    expect(pillNames()).toEqual([alpha.name, bravo.name]);
    const [first] = screen.getAllByRole("tab");
    // Every element shares one faked 100-wide rect, so a cursor past the common
    // center of the pills resolves to the slot after the last one.
    act(() => {
      fireEvent.dragStart(first);
      fireDragEvent(screen.getByRole("tablist"), "drop", { x: 200, y: 16 });
    });
    await waitFor(() => expect(pillNames()).toEqual([bravo.name, alpha.name]));
    expect(Session.Panel.selectOrder(store.getState())).toEqual([bravo.key, alpha.key]);
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
      const source = await createProjectPanel(projectKey, { tab });
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
      const source = await createProjectPanel(projectKey, { tab });
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

  describe("without create permission", () => {
    it("should offer no create button", async () => {
      const proj = await client.projects.create({
        name: uniqueName("project"),
        layout: {},
      });
      const pan = await createProjectPanel(proj.key, { name: "alpha" });
      const viewer = await createTestClientWithPolicy(client, {
        name: uuid.create(),
        objects: [
          panel.TYPE_ONTOLOGY_ID,
          project.TYPE_ONTOLOGY_ID,
          user.TYPE_ONTOLOGY_ID,
          access.role.TYPE_ONTOLOGY_ID,
          access.policy.TYPE_ONTOLOGY_ID,
        ],
        actions: ["retrieve"],
      });
      const { wrapper } = await createPanelWrapper({
        client: viewer,
        project: proj.key,
      });
      await act(async () => {
        render(<Selector />, { wrapper });
      });
      await waitFor(() => expect(screen.getByText(pan.name)).toBeTruthy());
      expect(screen.queryByRole("button")).toBeNull();
    });
  });

  describe("open in new window", () => {
    // Waits on Delete so the menu is proven open before the window item is missed.
    const openPillMenu = async (): Promise<void> => {
      const { key: projectKey } = await client.projects.create({
        name: uniqueName("project"),
        layout: {},
      });
      const pan = await createProjectPanel(projectKey);
      await renderStrip([pan], projectKey);
      fireEvent.contextMenu(screen.getByText(pan.name));
      await screen.findByText("Delete");
    };

    it("should offer the panel in a second window in the tauri engine", async () => {
      mocks.engine = "tauri";
      await openPillMenu();
      expect(screen.getByText("Open in new window")).toBeTruthy();
    });

    it("should hide the item in the browser, which cannot open a window", async () => {
      await openPillMenu();
      expect(screen.queryByText("Open in new window")).toBeNull();
    });

    const openPillMenuBeside = async (onSelected: boolean): Promise<void> => {
      const { key: projectKey } = await client.projects.create({
        name: uniqueName("project"),
        layout: {},
      });
      const panels = [
        await createProjectPanel(projectKey, { name: "alpha" }),
        await createProjectPanel(projectKey, { name: "bravo" }),
      ];
      const { row } = await renderStrip(panels, projectKey);
      await act(async () => {
        fireEvent.click(screen.getByText(row[0].name));
      });
      fireEvent.contextMenu(screen.getByText(row[onSelected ? 0 : 1].name));
      await screen.findByText("Open in new window");
    };

    it("should show the shortcut on the selected pill", async () => {
      mocks.engine = "tauri";
      await openPillMenuBeside(true);
      expect(screen.queryByLabelText("trigger-indicator")).not.toBeNull();
    });

    it("should hide the shortcut on a pill that is not selected", async () => {
      mocks.engine = "tauri";
      await openPillMenuBeside(false);
      expect(screen.queryByLabelText("trigger-indicator")).toBeNull();
    });
  });
});
