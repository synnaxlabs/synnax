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
import { createEnsureState } from "@/hooks/useEnsureState";
import { Layout } from "@/layout";
import {
  useSelectEditable,
  useSelectExists,
  useSelectHideIndicators,
  useSelectPendingUpload,
  useSelectSelectedCellKeys,
} from "@/table/selectors";
import {
  clearPendingUpload,
  internalCreate,
  setEditable,
  setHideIndicators,
  setSelectedCells,
} from "@/table/slice";
import { createUseAutoUpload } from "@/vis/useAutoUpload";

export { create, LAYOUT_TYPE, type LayoutType } from "@/table/layout";

const useAutoUpload = createUseAutoUpload({
  useSelectPendingUpload,
  toCreateParams: (pending, fields) => ({ ...pending, ...fields }),
  useCreate: Base.useCreate,
  clearPendingUpload,
});

const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const editable = useSelectEditable(layoutKey);
  const hideIndicators = useSelectHideIndicators(layoutKey);
  const selected = useSelectSelectedCellKeys(layoutKey);
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(layoutKey));
  const canEdit = hasUpdatePermission && editable;
  const dispatch = useDispatch();

  const handleSelectionChange = useCallback(
    (cells: string[]) =>
      dispatch(
        setSelectedCells({ key: layoutKey, cells, anchor: cells.at(-1) ?? null }),
      ),
    [dispatch, layoutKey],
  );

  const handleEditableChange = useCallback(
    (next: boolean) => dispatch(setEditable({ key: layoutKey, editable: next })),
    [dispatch, layoutKey],
  );

  const handleShowIndicatorsChange = useCallback(
    (next: boolean) =>
      dispatch(setHideIndicators({ key: layoutKey, hideIndicators: !next })),
    [dispatch, layoutKey],
  );

  const handleDoubleClick = useCallback(() => {
    if (canEdit)
      dispatch(Layout.setNavDrawerVisible({ key: "visualization", value: true }));
  }, [canEdit, dispatch]);

  // When editing, indicators always show; the hideIndicators setting only
  // takes effect outside edit mode.
  const showIndicators = canEdit || !hideIndicators;

  return (
    <div className={CSS.B("table")}>
      <Base.Table
        resourceKey={layoutKey}
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
      <TableControls tableKey={layoutKey} />
    </div>
  );
};

interface TableControlsProps {
  tableKey: string;
}

const TableControls = ({ tableKey }: TableControlsProps): ReactElement | null => {
  const editable = useSelectEditable(tableKey);
  const hideIndicators = useSelectHideIndicators(tableKey);
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(tableKey));
  const dispatch = useDispatch();
  const handleEdit = useCallback(
    () => dispatch(setEditable({ key: tableKey })),
    [dispatch, tableKey],
  );
  const handleToggleHideIndicators = useCallback(
    () => dispatch(setHideIndicators({ key: tableKey })),
    [dispatch, tableKey],
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

const useEnsureState = createEnsureState({
  useExists: useSelectExists,
  create: (key) => internalCreate({ key }),
});

export const Table: Layout.Renderer = (props) => {
  const exists = useEnsureState(props.layoutKey);
  const uploaded = useAutoUpload(props.layoutKey);
  if (!exists || !uploaded) return null;
  return <Loaded {...props} />;
};

Table.useName = Layout.createUseFluxName(
  Base.useRename,
  Base.useRetrieveObservableName,
);
Table.icon = <Icon.Table />;
