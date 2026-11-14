// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, type project, query, type user } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Status } from "@synnaxlabs/lyra/status";
import { destructor } from "@synnaxlabs/x";
import { useState } from "react";

import {
  selectActiveWindow,
  selectOrder,
  selectSelected,
} from "@/session/panel/selectors";
import {
  type Action,
  clearSelected,
  reconcileOrder,
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

// The client learns the user at login, so reading it never touches the network.
const cachedUser = (client: Params["client"]): user.Key | undefined => {
  const cached = client.users.current.watch().getCached();
  return query.isLive(cached) ? cached.key : undefined;
};

const hasTabs = (node: panel.Node): boolean =>
  node.variant === "leaf"
    ? node.tabs.length > 0
    : hasTabs(node.first) || hasTabs(node.last);

const DELETE_FAILED = { name: "Draft", message: "Failed to delete draft" };

// Drafts of the user that lost their last tab. A rejected delete puts the draft back,
// so a failed key is not deleted again until the draft holds a tab once more.
interface Emptied {
  deleting: Set<panel.Key>;
  failed: Set<panel.Key>;
  handleError: Status.ErrorHandler;
}

const deleteEmptied = (params: Params, state: Emptied, key: panel.Key): void => {
  const { client } = params;
  if (state.deleting.has(key)) return;
  state.deleting.add(key);
  // The delete is outside the panel's dispatch queue, so it waits for the close. A
  // close that failed put the tab back, so the draft is checked again.
  client.panels
    .settled(key)
    .then(async () => {
      const cached = client.panels.watch(key).getCached();
      if (!query.isLive(cached) || hasTabs(cached.root)) return;
      await client.panels.delete(key);
    })
    .catch((e: unknown) => {
      state.failed.add(key);
      state.handleError(e, DELETE_FAILED);
      // The rollback ran before the key was marked failed, so it left the draft out.
      return repairMembership(params, state);
    })
    .finally(() => state.deleting.delete(key))
    .catch(console.error);
};

const applyMembership = (
  params: Params,
  state: Emptied,
  candidates: panel.Panel[],
): void => {
  const { client, store } = params;
  // A draft holds work only while it holds a tab, so closing its last one deletes it.
  const user = cachedUser(client);
  const emptied = new Set<panel.Key>();
  candidates.forEach(({ key, owner, root }) => {
    if (owner == null || owner !== user) return;
    if (hasTabs(root)) state.failed.delete(key);
    else if (!state.failed.has(key)) emptied.add(key);
  });
  // A retrieve can race a delete and return an already-deleted panel; the
  // local tombstone is authoritative.
  const live = candidates.filter(
    ({ key }) =>
      !emptied.has(key) && !query.Deleted.matches(client.panels.watch(key).getCached()),
  );
  const keys = live.map(({ key }) => key);
  const win = selectActiveWindow(store.getState());
  // Leaving an emptied draft returns to the panel the window showed before it. The
  // optimistic delete hands the window to a strip neighbor, so this runs first.
  if (win?.state.selected != null && emptied.has(win.state.selected)) {
    const previous = win.state.mounted.find((key) => keys.includes(key));
    if (previous != null) store.dispatch(select({ key: previous, windowKey: win.key }));
  }
  emptied.forEach((key) => deleteEmptied(params, state, key));
  store.dispatch(
    reconcileOrder({ panels: live.map(({ key, name }) => ({ key, name })) }),
  );
  if (win == null) return;
  const selected = selectActiveWindow(store.getState())?.state.selected;
  if (selected != null && keys.includes(selected)) return;
  if (keys.length === 0) {
    if (selected != null) store.dispatch(clearSelected({ windowKey: win.key }));
    return;
  }
  // The reconcile above put every live key in the order, so the first live
  // entry is the strip's leftmost pill.
  const first = selectOrder(store.getState()).find((key) => keys.includes(key));
  store.dispatch(select({ key: first ?? keys[0], windowKey: win.key }));
};

const repairMembership = async (params: Params, state: Emptied): Promise<void> => {
  const { client, store } = params;
  const projectKey = store.getState().project.selected;
  if (projectKey == null) return;
  const user = cachedUser(client);
  if (user == null) return;
  const { records } = await client.panels
    .watch({ where: panel.visibleTo(projectKey, user) })
    .retrieve();
  if (store.getState().project.selected !== projectKey) return;
  applyMembership(params, state, records);
};

// The session's strip order and selection both follow the project's live membership:
// the order converges first, then the selection, so a repaired selection always lands
// on the strip's leftmost pill. The selection outlives the project it was made in, so a
// panel outside the active project must never stay selected.
const createMembership = (
  state: Emptied,
): Synchronizer.Callbacks<RequiredStoreState, RequiredAction> => ({
  reconcile: async (params) => await repairMembership(params, state),
  listen: (params) => {
    const { client, store } = params;
    const repair = (): void => {
      repairMembership(params, state).catch(console.error);
    };
    // The client maintains the by-project answer itself, tracking optimistic creates
    // and project changes a manual retrieve would race.
    let removeQuery: destructor.Destructor | undefined;
    const subscribeProject = (projectKey: project.Key | undefined): void => {
      removeQuery?.();
      removeQuery = undefined;
      const user = cachedUser(client);
      if (projectKey == null || user == null) return;
      removeQuery = client.panels
        .watch({ where: panel.visibleTo(projectKey, user) })
        .onChange((window) => {
          if (window == null) return repair();
          applyMembership(params, state, window.records);
        });
    };
    subscribeProject(store.getState().project.selected);
    const unwatchUser = client.users.current.watch().onChange(() => {
      subscribeProject(store.getState().project.selected);
      repair();
    });
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
    return () =>
      destructor.unwind(removeQuery, unwatchUser, unwatchProject, unwatchSelected);
  },
});

const useMembership = (): Synchronizer.Callbacks<
  RequiredStoreState,
  RequiredAction
> => {
  const handleError = Status.useErrorHandler();
  const [callbacks] = useState(() =>
    createMembership({ deleting: new Set(), failed: new Set(), handleError }),
  );
  return callbacks;
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
    const cached = client.panels.watch(selected).getCached();
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
    const panels = await client.panels.retrieve({ key: [...keys] });
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
      const cached = client.panels.watch(selected).getCached();
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
  const [callbacks] = useState(() => createTabSelections(new Map()));
  return callbacks;
};

export const WINDOW_SYNCHRONIZERS: Synchronizer.Synchronizers<
  RequiredStoreState,
  RequiredAction
> = [
  { name: "reconcile panel order and selection", use: useMembership },
  { name: "sync window title", use: () => windowTitle },
  { name: "reconcile tab selections", use: useTabSelections },
];
