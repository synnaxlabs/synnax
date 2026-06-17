// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/Table.css";

import { table } from "@synnaxlabs/client";
import { Access, Button, Icon, Table as Base } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu, Controls } from "@/components";
import { CSS } from "@/css";
import { Nav } from "@/nav";
import { Session } from "@/table/session";
import { Tab } from "@/table/tab";

export interface ContentProps {
  visible?: boolean;
}

const TableControls = (): ReactElement | null => {
  const key = Base.useKey();
  const editable = Session.useSelectEditable();
  const hideIndicators = Session.useSelectHideIndicators();
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(key));
  const dispatch = useDispatch();
  const handleEdit = useCallback(
    () => dispatch(Session.setEditable({ key })),
    [dispatch, key],
  );
  const handleToggleHideIndicators = useCallback(
    () => dispatch(Session.setHideIndicators({ key })),
    [dispatch, key],
  );
  const canEdit = hasUpdatePermission && editable;
  // Hide-indicators only matters outside edit mode; the toggle is irrelevant while
  // editing because indicators are forced visible.
  const showHideToggle = !canEdit;
  if (!hasUpdatePermission && !showHideToggle) return null;
  return (
    <Controls x>
      {showHideToggle && (
        <Button.Toggle
          value={hideIndicators}
          onChange={handleToggleHideIndicators}
          size="small"
          tooltipLocation={location.BOTTOM_LEFT}
          tooltip={`${hideIndicators ? "Show" : "Hide"} indicators`}
        >
          {hideIndicators ? <Icon.Visible /> : <Icon.Hidden />}
        </Button.Toggle>
      )}
      {hasUpdatePermission && (
        <Button.Toggle
          value={canEdit}
          onChange={handleEdit}
          size="small"
          tooltipLocation={location.BOTTOM_LEFT}
          tooltip={`${canEdit ? "Disable" : "Enable"} editing`}
        >
          {canEdit ? <Icon.EditOff /> : <Icon.Edit />}
        </Button.Toggle>
      )}
    </Controls>
  );
};

export const Content = Tab.createSuspended<ContentProps>(({ visible = true }) => {
  const key = Base.useKey();
  const editable = Session.useSelectEditable();
  const hideIndicators = Session.useSelectHideIndicators();
  const selected = Session.useSelectSelectedCellKeys();
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(key));
  const canEdit = hasUpdatePermission && editable;
  const dispatch = useDispatch();

  const handleSelectionChange = useCallback(
    (cells: string[]) =>
      dispatch(Session.setSelectedCells({ key, cells, anchor: cells.at(-1) ?? null })),
    [dispatch, key],
  );

  const handleEditableChange = useCallback(
    (next: boolean) => dispatch(Session.setEditable({ key, editable: next })),
    [dispatch, key],
  );

  const handleShowIndicatorsChange = useCallback(
    (next: boolean) =>
      dispatch(Session.setHideIndicators({ key, hideIndicators: !next })),
    [dispatch, key],
  );

  const handleDoubleClick = useCallback(() => {
    if (canEdit) dispatch(Nav.setBottomVisible(true));
  }, [canEdit, dispatch]);

  // When editing, indicators always show; the hideIndicators setting only takes effect
  // outside edit mode.
  const showIndicators = canEdit || !hideIndicators;

  return (
    <div className={CSS.B("table")}>
      <Base.Table
        resourceKey={key}
        selected={selected}
        onSelectionChange={handleSelectionChange}
        editable={canEdit}
        visible={visible}
        onEditableChange={hasUpdatePermission ? handleEditableChange : undefined}
        showIndicators={showIndicators}
        onShowIndicatorsChange={handleShowIndicatorsChange}
        onDoubleClick={handleDoubleClick}
        extraMenuItems={<ContextMenu.ReloadConsoleItem />}
      />
      <TableControls />
    </div>
  );
});
