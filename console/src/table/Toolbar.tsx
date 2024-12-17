import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Breadcrumb,
  Form,
  Header,
  type Icon as PIcon,
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
    const theme = selectTheme(storeState);
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
  return (
    <Align.Space empty style={{ width: "100%", height: "100%" }}>
      <ToolbarHeader>
        <Align.Space direction="x" align="center">
          <Breadcrumb.Breadcrumb level="p">{breadCrumbs}</Breadcrumb.Breadcrumb>
          {isSingleCellSelected && (
            <SelectCellTypeField
              tableKey={layoutKey}
              cellKey={firstCell.key}
              onChange={(variant) => handleVariantChange(variant, firstCell.key)}
            />
          )}
        </Align.Space>
      </ToolbarHeader>
      <Align.Space style={{ width: "100%", height: "100%" }}>
        {selectedCells.length === 0 ? (
          <EmptyContent />
        ) : (
          <CellForm
            tableKey={layoutKey}
            cell={firstCell}
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

type CellEntry = KeyedNamed<TableCells.Variant> & {
  icon: ReactElement<PIcon.BaseProps>;
};

const CELL_TYPE_OPTIONS: CellEntry[] = [
  { key: TableCells.CELLS.text.key, name: "Text", icon: <Icon.Text /> },
  { key: TableCells.CELLS.value.key, name: "Value", icon: <Icon.Value /> },
];

interface SelectCellTypeFieldProps {
  tableKey: string;
  cellKey: string;
  onChange: (variant: TableCells.Variant) => void;
}

const SelectCellTypeField = ({
  tableKey,
  cellKey,
  onChange,
}: SelectCellTypeFieldProps) => {
  const cellType = useSelectCellType(tableKey, cellKey);
  return (
    <Select.DropdownButton<TableCells.Variant, CellEntry>
      value={cellType}
      onChange={onChange}
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

const EmptyContent = () => (
  <Align.Center direction="x" size="small" style={{ width: "100%", height: "100%" }}>
    <Status.Text variant="disabled" hideIcon>
      No cell selected. Select a cell to view its properties.
    </Status.Text>
  </Align.Center>
);
