// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import {
  Breadcrumb,
  Flex,
  Form,
  Icon,
  Table as Base,
  TableCells,
  Text,
} from "@synnaxlabs/pluto";
import { deep, record } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useMemo } from "react";
import { useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { Toolbar as Tb } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { type RootState } from "@/store";
import { useExport } from "@/table/export";
import { useSelectPendingUpload, useSelectSelectedCellKeys } from "@/table/selectors";

export interface ToolbarProps {
  layoutKey: string;
}

const TOOLBAR_BUTTONS_STYLE = { width: 66 };

const cellPositionInRows = (
  rows: table.Row[],
  cellKey: string,
): { x: number; y: number } | null => {
  for (let y = 0; y < rows.length; y++) {
    const x = rows[y].cells.indexOf(cellKey);
    if (x !== -1) return { x, y };
  }
  return null;
};

const Internal = ({ layoutKey }: ToolbarProps): ReactElement => {
  Base.useEnsureRetrieved({ key: layoutKey });
  const { name } = Layout.useSelectRequired(layoutKey);
  const selectedCellKeys = useSelectSelectedCellKeys(layoutKey);
  const rows = Base.useSelectRows({ key: layoutKey });
  const firstSelectedKey = selectedCellKeys[0];
  const selectedCellPos = useMemo(
    () =>
      firstSelectedKey != null ? cellPositionInRows(rows, firstSelectedKey) : null,
    [firstSelectedKey, rows],
  );
  const handleExport = useExport();
  return (
    <Tb.Content>
      <Tb.Header>
        <Flex.Box x align="center">
          <Breadcrumb.Breadcrumb>
            <Breadcrumb.Segment weight={500} color={9} level="h5">
              <Icon.Table />
              {name}
            </Breadcrumb.Segment>
            {selectedCellPos != null && (
              <Breadcrumb.Segment color={8}>
                {Base.getCellColumn(selectedCellPos.x)}
                {selectedCellPos.y + 1}
              </Breadcrumb.Segment>
            )}
          </Breadcrumb.Breadcrumb>
        </Flex.Box>
        <Flex.Box x style={TOOLBAR_BUTTONS_STYLE} empty>
          <Export.ToolbarButton onExport={() => handleExport(layoutKey)} />
          <Cluster.CopyLinkToolbarButton
            name={name}
            ontologyID={table.ontologyID(layoutKey)}
          />
        </Flex.Box>
      </Tb.Header>
      <Flex.Box full>
        {firstSelectedKey == null ? (
          <EmptyContent />
        ) : (
          <CellForm
            key={firstSelectedKey}
            tableKey={layoutKey}
            cellKey={firstSelectedKey}
          />
        )}
      </Flex.Box>
    </Tb.Content>
  );
};

export const Toolbar = (props: ToolbarProps): ReactElement | null => {
  const pendingUpload = useSelectPendingUpload(props.layoutKey);
  return pendingUpload == null ? <Internal {...props} /> : null;
};

interface CellFormProps {
  tableKey: string;
  cellKey: string;
}

const CellForm = ({ tableKey, cellKey }: CellFormProps): ReactElement | null => {
  const cell = Base.useSelectCell({ key: tableKey, cellKey });
  const { dispatch } = Base.useDispatch();
  const store = useStore<RootState>();

  const handleVariantChange = useCallback(
    (variant: TableCells.Variant) => {
      if (cell == null || variant === cell.variant) return;
      const theme = Layout.selectTheme(store.getState());
      if (theme == null) throw new Error("Theme is null");
      const spec = TableCells.CELLS[variant];
      const nextProps = deep.overrideValidItems(
        cell.props,
        spec.defaultProps(theme),
        spec.schema,
      );
      dispatch({
        key: tableKey,
        actions: [table.setCell({ cell: { key: cellKey, variant, props: nextProps } })],
      });
    },
    [cell, cellKey, dispatch, store, tableKey],
  );

  const handleChange = useCallback(
    ({ values }: Form.OnChangeArgs<ReturnType<typeof record.unknownZ>>) => {
      if (cell == null) return;
      dispatch({
        key: tableKey,
        actions: [
          table.setCell({
            cell: { key: cellKey, variant: cell.variant, props: deep.copy(values) },
          }),
        ],
      });
    },
    [cell, cellKey, dispatch, tableKey],
  );

  const methods = Form.use<ReturnType<typeof record.unknownZ>>({
    values: cell != null ? deep.copy(cell.props) : {},
    schema: record.unknownZ(),
    onChange: handleChange,
    sync: true,
  });

  if (cell == null) return null;
  const F = TableCells.CELLS[cell.variant as keyof typeof TableCells.CELLS]?.Form;
  if (F == null) return null;
  return (
    <Form.Form<ReturnType<typeof record.unknownZ>> {...methods}>
      <F onVariantChange={handleVariantChange} />
    </Form.Form>
  );
};

const EmptyContent = (): ReactElement => (
  <Text.Text status="disabled" center>
    No cell selected. Select a cell to view its properties.
  </Text.Text>
);
