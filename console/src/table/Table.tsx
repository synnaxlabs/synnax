import "@/table/Table.css";

import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Icon } from "@synnaxlabs/media";
import {
  Button,
  Color,
  Input,
  Menu,
  Table as Core,
  telem,
  Text,
  Value,
} from "@synnaxlabs/pluto";
import { bounds, box, clamp, dimensions, scale, xy } from "@synnaxlabs/x";
import { memo, type MouseEventHandler, type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { type CellState } from "@/table/migrations";
import {
  useSelectCell,
  useSelectLayout,
  useSelectSelectedColumns,
} from "@/table/selectors";
import {
  addCol,
  addRow,
  type CellLayout,
  deleteCol,
  deleteRow,
  internalCreate,
  resizeCol,
  resizeRow,
  selectCells,
  selectCol,
  type SelectionMode,
  selectRow,
  setCellState,
  type State,
  ZERO_STATE,
} from "@/table/slice";

export const LAYOUT_TYPE = "table";
export type LayoutType = typeof LAYOUT_TYPE;

export const Table: Layout.Renderer = ({ layoutKey, visible }) => {
  const layout = useSelectLayout(layoutKey);
  const dispatch = useDispatch();

  const handleAddRow = () => {
    dispatch(addRow({ key: layoutKey }));
  };

  const handleAddCol = () => {
    dispatch(addCol({ key: layoutKey }));
  };

  const contextMenu = ({ keys }: Menu.ContextMenuMenuProps) => (
    <Menu.Menu
      onChange={{
        addRowBelow: () => {
          dispatch(addRow({ key: layoutKey, loc: "bottom", cellKey: keys[0] }));
        },
        addRowAbove: () =>
          dispatch(addRow({ key: layoutKey, loc: "top", cellKey: keys[0] })),
        addColRight: () =>
          dispatch(addCol({ key: layoutKey, loc: "right", cellKey: keys[0] })),
        addColLeft: () =>
          dispatch(addCol({ key: layoutKey, loc: "left", cellKey: keys[0] })),
        deleteRow: () => dispatch(deleteRow({ key: layoutKey, cellKey: keys[0] })),
        deleteCol: () => dispatch(deleteCol({ key: layoutKey, cellKey: keys[0] })),
      }}
      iconSpacing="small"
      level="small"
    >
      <Menu.Item size="small" startIcon={<Icon.Add />} itemKey="addRowBelow">
        Add Row Below
      </Menu.Item>
      <Menu.Item size="small" startIcon={<Icon.Add />} itemKey="addRowAbove">
        Add Row Above
      </Menu.Item>
      <Menu.Divider />
      <Menu.Item size="small" startIcon={<Icon.Add />} itemKey="addColRight">
        Add Column Right
      </Menu.Item>
      <Menu.Item size="small" startIcon={<Icon.Add />} itemKey="addColLeft">
        Add Column Left
      </Menu.Item>
      <Menu.Divider />
      <Menu.Item size="small" startIcon={<Icon.Delete />} itemKey="deleteRow">
        Delete Row
      </Menu.Item>
      <Menu.Item size="small" startIcon={<Icon.Delete />} itemKey="deleteCol">
        Delete Column
      </Menu.Item>
    </Menu.Menu>
  );

  const menuProps = Menu.useContextMenu();

  const handleColResize = useCallback((size: number, index: number) => {
    dispatch(resizeCol({ key: layoutKey, index, size: clamp(size, 32) }));
  }, []);

  const windowKey = useSelectWindowKey() as string;

  const handleDoubleClick = useCallback(() => {
    dispatch(
      Layout.setNavDrawerVisible({ windowKey, key: "visualization", value: true }),
    );
  }, []);

  const colSizes = layout.columns.map((col) => col.size);

  let currPos = 3.5 * 6;
  return (
    <div className={CSS.B("table")} onDoubleClick={handleDoubleClick}>
      <Menu.ContextMenu menu={contextMenu} {...menuProps}>
        <Core.Table visible={visible}>
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
      </Menu.ContextMenu>
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
    </div>
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
    <Core.Row index={index} size={size} onResize={handleResize} onSelect={handleSelect}>
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

interface CellCProps<T extends string = string, P = unknown> {
  box: box.Box;
  state: CellState<T, P>;
  onChange: (state: Partial<CellState>) => void;
  onSelect: MouseEventHandler;
}

export const TEXT_CELL_TYPE = "text";
export type TextCellType = typeof TEXT_CELL_TYPE;
export const textCellPropsZ = z.object({
  value: z.string(),
});
export type TextCellProps = z.infer<typeof textCellPropsZ>;
export const ZERO_TEXT_CELL_PROPS: TextCellProps = { value: "" };

const TextCell = ({
  state,
  onChange,
  onSelect,
}: CellCProps<TextCellType, TextCellProps>): ReactElement => (
  <Core.Cell
    id={state.key}
    className={CSS(Menu.CONTEXT_TARGET, state.selected && Menu.CONTEXT_SELECTED)}
    selected={state.selected}
    onClick={onSelect}
    onContextMenu={onSelect}
  >
    <Input.Text
      level="p"
      variant="natural"
      value={state.props.value}
      onChange={(value) => onChange({ props: { value } })}
      selectOnFocus
      onlyChangeOnBlur
      style={{ width: "100%" }}
    />
  </Core.Cell>
);

export const VALUE_CELL_TYPE = "value";
export type ValueCellType = typeof VALUE_CELL_TYPE;
export const valueCellPropsZ = z.object({
  telem: telem.stringSourceSpecZ,
  redline: z.object({
    bounds: bounds.bounds,
    gradient: Color.gradientZ,
  }),
  level: Text.levelZ,
  color: z.string(),
  units: z.string(),
});
export type ValueCellProps = z.infer<typeof valueCellPropsZ>;
export const ZERO_VALUE_CELL_PROPS: ValueCellProps = {
  telem: telem.sourcePipeline("string", {
    connections: [
      { from: "valueStream", to: "rollingAverage" },
      { from: "rollingAverage", to: "stringifier" },
    ],
    segments: {
      valueStream: telem.streamChannelValue({ channel: 0 }),
      rollingAverage: telem.rollingAverage({ windowSize: 1 }),
      stringifier: telem.stringifyNumber({ precision: 2, notation: "standard" }),
    },
    outlet: "stringifier",
  }),
  redline: {
    bounds: { lower: 0, upper: 1 },
    gradient: [],
  },
  color: "#FFFFFF",
  level: "p",
  units: "",
};

export type ValueCellCProps = CellCProps<
  ValueCellType,
  z.infer<typeof valueCellPropsZ>
>;

export type CellType = TextCellType | ValueCellType;
export type CellProps = TextCellProps | ValueCellProps;

export const ValueCell = ({
  state: {
    key,
    props: {
      telem: t,
      level,
      color,
      redline: { gradient, bounds },
    },
    selected,
  },
  box: b,
  onSelect,
}: ValueCellCProps) => {
  const { width } = Value.use({
    aetherKey: key,
    box: b,
    telem: t,
    level,
    color,
    backgroundTelem: telem.sourcePipeline("color", {
      connections: [
        { from: "source", to: "scale" },
        { from: "scale", to: "gradient" },
      ],
      segments: {
        source: t,
        scale: telem.scaleNumber({
          scale: scale.scaleToTransform(scale.Scale.scale<number>(bounds).scale(0, 1)),
        }),
        gradient: telem.colorGradient({ gradient }),
      },
      outlet: "gradient",
    }),
  });

  return (
    <Core.Cell
      id={key}
      selected={selected}
      onClick={onSelect}
      onContextMenu={onSelect}
      style={{ height: "5rem", width }}
      className={CSS(Menu.CONTEXT_TARGET, selected && Menu.CONTEXT_SELECTED)}
    />
  );
};

const CELL_TYPES: Record<CellType, React.FC<CellCProps<any, any>>> = {
  [TEXT_CELL_TYPE]: TextCell,
  [VALUE_CELL_TYPE]: ValueCell,
};

export const ZERO_PROPS: Record<CellType, CellProps> = {
  [TEXT_CELL_TYPE]: ZERO_TEXT_CELL_PROPS,
  [VALUE_CELL_TYPE]: ZERO_VALUE_CELL_PROPS,
};

export const ZERO_SCHEMAS: Record<CellType, z.ZodType<any>> = {
  [TEXT_CELL_TYPE]: textCellPropsZ,
  [VALUE_CELL_TYPE]: valueCellPropsZ,
};

const Cell = memo(({ tableKey, cellKey, box }: CellContainerProps): ReactElement => {
  const state = useSelectCell<CellType>(tableKey, cellKey);
  const dispatch = useDispatch();
  const handleSelect: React.MouseEventHandler = ({ shiftKey, ctrlKey, metaKey }) => {
    let mode: SelectionMode = "replace";
    if (shiftKey) mode = "region";
    if (ctrlKey || metaKey) mode = "add";
    dispatch(selectCells({ key: tableKey, mode, cells: [cellKey] }));
  };
  const handleChange = (state: Partial<CellState>) =>
    dispatch(setCellState({ key: tableKey, state: { key: cellKey, ...state } }));
  const C = CELL_TYPES[state?.type ?? TEXT_CELL_TYPE];
  return <C box={box} state={state} onChange={handleChange} onSelect={handleSelect} />;
});
Cell.displayName = "Cell";

export const create =
  (initial: Partial<State> & Omit<Partial<Layout.State>, "type">): Layout.Creator =>
  ({ dispatch }) => {
    const key = initial.key ?? uuidv4();
    const { name = "Table", location = "mosaic", window, tab, ...rest } = initial;
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

export const SELECTABLE: Layout.Selectable = {
  key: LAYOUT_TYPE,
  title: "Table",
  icon: <Icon.Table />,
  create: (layoutKey: string) => create({ key: layoutKey }),
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
    <Core.ColResizer
      onSelect={handleSelect}
      selected={selectedCols}
      onResize={onResize}
      columns={columns}
    />
  );
};
