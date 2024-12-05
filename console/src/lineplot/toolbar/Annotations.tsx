// Copyright 2024 Synnax Labs, Inc.
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
  Header,
  Input,
  Menu as PMenu,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { List as PList } from "@synnaxlabs/pluto/list";
import { bounds, id } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";

import { Menu } from "@/components/menu";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { AXIS_KEYS, type AxisKey, Y1 } from "@/lineplot/axis";
import { useSelect } from "@/lineplot/selectors";
import { removeRule, type RuleState, setRule } from "@/lineplot/slice";

interface ListItemProps extends PList.ItemProps<string, RuleState> {}

const ListItem = ({
  onSelect,
  selected,
  translate,
  entry: { key, label },
}: ListItemProps): ReactElement => (
  <Button.Button
    key={key}
    id={key}
    className={CSS(PMenu.CONTEXT_TARGET, selected && PMenu.CONTEXT_SELECTED)}
    onClick={() => {
      onSelect?.(key);
    }}
    style={{
      position: "absolute",
      transform: `translateY(${translate}px)`,
      width: "100%",
      backgroundColor: selected ? "var(--pluto-primary-z-20)" : "",
      borderRadius: 0,
    }}
    variant="text"
  >
    {label}
  </Button.Button>
);

interface ListProps {
  rules: RuleState[];
  selected: string[];
  onChange: (keys: string[]) => void;
  onCreate: () => void;
  onRemoveRules: (keys: string[]) => void;
}

const List = ({
  selected,
  onChange,
  rules,
  onCreate,
  onRemoveRules,
}: ListProps): ReactElement => {
  const menuProps = PMenu.useContextMenu();
  return (
    <PMenu.ContextMenu
      menu={({ keys }) => {
        const onChange = (key: string): void => {
          switch (key) {
            case "remove":
              onRemoveRules(keys);
          }
        };
        return (
          <PMenu.Menu level="small" onChange={onChange}>
            <PMenu.Item itemKey="remove" size="small" startIcon={<Icon.Delete />}>
              Remove Annotation
            </PMenu.Item>
            <Menu.HardReloadItem />
          </PMenu.Menu>
        );
      }}
      {...menuProps}
      style={{ height: "100%" }}
    >
      <PList.List<string, RuleState> data={rules}>
        <Header.Header level="p">
          <Header.Title>Annotations</Header.Title>
          <Header.Actions>
            {[
              {
                key: "add",
                title: "Add",
                children: <Icon.Add />,
                onClick: onCreate,
              },
            ]}
          </Header.Actions>
        </Header.Header>
        <PList.Selector
          value={selected}
          allowNone={false}
          replaceOnSingle
          onChange={onChange}
        >
          <PList.Core.Virtual<string, RuleState>
            itemHeight={27}
            style={{ height: "100%", width: 200 }}
          >
            {ListItem}
          </PList.Core.Virtual>
        </PList.Selector>
      </PList.List>
    </PMenu.ContextMenu>
  );
};

interface EmptyContentProps {
  onCreateRule: () => void;
}

const EmptyContent = ({ onCreateRule }: EmptyContentProps): ReactElement => (
  <Align.Center direction="x" size="small">
    <Status.Text variant="disabled" hideIcon>
      No annotations added.
    </Status.Text>
    <Text.Link level="p" onClick={onCreateRule}>
      Create a new one.
    </Text.Link>
  </Align.Center>
);

const AXIS_DATA = AXIS_KEYS.map((key) => ({ name: key.toUpperCase(), key }));

interface RuleContentProps {
  linePlotKey: string;
  rule: RuleState;
  onLabelChange: (label: string) => void;
  onUnitsChange: (units: string) => void;
  onPositionChange: (position: number) => void;
  onColorChange: (color: Color.Color) => void;
  onAxisChange: (axis: AxisKey) => void;
  onLineWidthChange: (lineWidth: number) => void;
  onLineDashChange: (lineDash: number) => void;
}

const RuleContent = ({
  rule: { label, units, position, color, axis, lineWidth, lineDash },
  onLabelChange,
  onUnitsChange,
  onPositionChange,
  onColorChange,
  onAxisChange,
  onLineWidthChange,
  onLineDashChange,
}: RuleContentProps): ReactElement => (
  <Align.Space direction="y" grow empty>
    <Header.Header level="p">
      <Header.Title>{label}</Header.Title>
    </Header.Header>
    <Align.Space direction="x" style={{ padding: "2rem" }} wrap>
      <Input.Item label="Label">
        <Input.Text variant="shadow" onChange={onLabelChange} value={label} />
      </Input.Item>
      <Input.Item label="Units">
        <Input.Text variant="shadow" onChange={onUnitsChange} value={units} />
      </Input.Item>
      <Input.Item label="Position">
        <Input.Numeric variant="shadow" onChange={onPositionChange} value={position} />
      </Input.Item>
      <Input.Item label="Color">
        <Color.Swatch value={color} onChange={onColorChange} />
      </Input.Item>
      <Input.Item label="Axis">
        <Select.Single
          onChange={onAxisChange}
          value={axis}
          columns={[{ key: "name", name: "Axis" }]}
          data={AXIS_DATA}
          entryRenderKey="name"
          allowNone={false}
        />
      </Input.Item>
      <Input.Item label="Line Width">
        <Input.Numeric
          variant="shadow"
          bounds={{ lower: 1, upper: 10 }}
          onChange={onLineWidthChange}
          value={lineWidth}
        />
      </Input.Item>
      <Input.Item label="Line Dash">
        <Input.Numeric
          variant="shadow"
          bounds={{ lower: 0, upper: 50 }}
          onChange={onLineDashChange}
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
  const vis = useSelect(linePlotKey);
  const theme = Layout.useSelectTheme();
  const dispatch = useDispatch();
  const initialSelected = vis.rules.length > 0 ? [vis.rules[0].key] : [];
  const [selected, setSelected] = useState(initialSelected);
  const firstSelected = selected[0] ?? "";
  const handleLabelChange = (label: string): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key: firstSelected, label } }));
  };
  const handleUnitsChange = (units: string): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key: firstSelected, units } }));
  };
  const handlePositionChange = (position: number): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key: firstSelected, position } }));
  };
  const handleColorChange = (color: Color.Color): void => {
    dispatch(
      setRule({ key: linePlotKey, rule: { key: firstSelected, color: color.hex } }),
    );
  };
  const handleAxisChange = (axis: AxisKey): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key: firstSelected, axis } }));
  };
  const handleLineWidthChange = (lineWidth: number): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key: firstSelected, lineWidth } }));
  };
  const handleLineDashChange = (lineDash: number): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key: firstSelected, lineDash } }));
  };
  const handleCreateRule = (): void => {
    const visColors = theme?.colors.visualization.palettes.default ?? [];
    const color = visColors[vis.rules.length % visColors.length]?.hex;
    const key = id.id();
    const axis = Y1;
    const position = bounds.mean(vis.axes.axes[axis].bounds);
    dispatch(setRule({ key: linePlotKey, rule: { key, color, axis, position } }));
    setSelected([key]);
  };
  const handleRemoveRules = (keys: string[]): void => {
    dispatch(removeRule({ key: linePlotKey, ruleKeys: keys }));
    setSelected(selected.filter((k) => !keys.includes(k)));
  };
  const selectedRule = vis.rules.find((rule) => rule.key === firstSelected);
  const content: ReactElement =
    selectedRule == null ? (
      <EmptyContent onCreateRule={handleCreateRule} />
    ) : (
      <RuleContent
        linePlotKey={linePlotKey}
        rule={selectedRule}
        onLabelChange={handleLabelChange}
        onUnitsChange={handleUnitsChange}
        onPositionChange={handlePositionChange}
        onColorChange={handleColorChange}
        onAxisChange={handleAxisChange}
        onLineWidthChange={handleLineWidthChange}
        onLineDashChange={handleLineDashChange}
      />
    );
  return (
    <Align.Space direction="x" style={{ height: "100%", width: "100%" }} empty>
      <Align.Space direction="y" empty>
        <List
          selected={selected}
          onChange={setSelected}
          rules={vis.rules}
          onCreate={handleCreateRule}
          onRemoveRules={handleRemoveRules}
        />
      </Align.Space>
      <Divider.Divider direction="y" />
      {content}
    </Align.Space>
  );
};
