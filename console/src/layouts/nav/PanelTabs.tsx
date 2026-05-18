// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as Drift from "@synnaxlabs/drift/react";
import { Tabs } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";

export const PanelTabs = (): ReactElement | null => {
  const dispatch = useDispatch();
  const windowKey = Drift.useSelectWindowKey();
  const panels = Layout.useSelectOrderedPanels();
  const activeKey = Layout.useSelectActivePanelKey();

  const tabs = useMemo<Tabs.Tab[]>(
    () =>
      panels.map((p) => ({
        tabKey: p.key,
        name: p.name,
        closable: false,
        editable: true,
      })),
    [panels],
  );

  const handleSelect = useCallback(
    (key: string) => {
      if (windowKey == null) return;
      dispatch(Layout.setActivePanel({ windowKey, key }));
    },
    [dispatch, windowKey],
  );

  const handleClose = useCallback(
    (key: string) => {
      if (windowKey == null) return;
      dispatch(Layout.removePanel({ windowKey, key }));
    },
    [dispatch, windowKey],
  );

  const handleRename = useCallback(
    (key: string, name: string) => {
      dispatch(Layout.renamePanel({ key, name }));
    },
    [dispatch],
  );

  const handleCreate = useCallback(() => {
    if (windowKey == null) return;
    dispatch(Layout.createPanel({ windowKey, name: "Untitled" }));
  }, [dispatch, windowKey]);

  if (windowKey == null) return null;

  return (
    <Tabs.Provider
      value={{
        tabs,
        selected: activeKey ?? undefined,
        closable: true,
        onSelect: handleSelect,
        onClose: handleClose,
        onRename: handleRename,
        onCreate: handleCreate,
      }}
    >
      <Tabs.Selector size="medium" variant="pill" />
    </Tabs.Provider>
  );
};
