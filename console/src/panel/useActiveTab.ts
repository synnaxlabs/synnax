// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { Flux, Panel as Base, type Pluto } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useStore } from "react-redux";

import { selectActivePanelKey, selectActiveTabState } from "@/layout/selectors";
import { type RootState } from "@/store";

// useResolveActiveTab returns an imperative resolver for the effective active tab
// of the current window: the session cursor when set, otherwise the active
// panel's first tab (mirroring the mosaic adapter's default-to-first-tab
// behavior). Returns null when there is no active panel or it has no tabs.
export const useResolveActiveTab = (): (() => string | null) => {
  const store = useStore<RootState>();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  return useCallback(() => {
    const state = store.getState();
    const { tabKey } = selectActiveTabState(state);
    if (tabKey != null) return tabKey;
    const panelKey = selectActivePanelKey(state);
    if (panelKey == null) return null;
    const cached = fluxStore.panels.get(panelKey);
    return panel.firstTab(cached?.root)?.key ?? null;
  }, [store, fluxStore]);
};

// useCloseActiveTab returns a callback that removes the effective active tab from
// the active panel. Returns false when there is no active panel tab to close, so
// callers (e.g. the Ctrl+W trigger) can fall through to their next target.
export const useCloseActiveTab = (): (() => boolean) => {
  const store = useStore<RootState>();
  const resolveActiveTab = useResolveActiveTab();
  const { dispatch } = Base.useDispatch();
  return useCallback(() => {
    const panelKey = selectActivePanelKey(store.getState());
    const tabKey = resolveActiveTab();
    if (panelKey == null || tabKey == null) return false;
    dispatch({ key: panelKey, actions: [panel.removeTab({ key: tabKey })] });
    return true;
  }, [store, resolveActiveTab, dispatch]);
};
