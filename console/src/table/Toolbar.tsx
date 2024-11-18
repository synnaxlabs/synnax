import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Color,
  Form,
  Input,
  Select,
  Status,
  Tabs,
  Text,
  Value,
} from "@synnaxlabs/pluto";
import { deep, type KeyedNamed, scale, zodutil } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback } from "react";
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
      <Align.Space>
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

const ValueForm = () => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telem":
        return (
          <Align.Space direction="y" style={{ padding: "2rem" }}>
            <Value.TelemForm path="props" />
          </Align.Space>
        );
      case "redline":
        return (
          <Align.Space direction="y" style={{ padding: "2rem" }}>
            <RedlineForm />
          </Align.Space>
        );
      default:
        return (
          <Align.Space direction="y" grow empty style={{ padding: "2rem" }}>
            <Align.Space direction="x">
              <Form.Field<Color.Crude>
                hideIfNull
                label="Color"
                align="start"
                padHelpText={false}
                path="props.color"
              >
                {({ value, onChange, variant: _, ...props }) => (
                  <Color.Swatch
                    value={value ?? Color.ZERO.setAlpha(1).rgba255}
                    onChange={(v) => onChange(v.rgba255)}
                    {...props}
                    bordered
                  />
                )}
              </Form.Field>
              {/* <Form.TextField
                path="props.units"
                label="Units"
                align="start"
                padHelpText={false}
              /> */}
              {/* <Form.NumericField
                path="inlineSize"
                label="Value Width"
                hideIfNull
                inputProps={{
                  dragScale: { x: 1, y: 0.25 },
                  bounds: { lower: 40, upper: 500 },
                  endContent: "px",
                }}
              /> */}
              <Form.Field<Text.Level>
                path="props.level"
                label="Value Size"
                hideIfNull
                padHelpText={false}
              >
                {(p) => <Text.SelectLevel {...p} />}
              </Form.Field>
            </Align.Space>
          </Align.Space>
        );
    }
  }, []);
  const tabsProps = Tabs.useStatic({
    tabs: [
      { tabKey: "style", name: "Style" },
      { tabKey: "telem", name: "Telemetry" },
      { tabKey: "redline", name: "Redline" },
    ],
    content,
  });
  return <Tabs.Tabs size="small" {...tabsProps} />;
};

const RedlineForm = (): ReactElement => {
  const bounds = Form.useFieldValue("props.redline.bounds");
  return (
    <Align.Space direction="x" grow>
      <Form.NumericField
        inputProps={{ size: "small", showDragHandle: false }}
        style={{ width: 60 }}
        label="Lower"
        path="props.redline.bounds.lower"
      />
      <Form.Field<Color.Gradient>
        path="props.redline.gradient"
        label="Gradient"
        align="start"
        padHelpText={false}
      >
        {({ value, onChange }) => (
          <Color.GradientPicker
            value={deep.copy(value)}
            scale={scale.Scale.scale<number>(0, 1).scale(bounds)}
            onChange={(v) =>
              onChange(v.map((c) => ({ ...c, color: new Color.Color(c.color).hex })))
            }
          />
        )}
      </Form.Field>
      <Form.NumericField
        inputProps={{ size: "small", showDragHandle: false }}
        style={{ width: 60 }}
        label="Upper"
        path="props.redline.bounds.upper"
      />
    </Align.Space>
  );
};

const TextCellForm = () => (
  <Align.Space direction="x" grow style={{ padding: "2rem" }}>
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
  </Align.Space>
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
