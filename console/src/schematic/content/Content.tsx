// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Drift } from "@synnaxlabs/drift";
import {
  Control,
  Diagram,
  Menu,
  Schematic as PSchematic,
  Viewport,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch, useStore } from "react-redux";

import { ContextMenu } from "@/components/context-menu";
import { Nav } from "@/nav";
import { type Panel } from "@/panel";
import { selectIsFocused } from "@/panel/selectors";
import { Controller } from "@/schematic/content/Controller";
import { Controls } from "@/schematic/content/Controls";
import { useHandleNodeClickAction } from "@/schematic/content/navigate";
import { Session } from "@/schematic/session";
import { Tab } from "@/schematic/tab";

export interface ContentProps {
  visible?: boolean;
}

export const Content = Tab.createSuspended<ContentProps>(({ visible = true }) => {
  const key = PSchematic.useKey();
  const dispatch = useDispatch();
  const isSnapshot = PSchematic.useSelectSnapshot({});
  const { editable, viewport, selected, fitViewOnResize } = Session.useSelectState();
  const handleSelectionChange = Session.useSetSelected();
  const handleViewportChange = Session.useSetViewport();
  const handleEditableChange = Session.useSetEditable();
  const handleFitViewOnResizeChange = Session.useSetFitViewOnResize();
  const handleViewportModeChange = Session.useSetViewportMode();
  const triggers = useMemo(
    () => Viewport.DEFAULT_TRIGGERS[viewport.mode],
    [viewport.mode],
  );
  const handleDoubleClick = useCallback(() => {
    if (editable) dispatch(Nav.setBottomVisible(true));
  }, [editable, dispatch]);
  const handleNodeClickAction = useHandleNodeClickAction(key);
  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: { id: string }) =>
      handleNodeClickAction(node.id, false),
    [handleNodeClickAction],
  );
  const handleNodeDoubleClick = useCallback(
    (_: React.MouseEvent, node: { id: string }) => handleNodeClickAction(node.id, true),
    [handleNodeClickAction],
  );

  const store = useStore<Panel.StoreState & Drift.StoreState>();
  const enableTriggers = useCallback(
    () => selectIsFocused(store.getState(), key),
    [store, key],
  );

  const renderExtraMenuItems = useCallback(
    (): ReactElement => (
      <>
        {!isSnapshot && <Control.Menu.ToggleItem />}
        <Menu.Divider />
        <ContextMenu.ReloadConsoleItem />
      </>
    ),
    [isSnapshot],
  );

  return (
    <Controller>
      <PSchematic.Schematic
        enableTriggers={enableTriggers}
        extraMenuItems={renderExtraMenuItems}
        selected={selected}
        onSelectionChange={handleSelectionChange}
        viewportMode={viewport.mode}
        onViewportModeChange={handleViewportModeChange}
        viewport={viewport}
        onViewportChange={handleViewportChange}
        editable={editable}
        onEditableChange={handleEditableChange}
        fitViewOnResize={fitViewOnResize}
        setFitViewOnResize={handleFitViewOnResizeChange}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        onNodeClick={handleNodeClick}
        onNodeDoubleClick={handleNodeDoubleClick}
        visible={visible}
      >
        <Diagram.Background />
        <Controls />
      </PSchematic.Schematic>
    </Controller>
  );
});
