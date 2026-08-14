// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, panel, project, query } from "@synnaxlabs/client";
import { type Flux, Panel, Synnax } from "@synnaxlabs/pluto";
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

type InsertTarget = Pick<
  panel.InsertTabPayload,
  "targetLeaf" | "targetTab" | "location"
>;

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
      const cached = client?.panels.getCached(panelKey);
      if (!query.isLive(cached))
        throw new NotFoundError(`Panel with key ${panelKey} not found`);
      const { root } = cached;
      const { singleton, placement } = options ?? {};
      // A keyless tab opens beside the current one, but only when that tab lives in
      // this panel; otherwise its leaf can't be resolved and the insert would no-op, so
      // fall back to the first leaf.
      const besideCurrent =
        params[0]?.key == null &&
        parentTabKey != null &&
        panel.findTab(root, parentTabKey) != null;
      const placed =
        placement != null && panel.findNode(root, placement.leaf)?.variant === "leaf"
          ? placement
          : undefined;
      let target: InsertTarget = placed
        ? { targetLeaf: placed.leaf, location: placed.location }
        : besideCurrent
          ? { targetTab: parentTabKey }
          : {};
      const actions: panel.Action[] = [];
      let focus: panel.TabKey | undefined;
      params.forEach((p) => {
        const tab: panel.Tab = panel.tabZ.parse({ ...p });
        if (tab.variant === "resource") {
          const existing = panel.findTabByResource(root, tab.resource);
          if (existing != null) {
            focus = existing.key;
            return;
          }
        }
        if (tab.variant === "view" && singleton) {
          const existing = panel.findTabByType(root, tab.type);
          if (existing != null) {
            focus = existing.key;
            return;
          }
        }
        actions.push(panel.insertTab({ tab, ...target, singleton }));
        // Later tabs target the one before them so they join its leaf, whether the
        // first insert split the placed leaf or landed in an existing one.
        target = { targetTab: tab.key };
        focus = tab.key;
      });
      if (actions.length > 0) dispatch({ key: panelKey, actions });
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
