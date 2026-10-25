// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/lyra/menu";
import {
  Control,
  Diagram,
  Panel as PlutoPanel,
  Schematic as Base,
  Viewport,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

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
  const handleStartEditing = () => {
    dispatch(Session.Schematic.setEditable({ key, editable: true }));
    dispatch(Session.Schematic.selectToolbarTab({ key, tab: "symbols" }));
    dispatch(Session.Nav.showBottom({}));
  };
  return (
    <Empty.Action
      message="No symbols in this schematic"
      action={canEdit ? "Start editing" : ""}
      onClick={handleStartEditing}
    />
  );
};

const Internal = (): ReactElement => {
  const key = Base.useKey();
  const isSnapshot = Base.useIsSnapshot();
  const dispatch = Session.useDispatch();
  const viewport = Session.Schematic.useSelectViewport();
  const selected = Session.Schematic.useSelectSelected();
  const fitViewOnResize = Session.Schematic.useSelectFitViewOnResize();
  const tooltipsDisabled = Session.Schematic.useSelectTooltipsDisabled();
  const visible = Session.Panel.useSelectIsTabVisible();
  const { isCurrentlyEditable, canEdit } = Session.Schematic.useSelectEditable();

  const handleSelectionChange = (selected: string[]) =>
    dispatch(Session.Schematic.setSelected({ key, selected }));

  const handleViewportChange = (viewport: Diagram.Viewport) =>
    dispatch(Session.Schematic.setViewport({ key, viewport }));

  const handleEditableChange = (editable: boolean) =>
    dispatch(Session.Schematic.setEditable({ key, editable }));

  const handleFitViewOnResizeChange = (fitViewOnResize: boolean) =>
    dispatch(Session.Schematic.setFitViewOnResize({ key, fitViewOnResize }));

  const handleViewportModeChange = (mode: Viewport.Mode) =>
    dispatch(Session.Schematic.setViewportMode({ key, mode }));

  const triggers = Viewport.DEFAULT_TRIGGERS[viewport.mode];

  const handleDoubleClick = () => {
    if (isCurrentlyEditable) dispatch(Session.Nav.showBottom({}));
  };

  const handleNodeClickAction = useHandleNodeClickAction(key);

  const handleNodeClick = (_: React.MouseEvent, node: { id: string }) =>
    handleNodeClickAction(node.id, false);

  const handleNodeDoubleClick = (_: React.MouseEvent, node: { id: string }) =>
    handleNodeClickAction(node.id, true);

  const renderExtraMenuItems = (): ReactElement => (
    <>
      {canEdit && <Diagram.Menu.ToggleEditItem />}
      {!isSnapshot && <Control.Menu.ToggleItem />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </>
  );

  return (
    <Controller>
      <Base.Schematic
        enableTriggers={isCurrentlyEditable}
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
        tooltipsDisabled={tooltipsDisabled}
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
  const { key } = PlutoPanel.useTabResource();
  return (
    <Base.Suspended schematicKey={key}>
      <Internal />
    </Base.Suspended>
  );
};
