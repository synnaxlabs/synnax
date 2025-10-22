// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  Color,
  Divider,
  Flex,
  Icon,
  Input,
  List as PList,
  Menu as PMenu,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { bounds, color, id } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { EmptyAction, Menu } from "@/components";
import { Layout } from "@/layout";
import { type AxisKey, Y1, Y2 } from "@/lineplot/axis";
import { useSelectAxes, useSelectRule, useSelectRules } from "@/lineplot/selectors";
import { removeRule, type RuleState, setRule, setSelectedRule } from "@/lineplot/slice";

interface EmptyContentProps {
  onCreateRule: () => void;
}

const EmptyContent = ({ onCreateRule }: EmptyContentProps): ReactElement => (
  <EmptyAction
    x
    message="No annotations added."
    action="Create an annotation"
    onClick={onCreateRule}
  />
);

interface ListItemProps extends PList.ItemProps<string> {
  layoutKey: string;
  onChangeLabel: (label: string) => void;
}

const ListItem = ({
  layoutKey,
  onChangeLabel,
  ...rest
}: ListItemProps): ReactElement | null => {
  const { itemKey } = rest;
  const entry = useSelectRule(layoutKey, itemKey);
  if (entry == null) return null;
  const { label } = entry;
  return (
    <Select.ListItem
      {...rest}
      style={{ padding: "0.5rem 1.5rem" }}
      align="center"
      full="x"
      square={false}
    >
      <Text.Editable
        value={label}
        overflow="ellipsis"
        color={10}
        weight={500}
        onChange={onChangeLabel}
      />
    </Select.ListItem>
  );
};

interface ListProps {
  rules: RuleState[];
  selected: string[];
  layoutKey: string;
  onChange: (keys: string[]) => void;
  onCreate: () => void;
  onRemoveAnnotations: (keys: string[]) => void;
  onLabelChange: (label: string, key: string) => void;
}

const List = ({
  selected,
  onChange,
  rules,
  onCreate,
  onRemoveAnnotations,
  layoutKey,
  onLabelChange,
}: ListProps): ReactElement => {
  const menuProps = PMenu.useContextMenu();
  const { data } = PList.useStaticData<string, RuleState>({ data: rules });
  return (
    <Flex.Box x pack style={{ width: "20%" }} align="start">
      <Flex.Box style={{ padding: "0.5rem" }}>
        <Button.Button tooltip="Add Rule" onClick={onCreate} size="small">
          <Icon.Add />
        </Button.Button>
      </Flex.Box>
      <Divider.Divider y />
      <Select.Frame<string, RuleState>
        multiple
        data={data}
        value={selected}
        onChange={onChange}
        replaceOnSingle
        allowNone={false}
      >
        <PMenu.ContextMenu
          menu={({ keys }) => (
            <PMenu.Menu
              onChange={{ remove: () => onRemoveAnnotations(keys) }}
              level="small"
            >
              <PMenu.Item itemKey="remove" size="small">
                <Icon.Delete />
                Delete
              </PMenu.Item>
              <Divider.Divider x />
              <Menu.ReloadConsoleItem />
            </PMenu.Menu>
          )}
          {...menuProps}
        >
          <PList.Items<string, RuleState> onContextMenu={menuProps.open} grow>
            {({ key, ...rest }) => (
              <ListItem
                layoutKey={layoutKey}
                key={key}
                {...rest}
                onChangeLabel={(v) => onLabelChange(v, key)}
              />
            )}
          </PList.Items>
        </PMenu.ContextMenu>
      </Select.Frame>
    </Flex.Box>
  );
};

const AXIS_DATA: AxisKey[] = [Y1, Y2];

const SelectAxis = (
  props: Omit<Select.ButtonsProps<AxisKey>, "keys">,
): ReactElement => (
  <Select.Buttons {...props} keys={AXIS_DATA}>
    <Select.Button itemKey={Y1}>Y1</Select.Button>
    <Select.Button itemKey={Y2}>Y2</Select.Button>
  </Select.Buttons>
);

interface RuleContentProps {
  rule: RuleState;
  onChangeLabel: (label: string) => void;
  onChangeUnits: (units: string) => void;
  onChangePosition: (position: number) => void;
  onChangeColor: (color: color.Color) => void;
  onChangeAxis: (axis: AxisKey) => void;
  onChangeLineWidth: (lineWidth: number) => void;
  onChangeLineDash: (lineDash: number) => void;
}

const RuleContent = ({
  rule: { label, units, position, color, axis, lineWidth, lineDash },
  onChangeLabel,
  onChangeUnits,
  onChangePosition,
  onChangeColor,
  onChangeAxis,
  onChangeLineWidth,
  onChangeLineDash,
}: RuleContentProps): ReactElement => (
  <Flex.Box y grow style={{ padding: "1.5rem 2rem" }}>
    <Flex.Box x wrap>
      <Input.Item label="Label" grow>
        <Input.Text onChange={onChangeLabel} value={label} />
      </Input.Item>
      <Input.Item label="Units">
        <Input.Text onChange={onChangeUnits} value={units} style={{ width: "15rem" }} />
      </Input.Item>
      <Input.Item label="Position">
        <Input.Numeric
          onChange={onChangePosition}
          value={Number(position.toFixed(2))}
          style={{ width: "25rem" }}
        />
      </Input.Item>
      <Input.Item label="Axis">
        <SelectAxis value={axis} onChange={onChangeAxis} />
      </Input.Item>
    </Flex.Box>
    <Flex.Box x wrap>
      <Input.Item label="Color">
        <Color.Swatch value={color} onChange={onChangeColor} />
      </Input.Item>
      <Input.Item label="Line Width">
        <Input.Numeric
          bounds={{ lower: 1, upper: 10 }}
          onChange={onChangeLineWidth}
          value={lineWidth}
        />
      </Input.Item>
      <Input.Item label="Line Dash">
        <Input.Numeric
          bounds={{ lower: 0, upper: 50 }}
          onChange={onChangeLineDash}
          value={lineDash}
        />
      </Input.Item>
    </Flex.Box>
  </Flex.Box>
);

export interface AnnotationsProps {
  linePlotKey: string;
}

export const Annotations = ({ linePlotKey }: AnnotationsProps): ReactElement => {
  const axes = useSelectAxes(linePlotKey);
  const rules = useSelectRules(linePlotKey);
  const theme = Layout.useSelectTheme();
  const selectedRuleKeys = rules
    .filter((rule) => rule.selected)
    .map((rule) => rule.key);
  const dispatch = useDispatch();
  const setSelectedRuleKeys = (keys: string[]): void => {
    dispatch(setSelectedRule({ key: linePlotKey, ruleKey: keys }));
  };
  const shownRuleKey = selectedRuleKeys[selectedRuleKeys.length - 1];
  const shownRule = rules.find((rule) => rule.key === shownRuleKey);
  const handleCreateRule = (): void => {
    const visColors = theme?.colors.visualization.palettes.default ?? [];
    const colorVal = color.hex(
      visColors[rules.length % visColors.length] ?? color.ZERO,
    );
    const key = id.create();
    const axis = Y1;
    const position = bounds.mean(axes[axis].bounds);
    dispatch(
      setRule({ key: linePlotKey, rule: { key, color: colorVal, axis, position } }),
    );
    setSelectedRuleKeys([key]);
  };
  const handleChangeLabel = (label: string, key: string = shownRuleKey): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key, label } }));
  };
  const handleChangeUnits = (units: string): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key: shownRuleKey, units } }));
  };
  const handleChangePosition = (position: number): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key: shownRuleKey, position } }));
  };
  const handleChangeColor = (v: color.Color): void => {
    dispatch(
      setRule({ key: linePlotKey, rule: { key: shownRuleKey, color: color.hex(v) } }),
    );
  };
  const handleChangeAxis = (axis: AxisKey): void => {
    const position = bounds.mean(axes[axis].bounds);
    dispatch(
      setRule({ key: linePlotKey, rule: { key: shownRuleKey, axis, position } }),
    );
  };
  const handleChangeLineWidth = (lineWidth: number): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key: shownRuleKey, lineWidth } }));
  };
  const handleChangeLineDash = (lineDash: number): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key: shownRuleKey, lineDash } }));
  };
  const handleRemoveRules = (keys: string[]): void => {
    dispatch(removeRule({ key: linePlotKey, ruleKeys: keys }));
    const newSelectedRuleKey = rules.find((rule) => !keys.includes(rule.key))?.key;
    setSelectedRuleKeys(newSelectedRuleKey == null ? [] : [newSelectedRuleKey]);
  };
  if (shownRule == null) return <EmptyContent onCreateRule={handleCreateRule} />;
  return (
    <Flex.Box x style={{ height: "100%" }} empty>
      <List
        selected={selectedRuleKeys}
        onChange={setSelectedRuleKeys}
        rules={rules}
        onCreate={handleCreateRule}
        onRemoveAnnotations={handleRemoveRules}
        onLabelChange={handleChangeLabel}
        layoutKey={linePlotKey}
      />
      <Divider.Divider y />
      <RuleContent
        rule={shownRule}
        onChangeLabel={handleChangeLabel}
        onChangeUnits={handleChangeUnits}
        onChangePosition={handleChangePosition}
        onChangeColor={handleChangeColor}
        onChangeAxis={handleChangeAxis}
        onChangeLineWidth={handleChangeLineWidth}
        onChangeLineDash={handleChangeLineDash}
      />
    </Flex.Box>
  );
};
