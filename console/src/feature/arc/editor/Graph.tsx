// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc as Base, Diagram, Menu, Viewport } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import { TaskControls } from "@/feature/arc/editor/TaskControls";
import { ContextMenu } from "@/platform/context-menu";
import { Vis } from "@/platform/vis";
import { Session } from "@/session";

export interface GraphProps {
  visible: boolean;
}

export const Graph = ({ visible }: GraphProps): ReactElement => {
  const key = Base.useKey();
  const viewport = Session.Arc.useSelectViewport();
  const fitViewOnResize = Session.Arc.useSelectFitViewOnResize();
  const dispatch = Session.useDispatch();
  const { canEdit, isCurrentlyEditable } = Session.Arc.useSelectEditable();
  const selected = Session.Arc.useSelectSelected();
  const viewportMode = Session.Arc.useSelectViewportMode();

  const triggers = useMemo(
    () => Viewport.DEFAULT_TRIGGERS[viewportMode],
    [viewportMode],
  );

  const handleSelectionChange = useCallback(
    (selected: string[]) => dispatch(Session.Arc.setSelected({ key, selected })),
    [key, dispatch],
  );
  const handleViewportChange = useCallback(
    (viewport: Diagram.Viewport) =>
      dispatch(Session.Arc.setViewport({ key, viewport })),
    [key, dispatch],
  );
  const handleEditableChange = useCallback(
    (editable: boolean) => dispatch(Session.Arc.setEditable({ key, editable })),
    [key, dispatch],
  );
  const handleSetFitViewOnResize = useCallback(
    (fitViewOnResize: boolean) =>
      dispatch(Session.Arc.setFitViewOnResize({ key, fitViewOnResize })),
    [key, dispatch],
  );
  const handleViewportModeChange = useCallback(
    (mode: Viewport.Mode) => dispatch(Session.Arc.setViewportMode({ key, mode })),
    [key, dispatch],
  );
  const handleDoubleClick = useCallback(
    () => dispatch(Session.Nav.showBottom({})),
    [dispatch],
  );

  const renderExtraMenuItems = useCallback(
    (): ReactElement => (
      <>
        {canEdit && <Diagram.Menu.ToggleEditItem />}
        <Menu.Divider />
        <ContextMenu.ReloadConsoleItem />
      </>
    ),
    [canEdit],
  );

  return (
    <>
      <Base.Graph.Editor
        extraMenuItems={renderExtraMenuItems}
        viewport={viewport}
        viewportMode={viewportMode}
        onViewportModeChange={handleViewportModeChange}
        onViewportChange={handleViewportChange}
        selected={selected}
        onSelectionChange={handleSelectionChange}
        onEditableChange={handleEditableChange}
        editable={isCurrentlyEditable}
        triggers={triggers}
        onDoubleClick={handleDoubleClick}
        fitViewOnResize={fitViewOnResize}
        setFitViewOnResize={handleSetFitViewOnResize}
        visible={visible}
      >
        <Diagram.Background />
        <Vis.Controls x>
          <Diagram.Controls.FitView />
          {canEdit && <Diagram.Controls.ToggleEdit />}
        </Vis.Controls>
      </Base.Graph.Editor>
      <TaskControls />
    </>
  );
};
