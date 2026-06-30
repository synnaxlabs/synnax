// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/table/Table.css";

import { table } from "@synnaxlabs/client";
import { Access, Button, Icon, Table as Base } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu, Controls } from "@/component";
import { CSS } from "@/component/css";
import { Session } from "@/session";
import { Layout } from "@/layout";

const Internal: Layout.Renderer = ({ visible }) => {
  const key = Base.useKey();
  const editable = Session.Table.useSelectEditable();
  const hideIndicators = Session.Table.useSelectHideIndicators();
  const selected = Session.Table.useSelectSelectedCellKeys();
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(key));
  const canEdit = hasUpdatePermission && editable;
  const dispatch = Session.useDispatch();

  const handleSelectionChange = useCallback(
    (cells: string[]) =>
      dispatch(
        Session.Table.setSelectedCells({ key, cells, anchor: cells.at(-1) ?? null }),
      ),
    [dispatch, key],
  );

  const handleEditableChange = useCallback(
    (next: boolean) => dispatch(Session.Table.setEditable({ key, editable: next })),
    [dispatch, key],
  );

  const handleShowIndicatorsChange = useCallback(
    (next: boolean) =>
      dispatch(Session.Table.setHideIndicators({ key, hideIndicators: !next })),
    [dispatch, key],
  );

  const handleDoubleClick = useCallback(() => {
    if (canEdit) dispatch(Session.Nav.showBottom({}));
  }, [canEdit, dispatch]);

  // When editing, indicators always show; the hideIndicators setting only
  // takes effect outside edit mode.
  const showIndicators = canEdit || !hideIndicators;

  return (
    <div className={CSS.B("table")}>
      <Base.Table
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
};

const TableControls = (): ReactElement | null => {
  const key = Base.useKey();
  const editable = Session.Table.useSelectEditable();
  const hideIndicators = Session.Table.useSelectHideIndicators();
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(key));
  const dispatch = Session.useDispatch();
  const handleEdit = useCallback(
    () => dispatch(Session.Table.setEditable({ key })),
    [dispatch, key],
  );
  const handleToggleHideIndicators = useCallback(
    () => dispatch(Session.Table.setHideIndicators({ key })),
    [dispatch, key],
  );
  const canEdit = hasUpdatePermission && editable;
  // Hide-indicators only matters outside edit mode; the toggle is irrelevant
  // while editing because indicators are forced visible.
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

export const Table: Layout.Renderer = (props) => (
  <Base.Suspended tableKey={props.layoutKey}>
    <Internal {...props} />
  </Base.Suspended>
);
Table.useName = Layout.createUseFluxName(
  Base.useRename,
  Base.useRetrieveObservableName,
);
