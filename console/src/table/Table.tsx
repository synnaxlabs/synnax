import "@/table/Table.css";

import { Icon } from "@synnaxlabs/media";
import { Align, Button, Input, Table as Core, Text } from "@synnaxlabs/pluto";
import { memo, type ReactElement } from "react";
import { useDispatch } from "react-redux";
import { v4 as uuidv4 } from "uuid";

import { CSS } from "@/css";
import { type Layout } from "@/layout";
import { type CellState } from "@/table/migrations";
import { useSelect, useSelectCell, useSelectLayout } from "@/table/selectors";
import {
  addCol,
  addRow,
  internalCreate,
  selectCells,
  setCellState,
  type State,
  ZERO_STATE,
} from "@/table/slice";

export const LAYOUT_TYPE = "table";
export type LayoutType = typeof LAYOUT_TYPE;

export const Table: Layout.Renderer = ({ layoutKey }) => {
  const layout = useSelectLayout(layoutKey);
  const dispatch = useDispatch();

  const handleAddRow = () => {
    dispatch(addRow({ key: layoutKey }));
  };

  const handleAddCol = () => {
    dispatch(addCol({ key: layoutKey }));
  };

  console.log("RENDER TABLE");

  return (
    <div className={CSS.B("table")}>
      <Core.Table>
        {layout.rows.map((row, rowIndex) => (
          <Core.Row key={rowIndex}>
            {row.cells.map(({ key }) => (
              <CellContainer key={key} tableKey={layoutKey} cellKey={key} />
            ))}
          </Core.Row>
        ))}
      </Core.Table>
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

interface CellProps {
  onChange: (state: Partial<CellState>) => void;
  state: CellState;
}

const TextCell = ({ state, onChange }: CellProps): ReactElement => (
  <Input.Text
    level="p"
    variant="natural"
    value={state.props.value ?? "Hello"}
    onChange={(value) => onChange({ props: { value }, selected: false })}
    selectOnFocus
    onlyChangeOnBlur
    style={{ width: 100 }}
  />
);

const CELL_TYPES: Record<string, React.FC<CellProps>> = {
  text: TextCell,
};

export const CellContainer = memo(({ tableKey, cellKey }: CellContainerProps) => {
  const state = useSelectCell(tableKey, cellKey);
  const dispatch = useDispatch();

  console.log("RENDER CELL", cellKey);

  const handleClick: React.MouseEventHandler = ({ shiftKey, ctrlKey, metaKey }) => {
    let mode = "replace";
    if (shiftKey) mode = "region";
    if (ctrlKey || metaKey) mode = "add";
    dispatch(
      selectCells({
        key: tableKey,
        mode,
        cells: [cellKey],
      }),
    );
  };

  const C = CELL_TYPES[state?.type ?? "text"];

  return (
    <Core.Cell selected={state?.selected} onClick={handleClick}>
      <C
        state={state}
        onChange={(state) =>
          dispatch(
            setCellState({
              key: tableKey,
              state: {
                key: cellKey,
                ...state,
              },
            }),
          )
        }
      />
    </Core.Cell>
  );
});
CellContainer.displayName = "Cell";

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
