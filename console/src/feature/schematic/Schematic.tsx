// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Control,
  Diagram,
  Menu,
  Panel as PlutoPanel,
  Schematic as Base,
  Viewport,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import { Controller } from "@/feature/schematic/Controller";
import { Controls } from "@/feature/schematic/Controls";
import { Legend } from "@/feature/schematic/Legend";
import { useHandleNodeClickAction } from "@/feature/schematic/navigate";
import { ContextMenu } from "@/platform/context-menu";
import { Empty } from "@/platform/empty";
import { type Panel } from "@/platform/panel";
import { Session } from "@/session";

const EmptyContent = (): ReactElement => {
  const key = Base.useKey();
  const dispatch = Session.useDispatch();
  const { canEdit } = Session.Schematic.useSelectEditable();
  const handleStartEditing = useCallback(() => {
    dispatch(Session.Schematic.setEditable({ key, editable: true }));
    dispatch(Session.Schematic.selectToolbarTab({ key, tab: "symbols" }));
    dispatch(Session.Nav.showBottom({}));
  }, [dispatch, key]);
  return (
    <Empty.Action
      message="No symbols in this schematic."
      action={canEdit ? "Start editing" : ""}
      onClick={handleStartEditing}
    />
  );
};

const Internal = (): ReactElement => {
  const key = Base.useKey();
  const isSnapshot = Base.useSelectSnapshot();
  const dispatch = Session.useDispatch();
  const viewport = Session.Schematic.useSelectViewport();
  const selected = Session.Schematic.useSelectSelected();
  const fitViewOnResize = Session.Schematic.useSelectFitViewOnResize();
  const visible = Session.Panel.useSelectIsTabVisible();
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

  const modals = Session.Modals.useStore("Schematic");
  const getTabIsFocused = Session.Panel.useGetTabIsFocused();

  const enableTriggers = useCallback(
    () => !modals.isAnyOpen() && getTabIsFocused() && isCurrentlyEditable,
    [getTabIsFocused, isCurrentlyEditable, modals],
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
        emptyContent={isCurrentlyEditable ? undefined : <EmptyContent />}
      >
        {isCurrentlyEditable && <Diagram.Background />}
        <Controls />
      </Base.Schematic>
      <Legend />
    </Controller>
  );
};

export const Schematic: Panel.Content = () => {
  const { key } = PlutoPanel.useSelectTabResource();
  return (
    <Base.Suspended schematicKey={key}>
      <Internal />
    </Base.Suspended>
  );
};
