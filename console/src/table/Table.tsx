import { Icon } from "@synnaxlabs/media";
import { Table as Core, Text } from "@synnaxlabs/pluto";
import { memo } from "react";
import { v4 as uuidv4 } from "uuid";

import { Layout } from "@/layout";
import { useSelect, useSelectCell } from "@/table/selectors";
import { internalCreate, State, ZERO_STATE } from "@/table/slice";

export const LAYOUT_TYPE = "table";
export type LayoutType = typeof LAYOUT_TYPE;

export const Table: Layout.Renderer = ({ layoutKey }) => {
  const { layout } = useSelect(layoutKey);
  return (
    <Core.Table>
      {layout.rows.map((row, rowIndex) => (
        <Core.Row key={rowIndex}>
          {row.cells.map(({ key }, cellIndex) => (
            <Cell key={key} tableKey={layoutKey} cellKey={key} />
          ))}
        </Core.Row>
      ))}
    </Core.Table>
  );
};

interface CellProps {
  tableKey: string;
  cellKey: string;
}

export const Cell = memo(({ tableKey, cellKey }: CellProps) => {
  const state = useSelectCell(tableKey, cellKey);
  const dipatch = useDispatch();

  return (
    <Core.Cell
      selected={state?.selected}
      onClick={() => {
        //
      }}
    >
      Dog
    </Core.Cell>
  );
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
