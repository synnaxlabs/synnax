// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel, project } from "@synnaxlabs/client";
import { type Flux, Panel } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Session } from "@/session";

export type OpenTab = (params: panel.NewTab) => void;

export const useOpenTab = (): OpenTab => {
  const dispatchSession = Session.useDispatch();
  const { dispatch } = Panel.useDispatch();
  const selectTab = Session.Panel.useSelectTab();
  const parentPanelKey = Panel.useOptionalKey();
  const getSelected = Session.Panel.useGetSelected();
  const getSelectedProject = Session.Project.useGetSelected();
  const parentTabKey = Panel.useOptionalTabKey();
  const getRoot = Panel.useGetRoot();
  // insertIntoExisting adds the tab to a panel that is already on the cluster
  // (the scoped parent or the selected panel), so a remote dispatch is correct.
  const insertIntoExisting = useCallback(
    (panelKey: panel.Key, params: panel.NewTab) => {
      const tab: panel.Tab = panel.tabZ.parse({ ...params });
      if (tab.variant === "resource") {
        const existing = panel.findTabByResource(
          getRoot({ key: panelKey }),
          tab.resource,
        );
        if (existing != null) return selectTab(existing.key, panelKey);
      }
      // A keyless tab opens beside the current one, but only when that tab lives in
      // this panel; otherwise its leaf can't be resolved and the insert would no-op,
      // so fall back to the first leaf.
      const besideCurrent =
        params.key == null &&
        parentTabKey != null &&
        panel.findTab(getRoot({ key: panelKey }), parentTabKey) != null;
      const action = besideCurrent
        ? panel.insertTab({ tab, targetTab: parentTabKey })
        : panel.insertTab({ tab });
      dispatch({ key: panelKey, actions: [action] });
      selectTab(tab.key, panelKey);
    },
    [parentTabKey, getRoot, dispatch, selectTab],
  );
  const { update: createPanel } = Panel.useCreate({
    afterOptimistic: useCallback(
      ({ data: { key, root }, rollbacks }: Flux.AfterOptimisticParams<panel.Panel>) => {
        dispatchSession(Session.Panel.select({ key }));
        rollbacks.push(() => dispatchSession(Session.Panel.clearSelected({})));
        // This hook only creates panels with a single-tab leaf root (below), so the
        // tab to focus is the one seeded into the root. Focus is dispatched
        // directly: the panel is not yet retrievable, so useSelectTab's cached
        // leaf lookup would fail.
        if (root.variant === "leaf")
          dispatchSession(
            Session.Panel.internalSelectTab({
              key,
              tabKey: root.tabs[0].key,
              otherTabKeys: root.tabs.map((t) => t.key),
            }),
          );
      },
      [dispatchSession],
    ),
  });
  return useCallback(
    (params: panel.NewTab) => {
      const panelKey = parentPanelKey ?? getSelected();
      if (panelKey != null) return insertIntoExisting(panelKey, params);
      // No panel to insert into: seed the tab into the new panel's initial root
      // so the create persists both in a single request. This avoids a second
      // remote dispatch that would race the create and fail with "panel not
      // found", while the local store update keeps focus optimistic.
      const tab: panel.Tab = panel.tabZ.parse({ ...params });
      createPanel({
        name: "New Panel",
        root: { variant: "leaf", tabs: [tab] },
        parent: project.ontologyID(getSelectedProject()),
      });
    },
    [parentPanelKey, getSelected, getSelectedProject, insertIntoExisting, createPanel],
  );
};
