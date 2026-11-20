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
  panel,
  query,
  UnexpectedError,
  type user,
} from "@synnaxlabs/client";
import { Access, type Flux, Panel, Synnax } from "@synnaxlabs/pluto";
import type { location } from "@synnaxlabs/x";

import { useGetIconKey } from "@/platform/panel/tab";
import { Session } from "@/session";

/** Where in a panel's mosaic a tab lands. */
export interface Placement {
  /** Key of the leaf node the tab lands in. */
  leaf: number;
  /** An edge splits the leaf and takes the new half; "center" inserts in place. */
  location: location.Location;
}

export interface OpenTabOptions {
  /**
   * Keeps at most one view tab of the type in the panel. The Core enforces it, so a
   * second open from any window focuses the tab the panel has.
   */
  singleton?: boolean;
  /**
   * Focuses the first tab of the panel that matches, if any, instead of opening one.
   * Unlike singleton, it holds for this open only, and the panel may hold more matching
   * tabs.
   */
  focus?: (tab: panel.Tab) => boolean;
  /**
   * Places the tab in a specific leaf instead of beside the current one. Ignored when
   * the leaf is gone, which an await between the drop and the open allows.
   */
  placement?: Placement;
  /**
   * Lands the tabs in a draft of the signed-in user: a draft that already holds the
   * last tab, else the selected panel when it is such a draft, else a new draft with
   * this name.
   */
  draft?: Pick<panel.New, "name">;
}

export type OpenTab = (params: panel.NewTab, options?: OpenTabOptions) => void;

export type OpenTabs = (params: panel.NewTab[], options?: OpenTabOptions) => void;

// focusedTabKey reads back which tab a request landed on, after the dispatch has
// applied. An insert the reducer skipped as a duplicate resolves to the tab already
// holding that content, so the caller focuses it instead of opening a second one.
const focusedTabKey = (
  root: panel.Node,
  tab: panel.Tab,
  singleton?: boolean,
): panel.TabKey | undefined => {
  if (tab.variant === "resource")
    return panel.findTabByResource(root, tab.resource)?.key;
  if (singleton) return panel.findTabByType(root, tab.type)?.key;
  return panel.findTab(root, tab.key)?.key;
};

/**
 * Returns a callback that opens tabs in the scoped panel, the session-selected panel,
 * or a panel it creates when neither exists. A created panel takes the icon registered
 * for the type of the first tab. Every tab lands in one dispatch, so a batch is a
 * single request and a single undo entry. A placement splits once: the first tab takes
 * the new half and the rest join it.
 */
export const useOpenTabs = (): OpenTabs => {
  const dispatchSession = Session.useDispatch();
  const { dispatch } = Panel.useDispatch();
  const selectTab = Session.Panel.useSelectTab();
  const parentPanelKey = Panel.useOptionalKey();
  const getSelected = Session.Panel.useGetSelected();
  const getSelectedProject = Session.Project.useGetSelected();
  const parentTabKey = Panel.useOptionalTabKey();
  const client = Synnax.use();
  const getIconKey = useGetIconKey();
  // insertIntoExisting adds the tabs to a panel that is already on the Core (the scoped
  // parent or the selected panel), so a remote dispatch is correct.
  const insertIntoExisting = (
    panelKey: panel.Key,
    params: panel.NewTab[],
    options?: OpenTabOptions,
  ) => {
    const current = client?.panels.watch(panelKey).getCached();
    if (!query.isLive(current))
      throw new NotFoundError(`Panel with key ${panelKey} not found`);
    const { singleton, placement, focus } = options ?? {};
    const existing = focus == null ? undefined : panel.tabs(current.root).find(focus);
    if (existing != null) return selectTab(existing.key, panelKey);
    const tabs = params.map((p): panel.Tab => panel.tabZ.parse({ ...p }));
    const last = tabs.at(-1);
    if (last == null) return;
    // A keyless tab opens beside the current one. A keyed one is a reopen, so it
    // carries no placement and keeps wherever it already sits.
    const besideCurrent =
      params[0]?.key == null && panelKey === parentPanelKey ? parentTabKey : undefined;
    dispatch({
      key: panelKey,
      actions: panel.insertTabs({
        tabs,
        singleton,
        ...(placement != null
          ? { targetLeaf: placement.leaf, location: placement.location }
          : { targetTab: besideCurrent }),
      }),
    });
    const applied = client?.panels.watch(panelKey).getCached();
    if (!query.isLive(applied)) return;
    const focused = focusedTabKey(applied.root, last, singleton);
    if (focused != null) selectTab(focused, panelKey);
  };
  const { update: createPanel } = Panel.useCreate({
    afterOptimistic: ({
      data: { key, root },
      rollbacks,
    }: Flux.AfterOptimisticParams<panel.Panel>) => {
      dispatchSession(Session.Panel.select({ key }));
      rollbacks.push(() => dispatchSession(Session.Panel.clearSelected({})));
      // This hook only creates panels with a single leaf root (below), so focus goes
      // to the last tab seeded into it. Focus is dispatched directly: the panel is
      // not yet retrievable, so useSelectTab's cached leaf lookup would fail.
      const last = root.variant === "leaf" ? root.tabs.at(-1) : undefined;
      if (root.variant === "leaf" && last != null)
        dispatchSession(
          Session.Panel.internalSelectTab({
            key,
            tabKey: last.key,
            otherTabKeys: root.tabs.map((t) => t.key),
          }),
        );
    },
  });
  const getUser = (): user.Key => {
    const cached = client?.users.current.watch().getCached();
    if (!query.isLive(cached)) throw new UnexpectedError("the user is not loaded");
    return cached.key;
  };
  const findDraft = (
    user: user.Key,
    params: panel.NewTab[],
    singleton?: boolean,
  ): panel.Key | undefined => {
    const project = getSelectedProject();
    const last = params.at(-1);
    if (client == null || project == null || last == null) return undefined;
    const tab = panel.tabZ.parse({ ...last });
    // The strip keeps this list live in every window.
    const visible = client.panels
      .watch({ where: panel.visibleTo(project, user) })
      .getCached();
    const holder = visible?.records.find(
      ({ owner, root }) =>
        owner === user && focusedTabKey(root, tab, singleton) != null,
    );
    if (holder != null) return holder.key;
    const selected = getSelected();
    if (selected == null) return undefined;
    const cached = client.panels.watch(selected).getCached();
    return query.isLive(cached) && cached.owner === user ? selected : undefined;
  };
  return (params: panel.NewTab[], options?: OpenTabOptions) => {
    if (params.length === 0) return;
    const draft = options?.draft;
    const owner = draft != null ? getUser() : undefined;
    const panelKey =
      owner != null
        ? findDraft(owner, params, options?.singleton)
        : (parentPanelKey ?? getSelected());
    if (panelKey != null) {
      if (owner != null) dispatchSession(Session.Panel.select({ key: panelKey }));
      return insertIntoExisting(panelKey, params, options);
    }
    // No panel to insert into: seed the tabs into the new panel's initial root so the
    // create persists them in a single request. This avoids a second remote dispatch
    // that would race the create and fail with "panel not found", while the local
    // store update keeps focus optimistic.
    const tabs = params.map((p) => panel.tabZ.parse({ ...p }));
    createPanel({
      name: draft?.name ?? "New panel",
      icon: getIconKey(tabs[0]),
      root: { variant: "leaf", tabs },
      project: getSelectedProject(),
      owner,
    });
  };
};

/** Opens a single tab. See {@link useOpenTabs} for placement and focus. */
export const useOpenTab = (): OpenTab => {
  const openTabs = useOpenTabs();
  return (params: panel.NewTab, options?: OpenTabOptions) =>
    openTabs([params], options);
};

/**
 * Returns the key of the panel that a gesture in this window acts on: the scoped one,
 * else the selected one. Null when there is neither.
 */
export const useActiveKey = (): panel.Key | null => {
  const scoped = Panel.useOptionalKey();
  const selected = Session.Panel.useSelectSelected();
  return scoped ?? selected ?? null;
};

/**
 * Reports whether this window can open a tab. A tab landing in a panel that exists is
 * an update on it, which the owner of a draft needs no access for; with no panel to
 * land in, {@link useOpenTabs} mints one instead.
 */
export const useCanOpenTab = (): boolean => {
  const key = useActiveKey();
  const owned = Panel.useOwned({ key });
  const granted = Access.useGranted(
    key != null
      ? { objects: panel.resourceID(key), action: "update" }
      : { objects: panel.TYPE_RESOURCE_ID, action: "create" },
  );
  return owned || granted;
};

/** Reports whether this window can restructure the panel it is acting on. */
export const useCanEditActive = (): boolean => {
  const key = useActiveKey();
  const owned = Panel.useOwned({ key });
  const granted = Access.useGranted({
    objects: key != null ? panel.resourceID(key) : panel.TYPE_RESOURCE_ID,
    action: "update",
  });
  return owned || granted;
};
