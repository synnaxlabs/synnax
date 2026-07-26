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
import { Drift } from "@synnaxlabs/drift";
import { uuid } from "@synnaxlabs/x";
import { act, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Session } from "@/session";
import {
  createTestStore,
  renderHookWithConsole,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

beforeAll(async () => {
  // Epoch events only fire on a live change stream; open it up front so the
  // mount-time reconcile runs deterministically.
  await client.cache.ensureStreaming();
});

const leaf = (...tabKeys: string[]): panel.Node => ({
  variant: "leaf",
  tabs: tabKeys.map((key) => ({ variant: "view", key, type: "t", args: {} })),
});

const createPanel = async (key: panel.Key, root: panel.Node): Promise<panel.Panel> =>
  await client.panels.create(
    panel.panelZ.parse({ key, name: uniqueName("panel"), root }),
  );

const selectTab = (store: TestStore, key: panel.Key, tabKey: panel.TabKey): void =>
  void store.dispatch(
    Session.Panel.internalSelectTab({ key, tabKey, otherTabKeys: [tabKey] }),
  );

const mount = async () =>
  await renderHookWithConsole(
    () =>
      Session.Synchronizer.use({
        useReconcileTabSelections:
          Session.Panel.WINDOW_SYNCHRONIZERS.useReconcileTabSelections,
      }),
    { client },
  );

describe("Panel.WINDOW_SYNCHRONIZERS", () => {
  it("reconciles a stale selection when the panel document loads", async () => {
    const panelKey = uuid.create();
    const [tab, ghost] = [uuid.create(), uuid.create()];
    const { store } = await mount();
    act(() => selectTab(store, panelKey, ghost));
    await act(async () => {
      await createPanel(panelKey, leaf(tab));
    });
    await waitFor(() => {
      expect(Session.Panel.selectSelectedTabs(store.getState(), panelKey)).toEqual([
        tab,
      ]);
    });
  });

  it("converges the selection when the selected tab is removed from the tree", async () => {
    const panelKey = uuid.create();
    const [tabA, tabB] = [uuid.create(), uuid.create()];
    const { store } = await mount();
    act(() => selectTab(store, panelKey, tabA));
    await act(async () => {
      await createPanel(panelKey, leaf(tabA, tabB));
    });
    await waitFor(() => {
      expect(Session.Panel.selectSelectedTabs(store.getState(), panelKey)).toEqual([
        tabA,
      ]);
    });
    await act(async () => {
      await client.panels.dispatch(panelKey, panel.removeTab({ key: tabA }));
    });
    await waitFor(() => {
      expect(Session.Panel.selectSelectedTabs(store.getState(), panelKey)).toEqual([
        tabB,
      ]);
    });
  });

  it("reconciles from the cache when an already-cached panel is selected", async () => {
    const panelKey = uuid.create();
    const tab = uuid.create();
    const { store } = await mount();
    await act(async () => {
      await createPanel(panelKey, leaf(tab));
    });
    expect(Session.Panel.selectSelectedTabs(store.getState(), panelKey)).toEqual([]);
    act(() => {
      store.dispatch(Session.Panel.select({ key: panelKey }));
    });
    await waitFor(() => {
      expect(Session.Panel.selectSelectedTabs(store.getState(), panelKey)).toEqual([
        tab,
      ]);
    });
  });
});

const createProject = async (): Promise<project.Project> =>
  await client.projects.create({ name: uniqueName("project"), layout: {} });

const createProjectPanel = async (projectKey: project.Key): Promise<panel.Panel> =>
  await client.panels.create({
    key: uuid.create(),
    name: uniqueName("panel"),
    parent: project.ontologyID(projectKey),
    root: leaf(uuid.create()),
  });

const selectedPanel = (store: TestStore): panel.Key | undefined =>
  store.getState().panels.windows[Drift.MAIN_WINDOW]?.selected;

describe("useReconcileSelection", () => {
  const mountSelection = async (store: TestStore) =>
    await renderHookWithConsole(
      () =>
        Session.Synchronizer.use({
          useReconcileSelection:
            Session.Panel.WINDOW_SYNCHRONIZERS.useReconcileSelection,
        }),
      { client, store },
    );

  const createStoreWithProject = async (
    projectKey: project.Key,
    selected?: panel.Key,
  ): Promise<TestStore> => {
    const store = await createTestStore();
    store.dispatch(Session.Project.select(projectKey));
    if (selected != null)
      store.dispatch(
        Session.Panel.select({ key: selected, windowKey: Drift.MAIN_WINDOW }),
      );
    return store;
  };

  it("selects the project's first panel when the window has no selection", async () => {
    const proj = await createProject();
    const pan = await createProjectPanel(proj.key);
    const store = await createStoreWithProject(proj.key);
    await mountSelection(store);
    await waitFor(() => expect(selectedPanel(store)).toEqual(pan.key));
  });

  it("repairs a selection that points outside the active project", async () => {
    const proj = await createProject();
    const pan = await createProjectPanel(proj.key);
    const store = await createStoreWithProject(proj.key, uuid.create());
    await mountSelection(store);
    await waitFor(() => expect(selectedPanel(store)).toEqual(pan.key));
  });

  it("clears the selection when the project has no panels", async () => {
    const proj = await createProject();
    const store = await createStoreWithProject(proj.key, uuid.create());
    await mountSelection(store);
    await waitFor(() => expect(selectedPanel(store)).toBeUndefined());
  });

  it("adopts the project's first panel when one is created while none is selected", async () => {
    const proj = await createProject();
    const store = await createStoreWithProject(proj.key);
    await mountSelection(store);
    const pan = await createProjectPanel(proj.key);
    await waitFor(() => expect(selectedPanel(store)).toEqual(pan.key));
  });

  it("repairs the selection when the active project changes", async () => {
    const [projA, projB] = [await createProject(), await createProject()];
    const panA = await createProjectPanel(projA.key);
    const panB = await createProjectPanel(projB.key);
    const store = await createStoreWithProject(projA.key, panA.key);
    await mountSelection(store);
    act(() => {
      store.dispatch(Session.Project.select(projB.key));
    });
    await waitFor(() => expect(selectedPanel(store)).toEqual(panB.key));
  });
});

describe("useSyncWindowTitle", () => {
  const windowTitle = (store: TestStore): string | undefined =>
    Drift.selectWindows(store.getState()).find(({ key }) => key === Drift.MAIN_WINDOW)
      ?.title;

  const mountTitle = async () =>
    await renderHookWithConsole(
      () =>
        Session.Synchronizer.use({
          useSyncWindowTitle: Session.Panel.WINDOW_SYNCHRONIZERS.useSyncWindowTitle,
        }),
      { client },
    );

  it("sets the window title to the selected panel's name", async () => {
    const proj = await createProject();
    const pan = await createProjectPanel(proj.key);
    const { store } = await mountTitle();
    act(() => {
      store.dispatch(
        Session.Panel.select({ key: pan.key, windowKey: Drift.MAIN_WINDOW }),
      );
    });
    await waitFor(() => expect(windowTitle(store)).toEqual(pan.name));
  });

  it("tracks the selected panel's rename", async () => {
    const proj = await createProject();
    const pan = await createProjectPanel(proj.key);
    const { store } = await mountTitle();
    act(() => {
      store.dispatch(
        Session.Panel.select({ key: pan.key, windowKey: Drift.MAIN_WINDOW }),
      );
    });
    await waitFor(() => expect(windowTitle(store)).toEqual(pan.name));
    const next = uniqueName("renamed");
    await act(async () => {
      await client.panels.rename(pan.key, next);
    });
    await waitFor(() => expect(windowTitle(store)).toEqual(next));
  });
});
