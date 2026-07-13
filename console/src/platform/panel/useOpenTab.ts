// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { type Flux, Panel } from "@synnaxlabs/pluto";
import { type optional, uuid } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Session } from "@/session";

export type OpenTabParams =
  optional.Optional<panel.TabResource, "key"> | optional.Optional<panel.TabView, "key">;

export type OpenTab = (params: OpenTabParams) => void;

export const useOpenTab = (): ((params: OpenTabParams) => void) => {
  const dispatchSession = Session.useDispatch();
  const { dispatch } = Panel.useDispatch();
  const selectTab = Session.Panel.useSelectTab();
  const parentPanelKey = Panel.useOptionalKey();
  const getSelected = Session.Panel.useGetSelected();
  const parentTabKey = Panel.useOptionalTabKey();
  // insertIntoExisting adds the tab to a panel that is already on the cluster
  // (the scoped parent or the selected panel), so a remote dispatch is correct.
  const insertIntoExisting = useCallback(
    (panelKey: panel.Key, params: OpenTabParams) => {
      const tab: panel.Tab = { key: parentTabKey ?? uuid.create(), ...params };
      dispatch({ key: panelKey, actions: [panel.insertTab({ tab })] });
      selectTab(tab.key, panelKey);
    },
    [parentTabKey, dispatch, selectTab],
  );
  const { update: createPanel } = Panel.useCreate({
    afterOptimistic: useCallback(
      ({
        data: { key, root },
        rollbacks,
      }: Flux.AfterOptimisticParams<panel.Panel, false, Panel.FluxSubStore>) => {
        dispatchSession(Session.Panel.select({ key }));
        rollbacks.push(() => dispatchSession(Session.Panel.clearSelected({})));
        // This hook only creates panels with a single-tab leaf root (below), so the
        // tab to focus is the one seeded into the root.
        if (root.variant === "leaf") selectTab(root.tabs[0].key, key);
      },
      [dispatchSession, selectTab],
    ),
  });
  return useCallback(
    (params: OpenTabParams) => {
      const panelKey = parentPanelKey ?? getSelected();
      if (panelKey != null) return insertIntoExisting(panelKey, params);
      // No panel to insert into: seed the tab into the new panel's initial root
      // so the create persists both in a single request. This avoids a second
      // remote dispatch that would race the create and fail with "panel not
      // found", while the local store update keeps focus optimistic.
      const tab: panel.Tab = { key: parentTabKey ?? uuid.create(), ...params };
      createPanel({ name: "New Panel", root: { variant: "leaf", tabs: [tab] } });
    },
    [parentPanelKey, parentTabKey, getSelected, insertIntoExisting, createPanel],
  );
};
