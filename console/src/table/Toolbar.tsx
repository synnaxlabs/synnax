import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Breadcrumb,
  Form,
  Select,
  Status,
  Table,
  TableCells,
  Text,
  useSyncedRef,
} from "@synnaxlabs/pluto";
import { deep, type KeyedNamed } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch, useStore } from "react-redux";

import { ToolbarHeader } from "@/components";
import { Layout } from "@/layout";
import { selectTheme } from "@/layout/selectors";
import { type RootState } from "@/store";
import {
  selectCell,
  useSelectCellType,
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
    {
      label: name,
      icon: <Icon.Table />,
      level: "h5",
      weight: 500,
      shade: 8,
    },
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

  return (
    <Align.Space empty>
      <ToolbarHeader>
        <Align.Space direction="x" align="center">
          <Breadcrumb.Breadcrumb level="p">{breadCrumbs}</Breadcrumb.Breadcrumb>
          {isSingleCellSelected && (
            <SelectCellTypeField tableKey={layoutKey} cellKey={firstCell.key} />
          )}
        </Align.Space>
      </ToolbarHeader>
      <Align.Space>
        {selectedCells.length === 0 ? (
          <EmptyContent />
        ) : (
          <CellForm tableKey={layoutKey} cell={firstCell} />
        )}
      </Align.Space>
    </Align.Space>
  );
};

interface CellFormProps {
  tableKey: string;
  cell: CellState;
}

const CellForm = ({ tableKey, cell }: CellFormProps): ReactElement => {
  const tableRef = useSyncedRef(tableKey);
  const cellRef = useSyncedRef(cell?.key);

  const d = useSyncComponent(tableKey);

  const handleChange = useCallback(({ values }: Form.OnChangeProps<any>) => {
    d(setCellProps({ key: tableRef.current, cellKey: cellRef.current, props: values }));
  }, []);

  const methods = Form.use({
    values: deep.copy(cell.props),
    onChange: handleChange,
    sync: true,
  });

  const F = TableCells.CELLS[cell.variant].Form;
  return (
    <Form.Form {...methods}>
      <F />
    </Form.Form>
  );
};

type CellEntry = KeyedNamed<TableCells.Variant> & { icon: ReactElement };

const CELL_TYPE_OPTIONS: CellEntry[] = [
  {
    key: TableCells.CELLS.text.key,
    name: "Text",
    icon: <Icon.Text />,
  },
  {
    key: TableCells.CELLS.value.key,
    name: "Value",
    icon: <Icon.Value />,
  },
];

interface SelectCellTypeFieldProps {
  tableKey: string;
  cellKey: string;
}

export const SelectCellTypeField = ({
  tableKey,
  cellKey,
}: SelectCellTypeFieldProps) => {
  const cellType = useSelectCellType(tableKey, cellKey);
  const store = useStore<RootState>();
  const dispatch = useDispatch();
  const propsRef = useSyncedRef({ tableKey, cellKey });
  const handleChange = useCallback((variant: TableCells.Variant) => {
    const { tableKey, cellKey } = propsRef.current;
    const storeState = store.getState();
    const theme = selectTheme(storeState);
    const cellState = selectCell(storeState, tableKey, cellKey);
    if (variant === cellState.variant || theme == null) return;
    const spec = TableCells.CELLS[variant];
    const nextProps = deep.overrideValidItems(
      cellState.props,
      spec.defaultProps(theme),
      spec.schema,
    );
    dispatch(setCellType({ key: tableKey, cellKey, variant, nextProps }));
  }, []);
  return (
    <Select.DropdownButton<TableCells.Variant, CellEntry>
      value={cellType}
      onChange={handleChange}
      columns={[
        {
          key: "name",
          name: "Name",
          render: ({ entry }) => (
            <Text.WithIcon level="p" startIcon={entry.icon}>
              {entry.name}
            </Text.WithIcon>
          ),
        },
      ]}
      dropdownVariant="floating"
      data={CELL_TYPE_OPTIONS}
      entryRenderKey="name"
    >
      {({ selected, ...p }) => (
        <Select.BaseButton
          style={{ width: 80 }}
          variant="text"
          size="small"
          selected={selected}
          {...p}
          startIcon={selected?.icon}
        >
          {selected?.name}
        </Select.BaseButton>
      )}
    </Select.DropdownButton>
  );
};

export const EmptyContent = () => (
  <Align.Center direction="x" size="small">
    <Status.Text variant="disabled" hideIcon>
      No cell selected. Select a cell to view its properties.
    </Status.Text>
  </Align.Center>
);
