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
import { type color, type sticky } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch, useStore } from "react-redux";

import { ContextMenu } from "@/components/context-menu";
import { createEnsureState } from "@/hooks/useEnsureState";
import { Layout } from "@/layout";
import { Controller } from "@/schematic/Controller";
import { Controls } from "@/schematic/Controls";
import { useHandleNodeClickAction } from "@/schematic/navigate";
import { selectEditable, useSelect, useSelectExists } from "@/schematic/selectors";
import {
  internalCreate,
  setEditable,
  setFitViewOnResize,
  setLegend,
  setSelected,
  setViewport,
  setViewportMode,
} from "@/schematic/slice";
import { useAutoUpload } from "@/schematic/useUpload";
import { type RootState } from "@/store";

const Internal: Layout.Renderer = ({ layoutKey: key, visible }) => {
  Base.useEnsureRetrieved({ key });
  const isSnapshot = Base.useSelectSnapshot({ key });
  const dispatch = useDispatch();
  const {
    editable,
    viewport,
    controlStatus,
    selected,
    legend,
    authority,
    fitViewOnResize,
  } = useSelect(key);

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

  const handleLegendPositionChange = useCallback(
    (position: sticky.XY) => dispatch(setLegend({ key, legend: { position } })),
    [dispatch, key],
  );

  const handleLegendColorsChange = useCallback(
    (colors: Record<string, color.Color>) =>
      dispatch(setLegend({ key, legend: { colors } })),
    [key, dispatch],
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
    <Controller resourceKey={key} authority={authority}>
      <Base.Schematic
        enableTriggers={enableTriggers}
        extraMenuItems={renderExtraMenuItems}
        resourceKey={key}
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
        <Controls
          controlStatus={controlStatus}
          snapshot={isSnapshot}
          hasUpdatePermission={hasUpdatePermission}
        />
      </Base.Schematic>
      {legend.visible && (
        <Control.Legend
          position={legend.position}
          onPositionChange={handleLegendPositionChange}
          colors={legend.colors}
          onColorsChange={handleLegendColorsChange}
          allowEntryVisibleChange={false}
        />
      )}
    </Controller>
  );
};

const useEnsureState = createEnsureState({
  useExists: useSelectExists,
  create: (key) => internalCreate({ key }),
});

export const Schematic: Layout.Renderer = (props) => {
  const { layoutKey } = props;
  const exists = useEnsureState(layoutKey);
  const uploaded = useAutoUpload(layoutKey);
  if (!exists || !uploaded) return null;
  return <Internal {...props} />;
};
Schematic.useName = Layout.createUseFluxName(
  Base.useRename,
  Base.useRetrieveObservableName,
);
