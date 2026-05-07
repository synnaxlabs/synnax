// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import {
  Access,
  Control,
  Diagram,
  Schematic as Base,
  Viewport,
} from "@synnaxlabs/pluto";
import { type color, type sticky } from "@synnaxlabs/x";
import { useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { Controller } from "@/schematic/Controller";
import { Controls } from "@/schematic/Controls";
import { useHandleNodeClickAction } from "@/schematic/navigate";
import { useSelect } from "@/schematic/selectors";
import {
  setEditable,
  setFitViewOnResize,
  setLegend,
  setSelected,
  setViewport,
  setViewportMode,
} from "@/schematic/slice";
import { useAutoUpload } from "@/schematic/useUpload";

export const Schematic: Layout.Renderer = ({ layoutKey: key, visible }) => {
  const doc = Base.useRetrieve({ key });
  const windowKey = useSelectWindowKey() as string;
  const updateSession = useDispatch();
  const { update: updateDoc } = Base.useDispatch();
  const {
    editable,
    viewport,
    controlStatus,
    selected,
    legend,
    authority,
    fitViewOnResize,
  } = useSelect(key);
  useAutoUpload(key);

  const hasUpdatePermission =
    Access.useUpdateGranted(schematic.ontologyID(key)) && !doc.snapshot;
  const canEdit = hasUpdatePermission && editable;

  const handleSelectionChange = useCallback(
    (selected: string[]) => updateSession(setSelected({ key, selected })),
    [updateSession, key],
  );

  const handleViewportChange = useCallback(
    (viewport: Diagram.Viewport) => updateSession(setViewport({ key, viewport })),
    [updateSession, key],
  );

  const handleEditableChange = useCallback(
    (editable: boolean) => updateSession(setEditable({ key, editable })),
    [updateSession, key],
  );

  const handleFitViewOnResizeChange = useCallback(
    (fitViewOnResize: boolean) =>
      updateSession(setFitViewOnResize({ key, fitViewOnResize })),
    [updateSession, key],
  );

  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => updateSession(setViewportMode({ key, mode })),
    [updateSession, key],
  );
  const triggers = useMemo(
    () => Viewport.DEFAULT_TRIGGERS[viewport.mode],
    [viewport.mode],
  );

  const handleLegendPositionChange = useCallback(
    (position: sticky.XY) => updateSession(setLegend({ key, legend: { position } })),
    [updateSession, key],
  );

  const handleLegendColorsChange = useCallback(
    (colors: Record<string, color.Color>) =>
      updateSession(setLegend({ key, legend: { colors } })),
    [key, updateSession],
  );

  const handleDoubleClick = useCallback(() => {
    if (editable)
      updateSession(
        Layout.setNavDrawerVisible({
          windowKey,
          key: "visualization",
          value: true,
        }),
      );
  }, [windowKey, editable, updateSession]);

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

  return (
    <Controller resourceKey={key} authority={authority}>
      <Base.Schematic
        resourceKey={key}
        onChange={updateDoc}
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
        {...doc}
      >
        <Diagram.Background />
        <Controls
          controlStatus={controlStatus}
          snapshot={doc.snapshot}
          hasUpdatePermission={hasUpdatePermission}
        />
      </Base.Schematic>
      {legend.colors && (
        <Control.Legend
          position={legend.position}
          onPositionChange={handleLegendPositionChange}
          colors={legend.colors}
          onColorsChange={handleLegendColorsChange}
          allowVisibleChange={false}
        />
      )}
    </Controller>
  );
};
