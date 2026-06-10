// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { panel } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Flux, Panel as Base, type Pluto } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { selectActivePanelKey, selectActiveTabKey } from "@/layout/selectors";
import { setActivePanel, setActiveTab } from "@/layout/slice";
import { Project } from "@/project";
import { type RootState } from "@/store";

// useCreateEmptyTab returns a callback that inserts a contentless tab (which
// renders the component selector) into the active panel's active leaf and
// selects it. Returns false when there is no active panel to insert into, so
// callers can fall through to creating a panel first.
export const useCreateEmptyTab = (): (() => boolean) => {
  const store = useStore<RootState>();
  const reduxDispatch = useDispatch();
  const windowKey = useSelectWindowKey();
  const fluxStore = Flux.useStore<Pluto.FluxStore>();
  const { dispatch } = Base.useDispatch();
  return useCallback(() => {
    if (windowKey == null) return false;
    const state = store.getState();
    const panelKey = selectActivePanelKey(state);
    const cached = panelKey != null ? fluxStore.panels.get(panelKey) : null;
    if (panelKey == null || cached == null) return false;
    const activeTab = selectActiveTabKey(state);
    const targetLeaf =
      (activeTab != null ? panel.tabLeafPath(cached.root, activeTab) : null) ??
      panel.firstLeafPath(cached.root) ??
      panel.ROOT_PATH;
    const tab: panel.Tab = { key: uuid.create() };
    dispatch({ key: panelKey, actions: [panel.insertTab({ tab, targetLeaf })] });
    reduxDispatch(setActiveTab({ windowKey, key: tab.key }));
    return true;
  }, [store, fluxStore, dispatch, reduxDispatch, windowKey]);
};

// useCreatePanel returns a callback that creates an "Untitled" panel in the
// active project, makes it the window's active panel, and seeds it with one
// empty (selector) tab so the user lands in the component picker. No-ops when
// there is no active project (panels need an owner until drafts land).
export const useCreatePanel = (): (() => void) => {
  const reduxDispatch = useDispatch();
  const windowKey = useSelectWindowKey();
  const activeProjectKey = Project.useSelectActiveKey();
  const { dispatch } = Base.useDispatch();
  const { update } = Base.useCreate({
    afterSuccess: useCallback(
      ({ data }: Flux.AfterSuccessParams<panel.Panel>) => {
        if (windowKey == null) return;
        reduxDispatch(setActivePanel({ windowKey, key: data.key }));
        const tab: panel.Tab = { key: uuid.create() };
        dispatch({
          key: data.key,
          actions: [panel.insertTab({ tab, targetLeaf: panel.ROOT_PATH })],
        });
        reduxDispatch(setActiveTab({ windowKey, key: tab.key }));
      },
      [reduxDispatch, dispatch, windowKey],
    ),
  });
  return useCallback(() => {
    if (windowKey == null || activeProjectKey == null) return;
    update({ name: "Untitled", project: activeProjectKey });
  }, [update, windowKey, activeProjectKey]);
};
