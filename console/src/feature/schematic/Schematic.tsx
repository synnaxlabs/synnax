// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Control, Diagram, Menu, Schematic as Base, Viewport } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import { Controller } from "@/feature/schematic/Controller";
import { Controls } from "@/feature/schematic/Controls";
import { Legend } from "@/feature/schematic/Legend";
import { useHandleNodeClickAction } from "@/feature/schematic/navigate";
import { ContextMenu } from "@/platform/context-menu";
import { Session } from "@/session";

export interface SchematicProps {
  visible: boolean;
}

export const Schematic = ({ visible }: SchematicProps) => {
  const key = Base.useKey();
  const isSnapshot = Base.useSelectSnapshot();
  const dispatch = Session.useDispatch();
  const viewport = Session.Schematic.useSelectViewport();
  const selected = Session.Schematic.useSelectSelected();
  const fitViewOnResize = Session.Schematic.useSelectFitViewOnResize();
  const { isCurrentlyEditable, canEdit } = Session.Schematic.useSelectEditable();

  const handleSelectionChange = useCallback(
    (selected: string[]) => dispatch(Session.Schematic.setSelected({ key, selected })),
    [dispatch, key],
  );

  const handleViewportChange = useCallback(
    (viewport: Diagram.Viewport) =>
      dispatch(Session.Schematic.setViewport({ key, viewport })),
    [dispatch, key],
  );

  const handleEditableChange = useCallback(
    (editable: boolean) => dispatch(Session.Schematic.setEditable({ key, editable })),
    [dispatch, key],
  );

  const handleFitViewOnResizeChange = useCallback(
    (fitViewOnResize: boolean) =>
      dispatch(Session.Schematic.setFitViewOnResize({ key, fitViewOnResize })),
    [dispatch, key],
  );

  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => dispatch(Session.Schematic.setViewportMode({ key, mode })),
    [dispatch, key],
  );

  const triggers = useMemo(
    () => Viewport.DEFAULT_TRIGGERS[viewport.mode],
    [viewport.mode],
  );

  const handleDoubleClick = useCallback(() => {
    if (isCurrentlyEditable) dispatch(Session.Nav.showBottom({}));
  }, [isCurrentlyEditable, dispatch]);

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

  const store = Session.useStore();
  const modals = Session.Modals.useStore("Schematic");

  const enableTriggers = useCallback(
    () =>
      Session.Layout.selectActiveMosaicTabKeyAndNotBlurred(store.getState(), modals) ===
        key && isCurrentlyEditable,
    [store, key, isCurrentlyEditable, modals],
  );

  const renderExtraMenuItems = useCallback(
    (): ReactElement => (
      <>
        {canEdit && <Diagram.Menu.ToggleEditItem />}
        {!isSnapshot && <Control.Menu.ToggleItem />}
        <Menu.Divider />
        <ContextMenu.ReloadConsoleItem />
      </>
    ),
    [canEdit, isSnapshot],
  );

  return (
    <Controller>
      <Base.Schematic
        enableTriggers={enableTriggers}
        extraMenuItems={renderExtraMenuItems}
        selected={selected}
        onSelectionChange={handleSelectionChange}
        viewportMode={viewport.mode}
        onViewportModeChange={handleViewportModeChange}
        viewport={viewport}
        onViewportChange={handleViewportChange}
        editable={isCurrentlyEditable}
        onEditableChange={handleEditableChange}
        fitViewOnResize={fitViewOnResize}
        setFitViewOnResize={handleFitViewOnResizeChange}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        onNodeClick={handleNodeClick}
        onNodeDoubleClick={handleNodeDoubleClick}
        visible={visible}
      >
        {isCurrentlyEditable && <Diagram.Background />}
        <Controls />
      </Base.Schematic>
      <Legend />
    </Controller>
  );
};
