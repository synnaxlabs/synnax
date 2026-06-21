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
import { useDispatch, useStore } from "react-redux";

import { ContextMenu } from "@/components/context-menu";
import { Layout } from "@/layout";
import { Controller } from "@/schematic/body/Controller";
import { Controls } from "@/schematic/body/Controls";
import { Legend } from "@/schematic/body/Legend";
import { useHandleNodeClickAction } from "@/schematic/body/navigate";
import { Session } from "@/schematic/session";
import { type RootState } from "@/store";

const Internal: Layout.Renderer = ({ visible }) => {
  const key = Base.useKey();
  const isSnapshot = Base.useSelectSnapshot({});
  const dispatch = useDispatch();
  const viewport = Session.useSelectViewport();
  const selected = Session.useSelectSelected();
  const fitViewOnResize = Session.useSelectFitViewOnResize();
  const { isCurrentlyEditable, canEdit } = Session.useSelectEditable();

  const handleSelectionChange = useCallback(
    (selected: string[]) => dispatch(Session.setSelected({ key, selected })),
    [dispatch, key],
  );

  const handleViewportChange = useCallback(
    (viewport: Diagram.Viewport) => dispatch(Session.setViewport({ key, viewport })),
    [dispatch, key],
  );

  const handleEditableChange = useCallback(
    (editable: boolean) => dispatch(Session.setEditable({ key, editable })),
    [dispatch, key],
  );

  const handleFitViewOnResizeChange = useCallback(
    (fitViewOnResize: boolean) =>
      dispatch(Session.setFitViewOnResize({ key, fitViewOnResize })),
    [dispatch, key],
  );

  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => dispatch(Session.setViewportMode({ key, mode })),
    [dispatch, key],
  );

  const triggers = useMemo(
    () => Viewport.DEFAULT_TRIGGERS[viewport.mode],
    [viewport.mode],
  );

  const handleDoubleClick = useCallback(() => {
    if (isCurrentlyEditable)
      dispatch(Layout.setNavDrawerVisible({ key: "visualization", value: true }));
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

  const store = useStore<RootState>();

  const enableTriggers = useCallback(
    () =>
      Layout.selectActiveMosaicTabKeyAndNotBlurred(store.getState()) === key &&
      isCurrentlyEditable,
    [store, key, isCurrentlyEditable],
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
        editable={canEdit}
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
      </Base.Schematic>
      <Legend />
    </Controller>
  );
};

export const Schematic: Layout.Renderer = (props) => (
  <Base.Suspended schematicKey={props.layoutKey}>
    <Internal {...props} />
  </Base.Suspended>
);
Schematic.useName = Layout.createUseFluxName(
  Base.useRename,
  Base.useRetrieveObservableName,
);
