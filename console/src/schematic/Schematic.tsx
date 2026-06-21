// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import {
  Access,
  Control,
  Diagram,
  Menu,
  Schematic as Base,
  Viewport,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch, useStore } from "react-redux";

import { ContextMenu } from "@/components/context-menu";
import { Layout } from "@/layout";
import { Controller } from "@/schematic/Controller";
import { Controls } from "@/schematic/Controls";
import { useHandleNodeClickAction } from "@/schematic/navigate";
import {
  selectEditable,
  useSelectEditable,
  useSelectFitViewOnResize,
  useSelectSelected,
  useSelectViewport,
} from "@/schematic/selectors";
import {
  setEditable,
  setFitViewOnResize,
  setSelected,
  setViewport,
  setViewportMode,
} from "@/schematic/slice";
import { type RootState } from "@/store";

import { Legend } from "./Legend";

const Internal: Layout.Renderer = ({ layoutKey: key, visible }) => {
  const isSnapshot = Base.useSelectSnapshot({});
  const dispatch = useDispatch();
  const editable = useSelectEditable(key);
  const viewport = useSelectViewport(key);
  const selected = useSelectSelected(key);
  const fitViewOnResize = useSelectFitViewOnResize(key);
  const hasUpdatePermission =
    Access.useUpdateGranted(schematic.ontologyID(key)) && !isSnapshot;
  const canEdit = hasUpdatePermission && editable;

  const handleSelectionChange = useCallback(
    (selected: string[]) => dispatch(setSelected({ key, selected })),
    [dispatch, key],
  );

  const handleViewportChange = useCallback(
    (viewport: Diagram.Viewport) => dispatch(setViewport({ key, viewport })),
    [dispatch, key],
  );

  const handleEditableChange = useCallback(
    (editable: boolean) => dispatch(setEditable({ key, editable })),
    [dispatch, key],
  );

  const handleFitViewOnResizeChange = useCallback(
    (fitViewOnResize: boolean) =>
      dispatch(setFitViewOnResize({ key, fitViewOnResize })),
    [dispatch, key],
  );

  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => dispatch(setViewportMode({ key, mode })),
    [dispatch, key],
  );

  const triggers = useMemo(
    () => Viewport.DEFAULT_TRIGGERS[viewport.mode],
    [viewport.mode],
  );

  const handleDoubleClick = useCallback(() => {
    if (editable)
      dispatch(Layout.setNavDrawerVisible({ key: "visualization", value: true }));
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

  const store = useStore<RootState>();

  const enableTriggers = useCallback(
    () =>
      Layout.selectActiveMosaicTabKeyAndNotBlurred(store.getState()) === key &&
      hasUpdatePermission &&
      selectEditable(store.getState(), key),
    [store, key, hasUpdatePermission],
  );

  const renderExtraMenuItems = useCallback(
    (): ReactElement => (
      <>
        {hasUpdatePermission && <Diagram.Menu.ToggleEditItem />}
        {!isSnapshot && <Control.Menu.ToggleItem />}
        <Menu.Divider />
        <ContextMenu.ReloadConsoleItem />
      </>
    ),
    [hasUpdatePermission, isSnapshot],
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
        <Controls snapshot={isSnapshot} hasUpdatePermission={hasUpdatePermission} />
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
