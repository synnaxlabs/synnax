// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access, Arc as Base, Diagram, Viewport } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Controls } from "@/arc/editor/Controls";
import { useSelect, useSelectSelected, useSelectViewportMode } from "@/arc/selectors";
import {
  setEditable,
  setFitViewOnResize,
  setSelected,
  setViewport,
  setViewportMode,
} from "@/arc/slice";
import { ContextMenu as CMenu, Controls as BaseControls } from "@/components";
import { Layout } from "@/layout";

export const ContextMenu: Layout.ContextMenuRenderer = ({ layoutKey }) => (
  <CMenu.Menu>
    <Layout.MenuItems layoutKey={layoutKey} />
  </CMenu.Menu>
);

export const Editor: Layout.Renderer = ({ layoutKey, visible }): ReactElement => {
  const state = useSelect(layoutKey);
  const dispatch = useDispatch();
  const hasUpdatePermission = Access.useUpdateGranted(arc.ontologyID(layoutKey));
  const canEdit = hasUpdatePermission && state.graph.editable;
  const selected = useSelectSelected(layoutKey);
  const viewportMode = useSelectViewportMode();
  const triggers = useMemo(
    () => Viewport.DEFAULT_TRIGGERS[viewportMode],
    [viewportMode],
  );

  const handleSelectionChange = useCallback(
    (selected: string[]) => dispatch(setSelected({ key: layoutKey, selected })),
    [layoutKey, dispatch],
  );
  const handleViewportChange = useCallback(
    (viewport: Diagram.Viewport) => dispatch(setViewport({ key: layoutKey, viewport })),
    [layoutKey, dispatch],
  );
  const handleEditableChange = useCallback(
    (editable: boolean) => dispatch(setEditable({ key: layoutKey, editable })),
    [layoutKey, dispatch],
  );
  const handleSetFitViewOnResize = useCallback(
    (v: boolean) =>
      dispatch(setFitViewOnResize({ key: layoutKey, fitViewOnResize: v })),
    [layoutKey, dispatch],
  );
  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => dispatch(setViewportMode({ mode })),
    [dispatch],
  );
  const handleDoubleClick = useCallback(() => {
    if (!state.graph.editable) return;
    dispatch(Layout.setNavDrawerVisible({ key: "visualization", value: true }));
  }, [state.graph.editable, dispatch]);

  return (
    <>
      <Base.Graph.Graph
        resourceKey={layoutKey}
        viewport={state.graph.viewport}
        viewportMode={viewportMode}
        onViewportModeChange={handleViewportModeChange}
        onViewportChange={handleViewportChange}
        selected={selected}
        onSelectionChange={handleSelectionChange}
        onEditableChange={handleEditableChange}
        editable={canEdit}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        fitViewOnResize={state.graph.fitViewOnResize}
        setFitViewOnResize={handleSetFitViewOnResize}
        visible={visible}
      >
        <Diagram.Background />
        <BaseControls x>
          <Diagram.Controls.FitView />
          {hasUpdatePermission && <Diagram.Controls.ToggleEdit />}
        </BaseControls>
      </Base.Graph.Graph>
      <Controls state={state} />
    </>
  );
};
