// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Color,
  Divider,
  Input,
  List as PList,
  Menu as PMenu,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { bounds, id, type KeyedNamed } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { Menu } from "@/components/menu";
import { Layout } from "@/layout";
import { type AxisKey, Y1, Y2 } from "@/lineplot/axis";
import { useSelectAxes, useSelectRules } from "@/lineplot/selectors";
import { removeRule, type RuleState, selectRule, setRule } from "@/lineplot/slice";

interface EmptyContentProps {
  onCreateRule: () => void;
}

const EmptyContent = ({ onCreateRule }: EmptyContentProps): ReactElement => (
  <Align.Center direction="x" size="small">
    <Status.Text level="p" variant="disabled" hideIcon>
      No annotations added.
    </Status.Text>
    <Text.Link level="p" onClick={onCreateRule}>
      Create a new one.
    </Text.Link>
  </Align.Center>
);

interface ListItemProps extends PList.ItemProps<string, RuleState> {
  onChangeLabel: (label: string) => void;
}

const ListItem = ({ entry, onChangeLabel, ...props }: ListItemProps): ReactElement => (
  <PList.ItemFrame
    entry={entry}
    {...props}
    // style={{ paddingTop: "0.5", paddingBottom: "0.5rem" }}
    style={{ padding: "0.75rem 1.5rem" }}
  >
    <Text.Editable
      value={entry.label}
      level="p"
      noWrap
      shade={8}
      weight={500}
      style={{ overflow: "hidden", textOverflow: "ellipsis" }}
      onChange={onChangeLabel}
    />
  </PList.ItemFrame>
);

interface ListProps {
  rules: RuleState[];
  selected: string[];
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
  onLabelChange,
}: ListProps): ReactElement => {
  const menuProps = PMenu.useContextMenu();
  return (
    <Align.Space direction="x" empty style={{ width: "20%" }}>
      <Button.Icon tooltip="Add Rule" size="small" onClick={onCreate}>
        <Icon.Add />
      </Button.Icon>
      <Divider.Divider direction="y" />
      <PList.List<string, RuleState> data={rules}>
        <PList.Selector
          value={selected}
          allowNone={false}
          replaceOnSingle
          onChange={onChange}
        >
          <PMenu.ContextMenu
            style={{ width: "100%", overflow: "hidden" }}
            menu={({ keys }) => (
              <PMenu.Menu
                onChange={{ remove: () => onRemoveAnnotations(keys) }}
                level="small"
              >
                <PMenu.Item itemKey="remove" size="small" startIcon={<Icon.Delete />}>
                  Delete
                </PMenu.Item>
                <Divider.Divider direction="x" />
                <Menu.HardReloadItem />
              </PMenu.Menu>
            )}
            {...menuProps}
          >
            <PList.Core<string, RuleState> direction="y" empty grow>
              {({ key, ...props }) => (
                <ListItem
                  key={key}
                  {...props}
                  onChangeLabel={(v) => onLabelChange(v, key)}
                />
              )}
            </PList.Core>
          </PMenu.ContextMenu>
        </PList.Selector>
      </PList.List>
    </Align.Space>
  );
};

const AXIS_DATA: KeyedNamed<AxisKey>[] = [Y1, Y2].map((key) => ({
  name: key.toUpperCase(),
  key: key as AxisKey,
}));

interface RuleContentProps {
  rule: RuleState;
  onChangeLabel: (label: string) => void;
  onChangeUnits: (units: string) => void;
  onChangePosition: (position: number) => void;
  onChangeColor: (color: Color.Color) => void;
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
  <Align.Space direction="y" grow style={{ padding: "1.5rem 2rem" }}>
    <Align.Space direction="x" wrap>
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
        <Select.Button
          size="medium"
          onChange={onChangeAxis}
          value={axis}
          data={AXIS_DATA}
          entryRenderKey="name"
          allowNone={false}
        />
      </Input.Item>
    </Align.Space>
    <Align.Space direction="x" wrap>
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
    </Align.Space>
  </Align.Space>
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
    dispatch(selectRule({ key: linePlotKey, ruleKey: keys }));
  };
  const shownRuleKey = selectedRuleKeys[selectedRuleKeys.length - 1];
  const shownRule = rules.find((rule) => rule.key === shownRuleKey);
  const handleCreateRule = (): void => {
    const visColors = theme?.colors.visualization.palettes.default ?? [];
    const color = visColors[rules.length % visColors.length]?.hex;
    const key = id.id();
    const axis = Y1;
    const position = bounds.mean(axes[axis].bounds);
    dispatch(setRule({ key: linePlotKey, rule: { key, color, axis, position } }));
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
  const handleChangeColor = (color: Color.Color): void => {
    dispatch(
      setRule({ key: linePlotKey, rule: { key: shownRuleKey, color: color.hex } }),
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
    <Align.Space direction="x" style={{ height: "100%" }} empty>
      <List
        selected={selectedRuleKeys}
        onChange={setSelectedRuleKeys}
        rules={rules}
        onCreate={handleCreateRule}
        onRemoveAnnotations={handleRemoveRules}
        onLabelChange={handleChangeLabel}
      />
      <Divider.Divider direction="y" />
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
    </Align.Space>
  );
};
