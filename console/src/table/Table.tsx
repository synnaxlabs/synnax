import "@/table/Table.css";

import { Icon } from "@synnaxlabs/media";
import {
  Aether,
  Button,
  Canvas,
  Input,
  Menu,
  Table as Core,
  telem,
  Value,
} from "@synnaxlabs/pluto";
import { box, xy } from "@synnaxlabs/x";
import { memo, type MouseEventHandler, type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";
import { v4 as uuidv4 } from "uuid";
import { z } from "zod";

import { CSS } from "@/css";
import { type Layout } from "@/layout";
import { type CellState } from "@/table/migrations";
import { useSelectCell, useSelectLayout } from "@/table/selectors";
import {
  addCol,
  addRow,
  internalCreate,
  selectCells,
  type SelectionMode,
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

  const contextMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    console.log(keys);
    return (
      <Menu.Menu iconSpacing="small" level="small">
        <Menu.Item size="small" itemKey="addRowBelow">
          Add Row Below
        </Menu.Item>
        <Menu.Item size="small" itemKey="addRowAbove">
          Add Row Above
        </Menu.Item>
        <Menu.Item size="small" itemKey="addColRight">
          Add Column Right
        </Menu.Item>
        <Menu.Item size="small" itemKey="addColLeft">
          Add Column Left
        </Menu.Item>
      </Menu.Menu>
    );
  };

  const menuProps = Menu.useContextMenu();

  return (
    <div className={CSS.B("table")}>
      <Menu.ContextMenu menu={contextMenu} {...menuProps}>
        <Core.Table visible={visible}>
          {layout.rows.map((row, rowIndex) => (
            <Core.Row key={rowIndex}>
              {row.cells.map(({ key }) => (
                <Cell key={key} tableKey={layoutKey} cellKey={key} />
              ))}
            </Core.Row>
          ))}
        </Core.Table>
      </Menu.ContextMenu>
      <Button.Button
        className={CSS.BE("table", "add-col")}
        justify="center"
        align="center"
        onClick={handleAddCol}
      >
        <Icon.Add />
      </Button.Button>
      <Button.Button
        className={CSS.BE("table", "add-row")}
        justify="center"
        align="center"
        onClick={handleAddRow}
      >
        <Icon.Add />
      </Button.Button>
    </div>
  );
};

interface CellContainerProps {
  tableKey: string;
  cellKey: string;
}

interface CellCProps<T extends string = string, P = unknown> {
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
      value={state.props.value ?? "Hello"}
      onChange={(value) => onChange({ props: { value } })}
      selectOnFocus
      onlyChangeOnBlur
      style={{ width: 100 }}
    />
  </Core.Cell>
);

export const VALUE_CELL_TYPE = "value";
export type ValueCellType = typeof VALUE_CELL_TYPE;
export const valueCellPropsZ = z.object({
  telem: telem.stringSourceSpecZ,
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
    props: { telem },
    selected,
  },
  onSelect,
}: ValueCellCProps) => {
  const [b, setB] = useState<box.Box>(box.ZERO);
  const { width } = Value.use({ aetherKey: key, box: b, telem });
  const ref = Canvas.useRegion((cellB, el) => {
    // get the nearest parent element by className
    const parentEl = el.closest(".pluto-table");
    const parentB = box.construct(parentEl);
    setB(
      box.construct(
        xy.translate(box.topLeft(cellB), xy.scale(box.topLeft(parentB), -1)),
        box.dims(cellB),
      ),
    );
  });
  return (
    <Core.Cell
      id={key}
      ref={ref}
      selected={selected}
      onClick={onSelect}
      onContextMenu={onSelect}
      style={{ height: "5rem", width }}
      className={CSS(Menu.CONTEXT_TARGET, selected && Menu.CONTEXT_SELECTED)}
    >
      <div style={{ height: "100%", width: "100%" }} />
    </Core.Cell>
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

const Cell = memo(({ tableKey, cellKey }: CellContainerProps): ReactElement => {
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
  return <C state={state} onChange={handleChange} onSelect={handleSelect} />;
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
