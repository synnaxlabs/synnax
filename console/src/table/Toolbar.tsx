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
  Table,
  TableCells,
  Text,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { deep, record } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { Cluster } from "@/cluster";
import { Toolbar as Core } from "@/components";
import { Export } from "@/export";
import { Layout } from "@/layout";
import { type RootState } from "@/store";
import { useExport } from "@/table/export";
import {
  selectCell,
  useSelectSelectedCellPos,
  useSelectSelectedCells,
} from "@/table/selectors";
import { type CellState, setCellProps, setCellType } from "@/table/slice";
import { useSyncComponent } from "@/table/Table";

export interface ToolbarProps {
  layoutKey: string;
}

const TOOLBAR_BUTTONS_STYLE = { width: 66 };

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const selectedCells = useSelectSelectedCells(layoutKey);
  const selectedCellMeta = useSelectSelectedCellPos(layoutKey);

  const firstCell = selectedCells[0];
  const dispatch = useDispatch();
  const store = useStore<RootState>();
  const handleVariantChange = (variant: TableCells.Variant, cellKey: string): void => {
    const storeState = store.getState();
    const theme = Layout.selectTheme(storeState);
    const cellState = selectCell(storeState, layoutKey, cellKey);
    if (variant === cellState.variant) return;
    if (theme == null) throw new Error("Theme is null");
    const spec = TableCells.CELLS[variant];
    const nextProps = deep.overrideValidItems(
      cellState.props,
      spec.defaultProps(theme),
      spec.schema,
    );
    dispatch(setCellType({ key: layoutKey, cellKey, variant, nextProps }));
  };
  const handleExport = useExport();
  return (
    <Core.Content>
      <Core.Header>
        <Flex.Box x align="center">
          <Breadcrumb.Breadcrumb>
            <Breadcrumb.Segment weight={500} color={9} level="h5">
              <Icon.Table />
              {name}
            </Breadcrumb.Segment>
            {selectedCellMeta != null && (
              <Breadcrumb.Segment color={8}>
                {Table.getCellColumn(selectedCellMeta.x)}
                {selectedCellMeta.y + 1}
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
      </Core.Header>
      <Flex.Box full>
        {selectedCells.length === 0 ? (
          <EmptyContent />
        ) : (
          <CellForm
            tableKey={layoutKey}
            cell={firstCell}
            // trigger re-render if you select a different cell
            key={firstCell.key}
            onVariantChange={(variant) => handleVariantChange(variant, firstCell.key)}
          />
        )}
      </Flex.Box>
    </Core.Content>
  );
};

interface CellFormProps {
  tableKey: string;
  cell: CellState;
  onVariantChange: (variant: TableCells.Variant) => void;
}

const CellForm = ({ tableKey, cell, onVariantChange }: CellFormProps): ReactElement => {
  const tableRef = useSyncedRef(tableKey);
  const cellRef = useSyncedRef(cell?.key);
  const dispatchSync = useSyncComponent(tableKey);
  const handleChange = useCallback(
    ({ values }: Form.OnChangeArgs<ReturnType<typeof record.unknownZ>>) => {
      dispatchSync(
        setCellProps({
          key: tableRef.current,
          cellKey: cellRef.current,
          props: deep.copy(values),
        }),
      );
    },
    [],
  );
  const methods = Form.use<ReturnType<typeof record.unknownZ>>({
    values: deep.copy(cell.props),
    schema: record.unknownZ(),
    onChange: handleChange,
    sync: true,
  });
  const F = TableCells.CELLS[cell.variant].Form;
  return (
    <Form.Form<ReturnType<typeof record.unknownZ>> {...methods}>
      <F onVariantChange={onVariantChange} />
    </Form.Form>
  );
};

const EmptyContent = () => (
  <Text.Text status="disabled" center>
    No cell selected. Select a cell to view its properties.
  </Text.Text>
);
