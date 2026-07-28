// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, project, query } from "@synnaxlabs/client";
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

  // Regression: the delete flow prunes session state before the cluster delete,
  // so the empty selection triggers a repair whose retrieve races the delete and
  // resolves post-tombstone still holding the corpse; it must never be selected.
  it("never selects the corpse when deleting the last panel", async () => {
    const proj = await createProject();
    const pan = await createProjectPanel(proj.key);
    const store = await createStoreWithProject(proj.key);
    await mountSelection(store);
    await waitFor(() => expect(selectedPanel(store)).toEqual(pan.key));
    const violations: panel.Key[] = [];
    const unsubscribe = store.subscribe(() => {
      const selected = selectedPanel(store);
      if (selected == null) return;
      if (query.Deleted.matches(client.panels.getCached({ key: selected })))
        violations.push(selected);
    });
    await act(async () => {
      store.dispatch(Session.Panel.remove(pan.key));
      await client.panels.delete(pan.key);
      // The racing repair retrieve resolves after the delete; give every
      // in-flight repair a full round-trip to land before judging.
      await client.panels.retrieve({ parent: project.ontologyID(proj.key) });
      await client.panels.retrieve({ parent: project.ontologyID(proj.key) });
    });
    await waitFor(() => {
      expect(violations).toEqual([]);
      expect(selectedPanel(store)).toBeUndefined();
    });
    unsubscribe();
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
  const windowTitle = (
    store: TestStore,
    key: string = Drift.MAIN_WINDOW,
  ): string | undefined => Drift.selectWindow(store.getState(), key)?.title;

  const mountTitle = async (store?: TestStore) =>
    await renderHookWithConsole(
      () =>
        Session.Synchronizer.use({
          useSyncWindowTitle: Session.Panel.WINDOW_SYNCHRONIZERS.useSyncWindowTitle,
        }),
      { client, store },
    );

  it("titles the main window with its identity and the selected panel's name", async () => {
    const proj = await createProject();
    const pan = await createProjectPanel(proj.key);
    const { store } = await mountTitle();
    act(() => {
      store.dispatch(
        Session.Panel.select({ key: pan.key, windowKey: Drift.MAIN_WINDOW }),
      );
    });
    await waitFor(() => expect(windowTitle(store)).toEqual(`Main - ${pan.name}`));
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
    await waitFor(() => expect(windowTitle(store)).toEqual(`Main - ${pan.name}`));
    const next = uniqueName("renamed");
    await act(async () => {
      await client.panels.rename(pan.key, next);
    });
    await waitFor(() => expect(windowTitle(store)).toEqual(`Main - ${next}`));
  });

  it("falls back to Synnax when the main window has no selection", async () => {
    const { store } = await mountTitle();
    await waitFor(() => expect(windowTitle(store)).toEqual("Synnax"));
  });

  it("restores the fallback when the selection clears", async () => {
    const proj = await createProject();
    const pan = await createProjectPanel(proj.key);
    const { store } = await mountTitle();
    act(() => {
      store.dispatch(
        Session.Panel.select({ key: pan.key, windowKey: Drift.MAIN_WINDOW }),
      );
    });
    await waitFor(() => expect(windowTitle(store)).toEqual(`Main - ${pan.name}`));
    act(() => {
      store.dispatch(Session.Panel.clearSelected({ windowKey: Drift.MAIN_WINDOW }));
    });
    await waitFor(() => expect(windowTitle(store)).toEqual("Synnax"));
  });

  const AUX_LABEL = "aux";
  const AUX_KEY = "aux-key";

  const auxWindow: Drift.WindowState = {
    key: AUX_KEY,
    stage: "created",
    processCount: 0,
    reserved: true,
    focusCount: 0,
    centerCount: 0,
    ordinal: 2,
  };

  const createAuxStore = async (): Promise<TestStore> =>
    await createTestStore({
      windowLabel: AUX_LABEL,
      preloadedState: {
        drift: {
          ...Session.ZERO_STATE.drift,
          windows: { ...Session.ZERO_STATE.drift.windows, [AUX_LABEL]: auxWindow },
          labelKeys: { ...Session.ZERO_STATE.drift.labelKeys, [AUX_LABEL]: AUX_KEY },
          keyLabels: { ...Session.ZERO_STATE.drift.keyLabels, [AUX_KEY]: AUX_LABEL },
          nextOrdinal: 3,
        },
      },
    });

  it("titles a secondary window with its ordinal", async () => {
    const proj = await createProject();
    const pan = await createProjectPanel(proj.key);
    const store = await createAuxStore();
    await mountTitle(store);
    act(() => {
      store.dispatch(Session.Panel.select({ key: pan.key, windowKey: AUX_KEY }));
    });
    await waitFor(() => expect(windowTitle(store, AUX_KEY)).toEqual(`2 - ${pan.name}`));
    act(() => {
      store.dispatch(Session.Panel.clearSelected({ windowKey: AUX_KEY }));
    });
    await waitFor(() => expect(windowTitle(store, AUX_KEY)).toEqual("Window 2"));
  });
});
