// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/Table.css";

import { type Dispatch, type PayloadAction } from "@reduxjs/toolkit";
import { table } from "@synnaxlabs/client";
import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Menu as PMenu,
  Table as Core,
  TableCells,
  Triggers,
  usePrevious,
} from "@synnaxlabs/pluto";
import {
  box,
  clamp,
  dimensions,
  location,
  type UnknownRecord,
  xy,
} from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useEffect, useRef } from "react";
import { useDispatch } from "react-redux";
import { v4 as uuid } from "uuid";

import { Menu } from "@/components";
import { CSS } from "@/css";
import { useLoadRemote } from "@/hooks/useLoadRemote";
import { Layout } from "@/layout";
import { type Selector } from "@/selector";
import {
  select,
  useSelectCell,
  useSelectEditable,
  useSelectLayout,
  useSelectSelectedColumns,
  useSelectVersion,
} from "@/table/selectors";
import {
  addCol,
  addRow,
  type CellLayout,
  clearSelected,
  copySelected,
  deleteCol,
  deleteRow,
  internalCreate,
  pasteSelected,
  resizeCol,
  resizeRow,
  selectCells,
  selectCol,
  type SelectionMode,
  selectRow,
  setCellProps,
  setEditable,
  setRemoteCreated,
  type State,
  ZERO_STATE,
} from "@/table/slice";
import { Workspace } from "@/workspace";

export const LAYOUT_TYPE = "table";
export type LayoutType = typeof LAYOUT_TYPE;

const parseContextKey = (key: string): string | number => {
  if (key.startsWith("resizer")) {
    const [, , index] = key.split("-");
    return parseInt(index);
  }
  return key;
};

const parseRowCalArgs = <L extends location.Outer | undefined>(
  tableKey: string,
  keys: string[],
  loc?: L,
): { key: string; index?: number; cellKey?: string; loc: L } => {
  const cellKey = parseContextKey(keys[0]);
  if (typeof cellKey === "number")
    return { key: tableKey, index: cellKey, loc: loc as L };
  return { key: tableKey, cellKey: keys[0], loc: loc as L };
};

interface SyncPayload {
  key?: string;
}

export const useSyncComponent = (
  layoutKey: string,
): Dispatch<PayloadAction<SyncPayload>> =>
  Workspace.useSyncComponent<SyncPayload>(
    "Table",
    layoutKey,
    async (ws, store, client) => {
      const storeState = store.getState();
      const data = select(storeState, layoutKey);
      if (data == null) return;
      const layout = Layout.selectRequired(storeState, layoutKey);
      const setData = { ...data, key: undefined };
      if (!data.remoteCreated) store.dispatch(setRemoteCreated({ key: layoutKey }));
      await client.workspaces.table.create(ws, {
        key: layoutKey,
        name: layout.name,
        data: setData,
      });
    },
  );

const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const { name } = Layout.useSelectRequired(layoutKey);
  const layout = useSelectLayout(layoutKey);
  const syncDispatch = useSyncComponent(layoutKey);
  const editable = useSelectEditable(layoutKey);

  const handleAddRow = () => {
    syncDispatch(addRow({ key: layoutKey }));
  };

  const handleAddCol = () => {
    syncDispatch(addCol({ key: layoutKey }));
  };

  const prevName = usePrevious(name);

  useEffect(() => {
    if (prevName !== name) syncDispatch(Layout.rename({ key: layoutKey, name }));
  }, [syncDispatch, name, prevName]);

  const contextMenu = ({ keys }: PMenu.ContextMenuMenuProps) => (
    <PMenu.Menu
      onChange={{
        addRowBelow: () => {
          syncDispatch(addRow(parseRowCalArgs(layoutKey, keys, "bottom")));
        },
        addRowAbove: () =>
          syncDispatch(addRow(parseRowCalArgs(layoutKey, keys, "top"))),
        addColRight: () =>
          syncDispatch(addCol(parseRowCalArgs(layoutKey, keys, "right"))),
        addColLeft: () =>
          syncDispatch(addCol(parseRowCalArgs(layoutKey, keys, "left"))),
        deleteRow: () => syncDispatch(deleteRow(parseRowCalArgs(layoutKey, keys))),
        deleteCol: () => syncDispatch(deleteCol(parseRowCalArgs(layoutKey, keys))),
        toggleEdit: () => syncDispatch(setEditable({ key: layoutKey })),
      }}
      iconSpacing="small"
      level="small"
    >
      {keys.length > 0 && (
        <>
          <PMenu.Item size="small" startIcon={<Icon.Add />} itemKey="addRowBelow">
            Add Row Below
          </PMenu.Item>
          <PMenu.Item size="small" startIcon={<Icon.Add />} itemKey="addRowAbove">
            Add Row Above
          </PMenu.Item>
          <PMenu.Divider />
          <PMenu.Item size="small" startIcon={<Icon.Add />} itemKey="addColRight">
            Add Column Right
          </PMenu.Item>
          <PMenu.Item size="small" startIcon={<Icon.Add />} itemKey="addColLeft">
            Add Column Left
          </PMenu.Item>
          <PMenu.Divider />
          <PMenu.Item size="small" startIcon={<Icon.Delete />} itemKey="deleteRow">
            Delete Row
          </PMenu.Item>
          <PMenu.Item size="small" startIcon={<Icon.Delete />} itemKey="deleteCol">
            Delete Column
          </PMenu.Item>
          <PMenu.Divider />
        </>
      )}
      <PMenu.Item
        itemKey="toggleEdit"
        startIcon={editable ? <Icon.EditOff /> : <Icon.Edit />}
      >
        {`${editable ? "Disable" : "Enable"} editing`}
      </PMenu.Item>
      <PMenu.Divider />
      <Menu.HardReloadItem />
    </PMenu.Menu>
  );

  const menuProps = PMenu.useContextMenu();

  const handleColResize = useCallback((size: number, index: number) => {
    syncDispatch(resizeCol({ key: layoutKey, index, size: clamp(size, 32) }));
  }, []);

  const windowKey = useSelectWindowKey() as string;

  const handleDoubleClick = useCallback(() => {
    if (!editable) return;
    syncDispatch(
      Layout.setNavDrawerVisible({ windowKey, key: "visualization", value: true }),
    );
  }, [editable]);

  const colSizes = layout.columns.map((col) => col.size);
  const totalColSizes = colSizes.reduce((acc, size) => acc + size, 0);
  const totalRowSizes = layout.rows.reduce((acc, row) => acc + row.size, 0);

  const ref = useRef<HTMLDivElement>(null);

  Triggers.use({
    triggers: [["Control", "V"], ["Control", "C"], ["Delete"], ["Backspace"]],
    region: ref,
    callback: useCallback(
      ({ triggers, stage }: Triggers.UseEvent) => {
        if (ref.current == null || stage !== "start") return;
        const isCopy = triggers.some((t) => t.includes("C"));
        const isDelete = triggers.some(
          (t) => t.includes("Delete") || t.includes("Backspace"),
        );
        const isPaste = triggers.some((t) => t.includes("V"));
        if (isCopy) syncDispatch(copySelected({ key: layoutKey }));
        if (isDelete) syncDispatch(clearSelected({ key: layoutKey }));
        if (isPaste) syncDispatch(pasteSelected({ key: layoutKey }));
      },
      [syncDispatch, layoutKey],
    ),
  });

  let currPos = 3.5 * 6;
  return (
    <div className={CSS.B("table")} ref={ref} onDoubleClick={handleDoubleClick}>
      <PMenu.ContextMenu menu={contextMenu} {...menuProps}>
        <Core.Table
          visible={visible}
          style={{
            width: totalColSizes,
            height: totalRowSizes,
          }}
          onContextMenu={menuProps.open}
          className={menuProps.className}
        >
          <ColResizer
            tableKey={layoutKey}
            onResize={handleColResize}
            columns={colSizes}
          />
          {layout.rows.map((row, rowIndex) => {
            const pos = currPos;
            currPos += layout.rows[rowIndex].size;
            return (
              <Row
                key={rowIndex}
                tableKey={layoutKey}
                index={rowIndex}
                cells={row.cells}
                position={pos}
                columns={colSizes}
                size={row.size}
              />
            );
          })}
        </Core.Table>
        {editable && (
          <>
            <Button.Button
              className={CSS.BE("table", "add-col")}
              justify="center"
              align="center"
              size="small"
              onClick={handleAddCol}
            >
              <Icon.Add />
            </Button.Button>
            <Button.Button
              className={CSS.BE("table", "add-row")}
              variant="filled"
              justify="center"
              align="center"
              size="small"
              onClick={handleAddRow}
            >
              <Icon.Add />
            </Button.Button>
          </>
        )}
        <TableControls tableKey={layoutKey} />
      </PMenu.ContextMenu>
    </div>
  );
};

interface TableControls {
  tableKey: string;
}

const TableControls = ({ tableKey }: TableControls) => {
  const dispatch = useDispatch();
  const editable = useSelectEditable(tableKey);
  const handleEdit = useCallback(() => {
    dispatch(setEditable({ key: tableKey }));
  }, []);

  return (
    <Align.Pack className={CSS.BE("table", "edit")}>
      <Button.ToggleIcon
        value={editable}
        onChange={handleEdit}
        size="small"
        tooltipLocation={location.BOTTOM_LEFT}
        tooltip={`${editable ? "Disable" : "Enable"} editing`}
      >
        {editable ? <Icon.EditOff /> : <Icon.Edit />}
      </Button.ToggleIcon>
    </Align.Pack>
  );
};

interface RowProps {
  tableKey: string;
  index: number;
  size: number;
  cells: CellLayout[];
  position: number;
  columns: number[];
}

const Row = ({ cells, size, columns, position, index, tableKey }: RowProps) => {
  const dispatch = useDispatch();
  const handleResize = useCallback((size: number, index: number) => {
    dispatch(resizeRow({ key: tableKey, index, size: clamp(size, 32) }));
  }, []);
  const handleSelect = useCallback(() => {
    dispatch(selectRow({ key: tableKey, index }));
  }, []);
  let currPos = 3.5 * 6;
  return (
    <Core.Row
      index={index}
      position={position}
      size={size}
      onResize={handleResize}
      onSelect={handleSelect}
    >
      {cells.map((cell, i) => {
        const pos = currPos;
        currPos += columns[i];
        return (
          <Cell
            key={cell.key}
            tableKey={tableKey}
            box={box.construct(
              xy.construct({ y: position, x: pos }),
              dimensions.construct(columns[i], size),
            )}
            cellKey={cell.key}
          />
        );
      })}
    </Core.Row>
  );
};

interface CellContainerProps {
  box: box.Box;
  tableKey: string;
  cellKey: string;
}

export type CreateArg = Partial<State> & Omit<Partial<Layout.BaseState>, "type">;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {
    const { name = "Table", location = "mosaic", window, tab, ...rest } = initial;
    const key = table.keyZ.safeParse(initial.key).data ?? uuid();
    dispatch(internalCreate({ ...ZERO_STATE, ...rest, key }));
    return {
      key,
      type: LAYOUT_TYPE,
      icon: "Table",
      name,
      location,
      window,
      tab,
    };
  };

export const SELECTABLE: Selector.Selectable = {
  key: LAYOUT_TYPE,
  title: "Table",
  icon: <Icon.Table />,
  create: async ({ layoutKey }) => create({ key: layoutKey }),
};

interface ColResizerProps {
  tableKey: string;
  columns: number[];
  onResize: (size: number, index: number) => void;
}

const ColResizer = ({ tableKey, columns, onResize }: ColResizerProps) => {
  const dispatch = useDispatch();
  const selectedCols = useSelectSelectedColumns(tableKey);
  const handleSelect = useCallback((index: number) => {
    dispatch(selectCol({ key: tableKey, index }));
  }, []);

  return (
    <Core.ColumnIndicators
      onSelect={handleSelect}
      selected={selectedCols}
      onResize={onResize}
      columns={columns}
    />
  );
};

const Cell = memo(({ tableKey, cellKey, box }: CellContainerProps): ReactElement => {
  const state = useSelectCell(tableKey, cellKey);
  const dispatch = useDispatch();
  const handleSelect = (
    cellKey: string,
    { shiftKey, ctrlKey, metaKey }: MouseEvent,
  ) => {
    let mode: SelectionMode = "replace";
    if (shiftKey) mode = "region";
    if (ctrlKey || metaKey) mode = "add";
    dispatch(selectCells({ key: tableKey, mode, cells: [cellKey] }));
  };
  const handleChange = (props: UnknownRecord) =>
    dispatch(setCellProps({ key: tableKey, cellKey, props }));
  const C = TableCells.CELLS[state.variant];
  return (
    <C.Cell
      cellKey={cellKey}
      box={box}
      onChange={handleChange}
      onSelect={handleSelect}
      selected={state.selected}
      {...state.props}
    />
  );
});
Cell.displayName = "Cell";

export const Table: Layout.Renderer = ({ layoutKey, ...rest }): ReactElement | null => {
  const table = useLoadRemote({
    name: "Table",
    targetVersion: ZERO_STATE.version,
    layoutKey,
    useSelectVersion,
    fetcher: async (client, layoutKey) => {
      const { key, data } = await client.workspaces.table.retrieve(layoutKey);
      return { key, ...data } as State;
    },
    actionCreator: internalCreate,
  });
  if (table == null) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};
