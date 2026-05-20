// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as Drift from "@synnaxlabs/drift/react";
import { Panel as PlutoPanel, Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useEffect, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { Project } from "@/project";

// PanelTabs renders the project's panel tab strip in the top nav. Source of
// truth is Flux (panel.useList); per-window active panel state lives in the
// Redux layout slice. When there is no active project, the strip renders
// nothing — the welcome / project-picker handles that case at the viewport
// level.
export const PanelTabs = (): ReactElement | null => {
  const dispatch = useDispatch();
  const windowKey = Drift.useSelectWindowKey();
  const activeProjectKey = Project.useSelectActiveKey();
  const activeKey = Layout.useSelectActivePanelKey();
  const { data, retrieve, getItem } = PlutoPanel.useList();
  const { updateAsync: createAsync } = PlutoPanel.useCreate();

  // Fetch the panel list once on mount. Reactive channel listeners on the
  // Pluto panel store keep it fresh from here on.
  useEffect(() => {
    if (activeProjectKey == null) return;
    retrieve({});
  }, [activeProjectKey, retrieve]);

  const tabs = useMemo<Tabs.Tab[]>(() => {
    const out: Tabs.Tab[] = [];
    for (const key of data) {
      const p = getItem(key);
      if (p == null || Array.isArray(p)) continue;
      out.push({ tabKey: p.key, name: p.name, closable: true, editable: true });
    }
    return out;
  }, [data, getItem]);

  const handleSelect = useCallback(
    (key: string) => {
      if (windowKey == null) return;
      dispatch(Layout.setActivePanel({ windowKey, key }));
    },
    [dispatch, windowKey],
  );

  const handleCreate = useCallback(() => {
    if (windowKey == null || activeProjectKey == null) return;
    // Fire and forget: the action channel listener surfaces the new panel
    // into the Flux store, and the autoselect effect below picks it up.
    void createAsync({ name: "Untitled", project: activeProjectKey });
  }, [windowKey, activeProjectKey, createAsync]);

  // Autoselect: when no panel is active but the project has at least one,
  // select the first. Keeps the viewport from showing the empty-no-panel
  // state once a project has content.
  useEffect(() => {
    if (windowKey == null || activeKey != null || tabs.length === 0) return;
    dispatch(Layout.setActivePanel({ windowKey, key: tabs[0].tabKey }));
  }, [windowKey, activeKey, tabs, dispatch]);

  const providerValue = useMemo(
    () => ({
      tabs,
      selected: activeKey ?? undefined,
      closable: true,
      onSelect: handleSelect,
      onCreate: handleCreate,
    }),
    [tabs, activeKey, handleSelect, handleCreate],
  );

  if (windowKey == null || activeProjectKey == null) return null;

  return (
    <Tabs.Provider value={providerValue}>
      <Tabs.Selector size="medium" variant="pill" />
    </Tabs.Provider>
  );
};
