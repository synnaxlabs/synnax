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

import { ContextMenu } from "@/primitive/context-menu";
import { CSS } from "@/primitive/css";
import { Empty } from "@/primitive/empty";
import { Session } from "@/session";

interface EmptyContentProps {
  onCreateRule: () => void;
}

const EmptyContent = ({ onCreateRule }: EmptyContentProps): ReactElement => (
  <Empty.Action
    x
    message="No annotations added."
    action="Create an annotation"
    onClick={onCreateRule}
  />
);

interface ListItemProps extends PList.ItemProps<string> {}

const ListItem = (props: ListItemProps): ReactElement | null => {
  const { itemKey } = props;
  const entry = LinePlot.useSelectRule({ ruleKey: itemKey });
  const dispatch = LinePlot.useSingleDispatch();
  const handleChangeLabel = (label: string): void =>
    dispatch(lineplot.setRuleLabel({ key: itemKey, label }));
  return (
    <Select.ListItem
      {...props}
      className={CSS.BE("line-plot", "toolbar", "annotations-item")}
      align="center"
      full="x"
      square={false}
    >
      <Text.Editable
        value={entry.label}
        overflow="ellipsis"
        color={10}
        weight={500}
        onChange={handleChangeLabel}
      />
    </Select.ListItem>
  );
};

interface ListProps {
  rules: lineplot.Rule[];
  selected: string[];
  onChange: (keys: string[]) => void;
  onCreate: () => void;
  onRemoveAnnotations: (keys: string[]) => void;
}

const List = ({
  selected,
  onChange,
  rules,
  onCreate,
  onRemoveAnnotations,
}: ListProps): ReactElement => {
  const menuProps = Menu.useContextMenu();
  const { data } = PList.useStaticData<string, lineplot.Rule>({ data: rules });
  return (
    <Flex.Box
      x
      pack
      className={CSS.BE("line-plot", "toolbar", "annotations-list")}
      align="start"
    >
      <Flex.Box className={CSS.BE("line-plot", "toolbar", "annotations-add")}>
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
            {({ key, ...rest }) => <ListItem key={key} {...rest} />}
          </PList.Items>
        </Menu.ContextMenu>
      </Select.Frame>
    </Flex.Box>
  );
};

const AXIS_DATA: lineplot.AxisKey[] = ["y1", "y2"];

const LINE_WIDTH_BOUNDS: bounds.Bounds = { lower: 1, upper: 10 };
const LINE_DASH_BOUNDS: bounds.Bounds = { lower: 0, upper: 50 };

const SelectAxis = (
  props: Omit<Select.ButtonsProps<lineplot.AxisKey>, "keys">,
): ReactElement => (
  <Select.Buttons {...props} keys={AXIS_DATA}>
    <Select.Button itemKey="y1">Y1</Select.Button>
    <Select.Button itemKey="y2">Y2</Select.Button>
  </Select.Buttons>
);

interface DetailsProps {
  ruleKey: string;
}

const Details = ({ ruleKey }: DetailsProps): ReactElement | null => {
  const rule = LinePlot.useSelectRule({ ruleKey });
  const axes = LinePlot.useSelectAxes();
  const dispatch = LinePlot.useSingleDispatch();

  const handleChangeLabel = (label: string): void =>
    dispatch(lineplot.setRuleLabel({ key: ruleKey, label }));
  const handleChangeUnits = (units: string): void =>
    dispatch(lineplot.setRuleUnits({ key: ruleKey, units }));
  const handleChangePosition = (position: number): void =>
    dispatch(lineplot.setRulePosition({ key: ruleKey, position }));
  const handleChangeColor = (v: color.Color): void =>
    dispatch(lineplot.setRuleColor({ key: ruleKey, color: v }));
  const handleChangeAxis = (axis: lineplot.AxisKey): void => {
    const position = bounds.mean(axes[axis].bounds);
    dispatch([
      lineplot.setRuleAxis({ key: ruleKey, axis }),
      lineplot.setRulePosition({ key: ruleKey, position }),
    ]);
  };
  const handleChangeLineWidth = (lineWidth: number): void =>
    dispatch(lineplot.setRuleLineWidth({ key: ruleKey, lineWidth }));
  const handleChangeLineDash = (lineDash: number): void =>
    dispatch(lineplot.setRuleLineDash({ key: ruleKey, lineDash }));

  return (
    <Flex.Box y grow className={CSS.BE("line-plot", "toolbar", "annotations-details")}>
      <Flex.Box x wrap>
        <Input.Item label="Label" grow>
          <Input.Text onChange={handleChangeLabel} value={rule.label} />
        </Input.Item>
        <Input.Item label="Units">
          <Input.Text
            onChange={handleChangeUnits}
            value={rule.units}
            className={CSS.BE("line-plot", "toolbar", "annotations-units")}
          />
        </Input.Item>
        <Input.Item label="Position">
          <Input.Numeric
            onChange={handleChangePosition}
            value={Number(rule.position.toFixed(2))}
            className={CSS.BE("line-plot", "toolbar", "annotations-position")}
          />
        </Input.Item>
        <Input.Item label="Axis">
          <SelectAxis value={rule.axis} onChange={handleChangeAxis} />
        </Input.Item>
      </Flex.Box>
      <Flex.Box x wrap>
        <Input.Item label="Color">
          <Color.Swatch value={rule.color} onChange={handleChangeColor} />
        </Input.Item>
        <Input.Item label="Line Width">
          <Input.Numeric
            bounds={LINE_WIDTH_BOUNDS}
            onChange={handleChangeLineWidth}
            value={rule.lineWidth}
          />
        </Input.Item>
        <Input.Item label="Line Dash">
          <Input.Numeric
            bounds={LINE_DASH_BOUNDS}
            onChange={handleChangeLineDash}
            value={rule.lineDash}
          />
        </Input.Item>
      </Flex.Box>
    </Flex.Box>
  );
};

export const Annotations = (): ReactElement => {
  const key = LinePlot.useKey();
  const axes = LinePlot.useSelectAxes();
  const rules = LinePlot.useSelectRules();
  const selectedRuleKeys = Session.LinePlot.useSelectSelectedRules();
  const dispatchSession = useDispatch();
  const dispatch = LinePlot.useSingleDispatch();

  const setSelectedRuleKeys = useCallback(
    (ruleKey: string[]): void => {
      dispatchSession(Session.LinePlot.setSelectedRule({ key, ruleKey }));
    },
    [dispatchSession, key],
  );

  const handleCreateRule = useCallback((): void => {
    const ruleKey = id.create();
    const axis: lineplot.AxisKey = "y1";
    const position = bounds.mean(axes[axis].bounds);
    const label = `Rule ${rules.length + 1}`;
    const rule = lineplot.ruleZ.parse({ key: ruleKey, label, axis, position });
    dispatch(lineplot.setRule({ rule }));
    setSelectedRuleKeys([ruleKey]);
  }, [dispatch, key, axes, rules.length, setSelectedRuleKeys]);

  const handleRemoveRules = (keys: string[]): void => {
    dispatch(keys.map((ruleKey) => lineplot.removeRule({ key: ruleKey })));
    const newSelectedRuleKey = rules.find((rule) => !keys.includes(rule.key))?.key;
    setSelectedRuleKeys(newSelectedRuleKey == null ? [] : [newSelectedRuleKey]);
  };

  const shownRuleKey = selectedRuleKeys[selectedRuleKeys.length - 1];
  if (shownRuleKey == null || !rules.some((rule) => rule.key === shownRuleKey))
    return <EmptyContent onCreateRule={handleCreateRule} />;
  return (
    <Flex.Box x className={CSS.BE("line-plot", "toolbar", "annotations")} empty>
      <List
        selected={selectedRuleKeys}
        onChange={setSelectedRuleKeys}
        rules={rules}
        onCreate={handleCreateRule}
        onRemoveAnnotations={handleRemoveRules}
      />
      <Divider.Divider y />
      <Details ruleKey={shownRuleKey} />
    </Flex.Box>
  );
};
