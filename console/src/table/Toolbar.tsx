// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import {
  Align,
  Breadcrumb,
  Form,
  Icon,
  Status,
  Table,
  TableCells,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
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

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const breadCrumbs: Breadcrumb.Segments = [
    { label: name, icon: <Icon.Table />, level: "h5", weight: 500, shade: 8 },
  ];
  const selectedCells = useSelectSelectedCells(layoutKey);
  const selectedCellMeta = useSelectSelectedCellPos(layoutKey);
  if (selectedCellMeta != null)
    breadCrumbs.push({
      label: `${Table.getCellColumn(selectedCellMeta.x)}${selectedCellMeta.y + 1}`,
      level: "p",
      weight: 400,
      shade: 7,
    });
  const isSingleCellSelected = selectedCells.length === 1;
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
    <Align.Space empty style={{ width: "100%", height: "100%" }}>
      <Core.Header>
        <Align.Space x align="center">
          <Breadcrumb.Breadcrumb level="p">{breadCrumbs}</Breadcrumb.Breadcrumb>
          {isSingleCellSelected && (
            <TableCells.SelectVariant
              allowNone={false}
              value={firstCell.variant}
              onChange={(variant: TableCells.Variant) =>
                handleVariantChange(variant, firstCell.key)
              }
            />
          )}
        </Align.Space>
        <Align.Space x style={{ width: 66 }} empty>
          <Export.ToolbarButton onExport={() => handleExport(layoutKey)} />
          <Cluster.CopyLinkToolbarButton
            name={name}
            ontologyID={table.ontologyID(layoutKey)}
          />
        </Align.Space>
      </Core.Header>
      <Align.Space style={{ width: "100%", height: "100%" }}>
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
      </Align.Space>
    </Align.Space>
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
  const handleChange = useCallback(({ values }: Form.OnChangeProps<any>) => {
    dispatchSync(
      setCellProps({ key: tableRef.current, cellKey: cellRef.current, props: values }),
    );
  }, []);
  const methods = Form.use({
    values: deep.copy(cell.props),
    onChange: handleChange,
    sync: true,
  });
  const F = TableCells.CELLS[cell.variant].Form;
  return (
    <Form.Form {...methods}>
      <F onVariantChange={onVariantChange} />
    </Form.Form>
  );
};

const EmptyContent = () => (
  <Align.Center x gap="small" style={{ width: "100%", height: "100%" }}>
    <Status.Text variant="disabled" hideIcon>
      No cell selected. Select a cell to view its properties.
    </Status.Text>
  </Align.Center>
);
