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
  type ontology,
  panel,
  project,
  query,
} from "@synnaxlabs/client";
import { Access, type Flux, Panel, Synnax } from "@synnaxlabs/pluto";
import { type location } from "@synnaxlabs/x";
import { useCallback } from "react";

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
   * When set and the tab is a view, an existing view of the same type is focused
   * instead of opening a duplicate, making the view a per-panel singleton.
   */
  singleton?: boolean;
  /**
   * Places the tab in a specific leaf instead of beside the current one. Ignored when
   * the leaf is gone, which an await between the drop and the open allows.
   */
  placement?: Placement;
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
 * or a panel it creates when neither exists. Every tab lands in one dispatch, so a
 * batch is a single request and a single undo entry. A placement splits once: the first
 * tab takes the new half and the rest join it.
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
  // insertIntoExisting adds the tabs to a panel that is already on the cluster (the
  // scoped parent or the selected panel), so a remote dispatch is correct.
  const insertIntoExisting = useCallback(
    (panelKey: panel.Key, params: panel.NewTab[], options?: OpenTabOptions) => {
      if (!query.isLive(client?.panels.getCached(panelKey)))
        throw new NotFoundError(`Panel with key ${panelKey} not found`);
      const { singleton, placement } = options ?? {};
      const tabs = params.map((p): panel.Tab => panel.tabZ.parse({ ...p }));
      const last = tabs.at(-1);
      if (last == null) return;
      // A keyless tab opens beside the current one. A keyed one is a reopen, so it
      // carries no placement and keeps wherever it already sits.
      const besideCurrent = params[0]?.key == null ? parentTabKey : undefined;
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
      const applied = client?.panels.getCached(panelKey);
      if (!query.isLive(applied)) return;
      const focus = focusedTabKey(applied.root, last, singleton);
      if (focus != null) selectTab(focus, panelKey);
    },
    [parentTabKey, client, dispatch, selectTab],
  );
  const { update: createPanel } = Panel.useCreate({
    afterOptimistic: useCallback(
      ({ data: { key, root }, rollbacks }: Flux.AfterOptimisticParams<panel.Panel>) => {
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
      [dispatchSession],
    ),
  });
  return useCallback(
    (params: panel.NewTab[], options?: OpenTabOptions) => {
      if (params.length === 0) return;
      const panelKey = parentPanelKey ?? getSelected();
      if (panelKey != null) return insertIntoExisting(panelKey, params, options);
      // No panel to insert into: seed the tabs into the new panel's initial root so the
      // create persists them in a single request. This avoids a second remote dispatch
      // that would race the create and fail with "panel not found", while the local
      // store update keeps focus optimistic.
      createPanel({
        name: "New Panel",
        root: { variant: "leaf", tabs: params.map((p) => panel.tabZ.parse({ ...p })) },
        parent: project.ontologyID(getSelectedProject()),
      });
    },
    [parentPanelKey, getSelected, getSelectedProject, insertIntoExisting, createPanel],
  );
};

/** Opens a single tab. See {@link useOpenTabs} for placement and focus. */
export const useOpenTab = (): OpenTab => {
  const openTabs = useOpenTabs();
  return useCallback(
    (params: panel.NewTab, options?: OpenTabOptions) => openTabs([params], options),
    [openTabs],
  );
};

// The panel a gesture in this window acts on: the scoped one, else the selected one.
const useActiveID = (): ontology.ID | null => {
  const key = Panel.useOptionalKey() ?? Session.Panel.useSelectSelected();
  return key != null ? panel.ontologyID(key) : null;
};

/**
 * Reports whether this window can open a tab. A tab landing in a panel that exists is
 * an update on it; with no panel to land in, {@link useOpenTabs} mints one instead.
 */
export const useCanOpenTab = (): boolean => {
  const id = useActiveID();
  return Access.useGranted(
    id != null
      ? { objects: id, action: "update" }
      : { objects: panel.TYPE_ONTOLOGY_ID, action: "create" },
  );
};

/** Reports whether this window can restructure the panel it is acting on. */
export const useCanEditActive = (): boolean =>
  Access.useGranted({
    objects: useActiveID() ?? panel.TYPE_ONTOLOGY_ID,
    action: "update",
  });
