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
  const windowKey = useSelectWindowKey() as string;
  const { data: doc } = Base.useRetrieve({ key });
  const snapshot = doc?.snapshot ?? false;

  const dispatch = useDispatch();
  const { editable, viewport, controlStatus, selected, legend, authority } =
    useSelect(key);
  useAutoUpload(key);

  const hasUpdatePermission =
    Access.useUpdateGranted(schematic.ontologyID(key)) && !snapshot;
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
      dispatch(
        Layout.setNavDrawerVisible({
          windowKey,
          key: "visualization",
          value: true,
        }),
      );
  }, [windowKey, editable, dispatch]);

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
        selected={selected}
        onSelectionChange={handleSelectionChange}
        viewportMode={viewport.mode}
        onViewportModeChange={handleViewportModeChange}
        viewport={viewport}
        onViewportChange={handleViewportChange}
        editable={canEdit}
        onEditableChange={handleEditableChange}
        setFitViewOnResize={handleFitViewOnResizeChange}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        onNodeClick={handleNodeClick}
        onNodeDoubleClick={handleNodeDoubleClick}
        fitViewOnResize={false}
        visible={visible}
      >
        <Diagram.Background />
        <Controls
          controlStatus={controlStatus}
          snapshot={snapshot}
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
