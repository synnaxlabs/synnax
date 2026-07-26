// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import { panel, project } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { type destructor } from "@synnaxlabs/x";

import {
  type Action,
  clearSelected,
  reconcileSelection,
  remove,
  select,
  type StoreState,
} from "@/session/panel/slice";
import { type Project } from "@/session/project";
import { Synchronizer } from "@/session/synchronizer";

interface RequiredStoreState extends StoreState, Project.StoreState {}
type RequiredAction = Action | Drift.Action;
type RequiredStore = Store<RequiredStoreState, RequiredAction>;
type Params = Synchronizer.Params<RequiredStoreState, RequiredAction>;

const selectKeys = (state: StoreState): string[] => {
  const keys = new Set<string>();
  Object.values(state.panels.windows).forEach((win) => {
    if (win.selected != null) keys.add(win.selected);
    Object.keys(win.panels).forEach((key) => keys.add(key));
  });
  return [...keys];
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers = {
  usePruneDeletedPanels: Synchronizer.create({
    onDelete: (client, handler) => client.panels.onDelete(handler),
    retrieveExisting: async (client, keys) =>
      (await client.panels.retrieve({ keys, ignoreNotFoundError: true })).map(
        ({ key }) => key,
      ),
    selectKeys,
    remove,
  }),
};

const selectActiveWindowSelected = (state: StoreState): panel.Key | undefined => {
  const windowKey = Drift.selectWindowKey(state);
  if (windowKey == null) return undefined;
  return state.panels.windows[windowKey]?.selected;
};

const applySelection = (store: RequiredStore, keys: panel.Key[]): void => {
  const state = store.getState();
  const windowKey = Drift.selectWindowKey(state);
  if (windowKey == null) return;
  const selected = state.panels.windows[windowKey]?.selected;
  if (selected != null && keys.includes(selected)) return;
  if (keys.length === 0) {
    if (selected != null) store.dispatch(clearSelected({ windowKey }));
    return;
  }
  store.dispatch(select({ key: keys[0], windowKey }));
};

const repairSelection = async ({ client, store }: Params): Promise<void> => {
  const projectKey = store.getState().project.selected;
  if (projectKey == null) return;
  const panels = await client.panels.retrieve({
    parent: project.ontologyID(projectKey),
  });
  if (store.getState().project.selected !== projectKey) return;
  applySelection(
    store,
    panels.map(({ key }) => key),
  );
};

// The session's selection outlives the project it was made in, so a panel
// outside the active project must never stay selected.
const selection: Synchronizer.Synchronizer<RequiredStoreState, RequiredAction> = {
  reconcile: repairSelection,
  listen: (params) => {
    const { client, store } = params;
    const repair = (): void => {
      repairSelection(params).catch(console.error);
    };
    // The client maintains the by-project answer itself, tracking optimistic
    // creates and ontology changes a manual retrieve would race.
    let removeQuery: destructor.Destructor | undefined;
    const subscribeProject = (projectKey: project.Key | undefined): void => {
      removeQuery?.();
      removeQuery = undefined;
      if (projectKey == null) return;
      removeQuery = client.panels.onChange(
        { parent: project.ontologyID(projectKey) },
        (result) => {
          if (result == null || result.variant === "deleted") return repair();
          applySelection(
            store,
            result.data.map(({ key }) => key),
          );
        },
      );
    };
    subscribeProject(store.getState().project.selected);
    const unwatchProject = Synchronizer.watch(
      store,
      (state) => state.project.selected,
      (projectKey) => {
        subscribeProject(projectKey);
        repair();
      },
    );
    const unwatchSelected = Synchronizer.watch(
      store,
      selectActiveWindowSelected,
      (selected) => {
        if (selected == null) repair();
      },
    );
    return () => {
      removeQuery?.();
      unwatchProject();
      unwatchSelected();
    };
  },
};

// The OS window list identifies a window by what it shows.
const syncTitle = ({ client, store }: Params): void => {
  const selected = selectActiveWindowSelected(store.getState());
  if (selected == null) return;
  const cached = client.panels.getCached({ key: selected });
  if (cached == null || cached.variant === "deleted") return;
  store.dispatch(Drift.setWindowTitle({ title: cached.data.name }));
};

const windowTitle: Synchronizer.Synchronizer<RequiredStoreState, RequiredAction> = {
  reconcile: syncTitle,
  listen: (params) => {
    const { client, store } = params;
    const removeOnSet = client.panels.onSet((pan) => {
      if (selectActiveWindowSelected(store.getState()) === pan.key)
        store.dispatch(Drift.setWindowTitle({ title: pan.name }));
    });
    const unwatchSelected = Synchronizer.watch(store, selectActiveWindowSelected, () =>
      syncTitle(params),
    );
    return () => {
      removeOnSet();
      unwatchSelected();
    };
  },
};

const reconcileTabs = (store: RequiredStore, pan: panel.Panel): void => {
  const state = store.getState();
  const windowKey = Drift.selectWindowKey(state);
  if (windowKey == null) return;
  const win = state.panels.windows[windowKey];
  if (win == null) return;
  if (win.selected !== pan.key && win.panels[pan.key] == null) return;
  store.dispatch(
    reconcileSelection({
      windowKey,
      key: pan.key,
      leaves: panel.leafTabGroups(pan.root),
    }),
  );
};

// Converges the window's tab selections to each referenced panel's live tree.
// Runs per window off its own cache feed, so the selection and the tree
// update in the same tick; cross-window echoes of the action are no-ops.
const tabSelections: Synchronizer.Synchronizer<RequiredStoreState, RequiredAction> = {
  reconcile: async ({ client, store }) => {
    const state = store.getState();
    const windowKey = Drift.selectWindowKey(state);
    if (windowKey == null) return;
    const win = state.panels.windows[windowKey];
    if (win == null) return;
    const keys = new Set<string>(Object.keys(win.panels));
    if (win.selected != null) keys.add(win.selected);
    if (keys.size === 0) return;
    const panels = await client.panels.retrieve({
      keys: [...keys],
      ignoreNotFoundError: true,
    });
    panels.forEach((pan) => reconcileTabs(store, pan));
  },
  listen: ({ client, store }) => {
    const removeOnSet = client.panels.onSet((pan) => reconcileTabs(store, pan));
    // A panel cached before the window first selects it fires no set event, so
    // the selection change itself reconciles against the cached tree.
    const unwatchSelected = Synchronizer.watch(
      store,
      selectActiveWindowSelected,
      (selected) => {
        if (selected == null) return;
        const cached = client.panels.getCached({ key: selected });
        if (cached == null || cached.variant === "deleted") return;
        reconcileTabs(store, cached.data);
      },
    );
    return () => {
      removeOnSet();
      unwatchSelected();
    };
  },
};

export const WINDOW_SYNCHRONIZERS: Synchronizer.Synchronizers = {
  useReconcileSelection: () => selection,
  useSyncWindowTitle: () => windowTitle,
  useReconcileTabSelections: () => tabSelections,
};
