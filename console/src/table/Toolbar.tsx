import { Icon } from "@synnaxlabs/media";
import { Align, Form, Select, Status, Text, Value } from "@synnaxlabs/pluto";
import { deep, type KeyedNamed, zodutil } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { useSelectSelectedCells } from "@/table/selectors";
import { type CellState, setCellState } from "@/table/slice";
import {
  type CellProps,
  type CellType,
  TEXT_CELL_TYPE,
  VALUE_CELL_TYPE,
  ZERO_PROPS,
  ZERO_SCHEMAS,
} from "@/table/Table";

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => {
  const selected = useSelectSelectedCells<CellType, CellProps>(layoutKey);
  const cell = selected[0];
  const dispatch = useDispatch();
  const handleChange = (values: Partial<CellState<CellType, CellProps>>) => {
    dispatch(setCellState({ key: layoutKey, state: { key: cell.key, ...values } }));
  };

  return (
    <Align.Space empty>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Table />}>Table</ToolbarTitle>
      </ToolbarHeader>
      <Align.Space style={{ padding: "2rem" }}>
        {selected.length === 0 ? (
          <EmptyContent />
        ) : (
          <CellForm key={cell.type} cell={cell} handleChange={handleChange} />
        )}
      </Align.Space>
    </Align.Space>
  );
};

interface CellFormProps {
  cell: CellState<CellType, CellProps>;
  handleChange: (state: Partial<CellState<CellType, CellProps>>) => void;
}

const CellForm = ({ cell, handleChange }: CellFormProps): ReactElement => {
  const F = FORMS[cell.type];
  const methods = Form.use({
    values: deep.copy(cell),
    onChange: ({ values }) => handleChange(values),
    sync: true,
  });

  return (
    <Form.Form {...methods}>
      <F />
    </Form.Form>
  );
};

type CellEntry = KeyedNamed<CellType> & { icon: ReactElement };

const CELL_TYPE_OPTIONS: CellEntry[] = [
  {
    key: TEXT_CELL_TYPE,
    name: "Text",
    icon: <Icon.Schematic />,
  },
  {
    key: VALUE_CELL_TYPE,
    name: "Value",
    icon: <Icon.TypeScript />,
  },
];

interface SelectCellTypeProps
  extends Omit<Select.DropdownButtonProps<CellType, CellEntry>, "data"> {}

export const SelectCellType = ({
  value,
  onChange,
  ...props
}: SelectCellTypeProps): ReactElement => (
  <Select.DropdownButton<CellType, CellEntry>
    value={value}
    onChange={onChange}
    {...props}
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
    data={CELL_TYPE_OPTIONS}
    entryRenderKey="name"
  />
);

const ValueForm = () => <Value.TelemForm path="props" />;

const TextCellForm = () => (
  <Form.Field<CellType>
    path="type"
    onChange={(value, { get, set }) => {
      const { value: prevValue } = get<CellState<CellType, CellProps>>("");
      if (prevValue.type == value) return;
      const nextProps = deep.overrideValidItems<CellProps, CellProps>(
        prevValue.props,
        ZERO_PROPS[value],
        ZERO_SCHEMAS[value],
      );
      set("props", nextProps);
    }}
  >
    {(p) => <SelectCellType {...p} />}
  </Form.Field>
);

export const EmptyContent = () => (
  <Align.Center direction="x" size="small">
    <Status.Text variant="disabled" hideIcon>
      No cell selected. Select a cell to view its properties.
    </Status.Text>
  </Align.Center>
);

const FORMS: Record<CellType, FC> = {
  [TEXT_CELL_TYPE]: TextCellForm,
  [VALUE_CELL_TYPE]: ValueForm,
};
