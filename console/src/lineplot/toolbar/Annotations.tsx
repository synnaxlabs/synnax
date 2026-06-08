// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import {
  Button,
  Color,
  Divider,
  Flex,
  Icon,
  Input,
  LinePlot,
  List as PList,
  Menu,
  Select,
  Text,
} from "@synnaxlabs/pluto";
import { bounds, type color, id } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu, EmptyAction } from "@/components";
import { useSelectSelectedRules } from "@/lineplot/selectors";
import { setSelectedRule } from "@/lineplot/slice";

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
  const entry = LinePlot.useSelectRule({ key: layoutKey, ruleKey: itemKey });
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
  rules: lineplot.Rule[];
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
  const menuProps = Menu.useContextMenu();
  const { data } = PList.useStaticData<string, lineplot.Rule>({ data: rules });
  return (
    <Flex.Box x pack style={{ width: "20%" }} align="start">
      <Flex.Box style={{ padding: "0.5rem" }}>
        <Button.Button tooltip="Add Rule" onClick={onCreate} size="small">
          <Icon.Add />
        </Button.Button>
      </Flex.Box>
      <Divider.Divider y />
      <Select.Frame<string, lineplot.Rule>
        multiple
        data={data}
        value={selected}
        onChange={onChange}
        replaceOnSingle
        allowNone={false}
      >
        <Menu.ContextMenu
          menu={({ keys }) => (
            <ContextMenu.Menu>
              <ContextMenu.DeleteItem onClick={() => onRemoveAnnotations(keys)} />
              <Divider.Divider x />
              <ContextMenu.ReloadConsoleItem />
            </ContextMenu.Menu>
          )}
          {...menuProps}
        >
          <PList.Items<string, lineplot.Rule> onContextMenu={menuProps.open} grow>
            {({ key, ...rest }) => (
              <ListItem
                layoutKey={layoutKey}
                key={key}
                {...rest}
                onChangeLabel={(v) => onLabelChange(v, key)}
              />
            )}
          </PList.Items>
        </Menu.ContextMenu>
      </Select.Frame>
    </Flex.Box>
  );
};

const AXIS_DATA: lineplot.AxisKey[] = ["y1", "y2"];

const SelectAxis = (
  props: Omit<Select.ButtonsProps<lineplot.AxisKey>, "keys">,
): ReactElement => (
  <Select.Buttons {...props} keys={AXIS_DATA}>
    <Select.Button itemKey="y1">Y1</Select.Button>
    <Select.Button itemKey="y2">Y2</Select.Button>
  </Select.Buttons>
);

interface RuleContentProps {
  rule: LinePlot.DerivedRule;
  onChangeLabel: (label: string) => void;
  onChangeUnits: (units: string) => void;
  onChangePosition: (position: number) => void;
  onChangeColor: (color: color.Color) => void;
  onChangeAxis: (axis: lineplot.AxisKey) => void;
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
  layoutKey: string;
}

export const Annotations = ({ layoutKey: key }: AnnotationsProps): ReactElement => {
  const axes = LinePlot.useSelectAxes({ key });
  const rules = LinePlot.useSelectRules({ key });
  const selectedRuleKeys = useSelectSelectedRules(key);
  const reduxDispatch = useDispatch();
  const { dispatch } = LinePlot.useDispatch();

  const setSelectedRuleKeys = useCallback(
    (keys: string[]): void => {
      reduxDispatch(setSelectedRule({ key, ruleKey: keys }));
    },
    [reduxDispatch, key],
  );

  const shownRuleKey = selectedRuleKeys[selectedRuleKeys.length - 1];
  const shownRule = rules.find((rule) => rule.key === shownRuleKey);

  const apply = useCallback(
    (action: lineplot.Action): void => {
      dispatch({ key, actions: [action] });
    },
    [dispatch, key],
  );

  const handleCreateRule = useCallback((): void => {
    const key = id.create();
    const axis: lineplot.AxisKey = "y1";
    const position = bounds.mean(axes[axis].bounds);
    const label = `Rule ${rules.length + 1}`;
    const rule = lineplot.ruleZ.parse({ key, label, axis, position });
    dispatch({ key, actions: [lineplot.setRule({ rule })] });
    setSelectedRuleKeys([key]);
  }, [dispatch, key, axes, rules.length, setSelectedRuleKeys]);

  const handleChangeLabel = (label: string, key: string = shownRuleKey): void => {
    if (key == null) return;
    apply(lineplot.setRuleLabel({ key, label }));
  };
  const handleChangeUnits = (units: string): void => {
    if (shownRule == null) return;
    apply(lineplot.setRuleUnits({ key: shownRule.key, units }));
  };
  const handleChangePosition = (position: number): void => {
    if (shownRule == null) return;
    apply(lineplot.setRulePosition({ key: shownRule.key, position }));
  };
  const handleChangeColor = (v: color.Color): void => {
    if (shownRule == null) return;
    apply(lineplot.setRuleColor({ key: shownRule.key, color: v }));
  };
  const handleChangeAxis = (axis: lineplot.AxisKey): void => {
    if (shownRule == null) return;
    const position = bounds.mean(axes[axis].bounds);
    dispatch({
      key,
      actions: [
        lineplot.setRuleAxis({ key: shownRule.key, axis }),
        lineplot.setRulePosition({ key: shownRule.key, position }),
      ],
    });
  };
  const handleChangeLineWidth = (lineWidth: number): void => {
    if (shownRule == null) return;
    apply(lineplot.setRuleLineWidth({ key: shownRule.key, lineWidth }));
  };
  const handleChangeLineDash = (lineDash: number): void => {
    if (shownRule == null) return;
    apply(lineplot.setRuleLineDash({ key: shownRule.key, lineDash }));
  };
  const handleRemoveRules = (keys: string[]): void => {
    dispatch({
      key,
      actions: keys.map((key) => lineplot.removeRule({ key })),
    });
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
        layoutKey={key}
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
