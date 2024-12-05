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
import { Layout } from "@/layout";
import { AXIS_KEYS, type AxisKey, Y1 } from "@/lineplot/axis";
import { useSelect } from "@/lineplot/selectors";
import { removeRule, type RuleState, setRule } from "@/lineplot/slice";

interface ListItemProps extends PList.ItemProps<string, RuleState> {
  onChangeLabel: (label: string) => void;
}

const ListItem = ({ entry, onChangeLabel, ...props }: ListItemProps): ReactElement => (
  <PList.ItemFrame entry={entry} {...props} style={{ padding: "1rem 0rem 1rem 2rem" }}>
    <Text.Editable
      value={entry.label}
      level="p"
      style={{ padding: 0 }}
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
    <PList.List<string, RuleState> data={rules}>
      <Button.Icon
        onClick={(e) => {
          e.stopPropagation();
          onCreate();
        }}
        tooltip="Add Annotation"
        style={{ zIndex: 1, position: "absolute", right: 5 }}
      >
        <Icon.Add />
      </Button.Icon>
      <PMenu.ContextMenu
        menu={({ keys }) => {
          const onChange = (key: string): void => {
            switch (key) {
              case "remove":
                onRemoveAnnotations(keys);
            }
          };
          const length = keys.length;
          return (
            <PMenu.Menu level="small" onChange={onChange}>
              <PMenu.Item itemKey="remove" size="small" startIcon={<Icon.Delete />}>
                {`Remove ${length === 1 ? "Annotation" : `${length} Annotations`}`}
              </PMenu.Item>
              <Menu.HardReloadItem />
            </PMenu.Menu>
          );
        }}
        {...menuProps}
      >
        <PList.Selector
          value={selected}
          allowNone={false}
          replaceOnSingle
          onChange={onChange}
        >
          <PList.Core<string, RuleState>>
            {({ key, ...props }) => (
              <ListItem
                key={key}
                {...props}
                onChangeLabel={(v) => onLabelChange(v, key)}
              />
            )}
          </PList.Core>
        </PList.Selector>
      </PMenu.ContextMenu>
    </PList.List>
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
  <Align.Space direction="y" grow style={{ padding: "2rem" }}>
    <Align.Space direction="x" wrap>
      <Input.Item label="Label" grow>
        <Input.Text onChange={onLabelChange} value={label} />
      </Input.Item>
      <Input.Item label="Units">
        <Input.Text onChange={onUnitsChange} value={units} />
      </Input.Item>
      <Input.Item label="Position">
        <Input.Numeric onChange={onPositionChange} value={position} />
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
    </Align.Space>
    <Align.Space direction="x" wrap>
      <Input.Item label="Color">
        <Color.Swatch value={color} onChange={onColorChange} />
      </Input.Item>
      <Input.Item label="Line Width">
        <Input.Numeric
          bounds={{ lower: 1, upper: 10 }}
          onChange={onLineWidthChange}
          value={lineWidth}
        />
      </Input.Item>
      <Input.Item label="Line Dash">
        <Input.Numeric
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
  const handleLabelChange = (label: string, key: string = firstSelected): void => {
    dispatch(setRule({ key: linePlotKey, rule: { key, label } }));
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
      <Align.Space
        direction="y"
        empty
        style={{ minWidth: "230px", height: "100%", position: "relative" }}
      >
        <List
          selected={selected}
          onChange={setSelected}
          rules={vis.rules}
          onCreate={handleCreateRule}
          onRemoveAnnotations={handleRemoveRules}
          onLabelChange={handleLabelChange}
        />
      </Align.Space>
      <Divider.Divider direction="y" />
      {content}
    </Align.Space>
  );
};
