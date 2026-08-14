// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, project, query } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { destructor } from "@synnaxlabs/x";
import { useRef } from "react";

import { selectActiveWindow, selectSelected } from "@/session/panel/selectors";
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
type RequiredStore = Synchronizer.Store<RequiredStoreState, RequiredAction>;
type Params = Synchronizer.Params<RequiredStoreState, RequiredAction>;

const selectKeys = (state: StoreState): string[] => {
  const keys = new Set<string>();
  Object.values(state.panels.windows).forEach((win) => {
    if (win.selected != null) keys.add(win.selected);
    Object.keys(win.panels).forEach((key) => keys.add(key));
  });
  return [...keys];
};

export const SYNCHRONIZERS: Synchronizer.Synchronizers<
  RequiredStoreState,
  RequiredAction
> = [
  Synchronizer.createRemover<RequiredStoreState, RequiredAction>({
    name: "remove deleted panels",
    domain: (client) => client.panels,
    selectKeys,
    remove: (keys) => remove({ keys }),
  }),
];

const applySelection = ({ client, store }: Params, candidates: panel.Key[]): void => {
  // A retrieve can race a delete and return an already-deleted panel; the
  // local tombstone is authoritative.
  const keys = candidates.filter(
    (key) => !query.Deleted.matches(client.panels.getCached(key)),
  );
  const win = selectActiveWindow(store.getState());
  if (win == null) return;
  const { selected } = win.state;
  if (selected != null && keys.includes(selected)) return;
  if (keys.length === 0) {
    if (selected != null) store.dispatch(clearSelected({ windowKey: win.key }));
    return;
  }
  store.dispatch(select({ key: keys[0], windowKey: win.key }));
};

const repairSelection = async (params: Params): Promise<void> => {
  const { client, store } = params;
  const projectKey = store.getState().project.selected;
  if (projectKey == null) return;
  const panels = await client.panels.retrieve({
    parent: project.ontologyID(projectKey),
  });
  if (store.getState().project.selected !== projectKey) return;
  applySelection(
    params,
    panels.map(({ key }) => key),
  );
};

// The session's selection outlives the project it was made in, so a panel
// outside the active project must never stay selected.
const selection: Synchronizer.Callbacks<RequiredStoreState, RequiredAction> = {
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
          if (!query.isLive(result)) return repair();
          applySelection(
            params,
            result.map(({ key }) => key),
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
    const unwatchSelected = Synchronizer.watch(store, selectSelected, (selected) => {
      if (selected == null) repair();
    });
    return () => destructor.unwind(removeQuery, unwatchProject, unwatchSelected);
  },
};

// The OS window list identifies a window by a stable identity plus what it
// shows: "Main - Ops", "2 - Ops". With nothing selected the identity stands
// alone as "Synnax" for the main window and "Window N" elsewhere.
const syncTitle = ({ client, store }: Params): void => {
  const state = store.getState();
  const win = Drift.selectWindow(state);
  // Pre-rendered windows run the app too; they are invisible and unnumbered.
  if (win == null || !win.reserved || win.ordinal == null) return;
  const isMain = win.key === Drift.MAIN_WINDOW;
  let name: string | undefined;
  const selected = selectSelected(state);
  if (selected != null) {
    const cached = client.panels.getCached(selected);
    if (query.isLive(cached)) name = cached.name;
  }
  let title: string;
  if (name != null) title = `${isMain ? "Main" : win.ordinal} - ${name}`;
  else title = isMain ? "Synnax" : `Window ${win.ordinal}`;
  store.dispatch(Drift.setWindowTitle({ title }));
};

const windowTitle: Synchronizer.Callbacks<RequiredStoreState, RequiredAction> = {
  reconcile: syncTitle,
  listen: (params) => {
    const { client, store } = params;
    const removeOnSet = client.panels.onSet((pan) => {
      if (selectSelected(store.getState()) === pan.key) syncTitle(params);
    });
    const unwatchSelected = Synchronizer.watch(store, selectSelected, () =>
      syncTitle(params),
    );
    return () => destructor.unwind(removeOnSet, unwatchSelected);
  },
};

// The leaves each panel last reconciled against. The client holds one version of a
// panel document, so the row a closed tab sat in is gone by the time the change
// arrives; keeping it here is what lets the selection move to the tab beside the
// closed one rather than to a fixed position.
type LeafGroups = Map<panel.Key, panel.TabKey[][]>;

const reconcileTabs = (
  store: RequiredStore,
  leafGroups: LeafGroups,
  pan: panel.Panel,
): void => {
  const win = selectActiveWindow(store.getState());
  if (win == null) return;
  if (win.state.selected !== pan.key && win.state.panels[pan.key] == null) return;
  const leaves = panel.leafTabGroups(pan.root);
  store.dispatch(
    reconcileSelection({
      windowKey: win.key,
      key: pan.key,
      leaves,
      previous: leafGroups.get(pan.key),
    }),
  );
  leafGroups.set(pan.key, leaves);
};

// Converges the window's tab selections to each referenced panel's live tree.
// Runs per window off its own cache feed, so the selection and the tree
// update in the same tick; cross-window echoes of the action are no-ops.
const createTabSelections = (
  leafGroups: LeafGroups,
): Synchronizer.Callbacks<RequiredStoreState, RequiredAction> => ({
  reconcile: async ({ client, store }) => {
    const win = selectActiveWindow(store.getState());
    if (win == null) return;
    const keys = new Set<string>(Object.keys(win.state.panels));
    if (win.state.selected != null) keys.add(win.state.selected);
    if (keys.size === 0) return;
    const panels = await client.panels.retrieve({
      keys: [...keys],
      ignoreNotFoundError: true,
    });
    panels.forEach((pan) => reconcileTabs(store, leafGroups, pan));
  },
  listen: ({ client, store }) => {
    const removeOnSet = client.panels.onSet((pan) =>
      reconcileTabs(store, leafGroups, pan),
    );
    // A panel cached before the window first selects it fires no set event, so
    // the selection change itself reconciles against the cached tree.
    const unwatchSelected = Synchronizer.watch(store, selectSelected, (selected) => {
      if (selected == null) return;
      const cached = client.panels.getCached(selected);
      if (!query.isLive(cached)) return;
      reconcileTabs(store, leafGroups, cached);
    });
    return () => destructor.unwind(removeOnSet, unwatchSelected);
  },
});

const useTabSelections = (): Synchronizer.Callbacks<
  RequiredStoreState,
  RequiredAction
> => {
  const leafGroups = useRef<LeafGroups>(new Map());
  return createTabSelections(leafGroups.current);
};

export const WINDOW_SYNCHRONIZERS: Synchronizer.Synchronizers<
  RequiredStoreState,
  RequiredAction
> = [
  { name: "reconcile panel selection", use: () => selection },
  { name: "sync window title", use: () => windowTitle },
  { name: "reconcile tab selections", use: useTabSelections },
];
