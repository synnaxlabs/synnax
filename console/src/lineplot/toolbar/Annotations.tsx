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
  Menu,
  Select,
  Status,
  Text,
  Theming,
} from "@synnaxlabs/pluto";
import { List } from "@synnaxlabs/pluto/list";
import { nanoid } from "nanoid/non-secure";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { AXIS_KEYS, AxisKey } from "@/lineplot/axis";
import { useSelect } from "@/lineplot/selectors";
import { removeRule, type RuleState, setRule } from "@/lineplot/slice";

export interface AnnotationsProps {
  layoutKey: string;
}

export const Annotations = ({ layoutKey }: AnnotationsProps): ReactElement => {
  const vis = useSelect(layoutKey);
  const theme = Theming.use();

  const dispatch = useDispatch();

  const [allSelected, setAllSelected] = useState<string[]>([vis?.rules[0]?.key ?? ""]);
  const selected = allSelected[0];

  const handleUnitsChange = (unit: string): void => {
    dispatch(
      setRule({
        key: layoutKey,
        rule: {
          key: selected,
          units: unit,
        },
      }),
    );
  };

  const handleLabelChange = (label: string): void => {
    dispatch(
      setRule({
        key: layoutKey,
        rule: {
          key: selected,
          label,
        },
      }),
    );
  };

  const handlePositionChange = (position: number): void => {
    dispatch(
      setRule({
        key: layoutKey,
        rule: {
          key: selected,
          position,
        },
      }),
    );
  };

  const handleColorChange = (color: Color.Color): void => {
    dispatch(
      setRule({
        key: layoutKey,
        rule: {
          key: selected,
          color: color.hex,
        },
      }),
    );
  };

  const handleAxisChange = (axis: AxisKey): void => {
    dispatch(
      setRule({
        key: layoutKey,
        rule: {
          key: selected,
          axis,
        },
      }),
    );
  };

  const handleLineWidthChange = (lineWidth: number): void => {
    dispatch(
      setRule({
        key: layoutKey,
        rule: {
          key: selected,
          lineWidth,
        },
      }),
    );
  };

  const handleLineDashChange = (lineDash: number): void => {
    dispatch(
      setRule({
        key: layoutKey,
        rule: {
          key: selected,
          lineDash,
        },
      }),
    );
  };

  const createRule = (): void => {
    const key = nanoid();
    dispatch(
      setRule({
        key: layoutKey,
        rule: {
          key,
          color: theme.colors.primary.z.hex,
        },
      }),
    );
    setAllSelected([key]);
  };

  const selectedRule = vis.rules.find((rule) => rule.key === selected);

  const emptyContent = (
    <Align.Center direction="x" size="small">
      <Status.Text variant="disabled" hideIcon>
        No annotations added.
      </Status.Text>
      <Text.Link
        level="p"
        onClick={(e) => {
          e.stopPropagation();
          createRule();
        }}
      >
        Create a new one.
      </Text.Link>
    </Align.Center>
  );

  let content: ReactElement = emptyContent;

  if (selectedRule != null) {
    content = (
      <Align.Space direction="y" style={{ flexGrow: "1" }} empty>
        <Header.Header level="p">
          <Header.Title>{`Rule - ${selectedRule.label}`}</Header.Title>
        </Header.Header>
        <Align.Space direction="x" style={{ padding: "2rem" }} wrap>
          <Input.Item label="Label">
            <Input.Text
              variant="shadow"
              onChange={handleLabelChange}
              value={selectedRule.label}
            />
          </Input.Item>
          <Input.Item label="Units">
            <Input.Text
              variant="shadow"
              onChange={handleUnitsChange}
              value={selectedRule.units}
            />
          </Input.Item>
          <Input.Item label="Position">
            <Input.Numeric
              variant="shadow"
              onChange={handlePositionChange}
              value={selectedRule.position ?? 0}
            />
          </Input.Item>
          <Input.Item label="Color">
            <Color.Swatch value={selectedRule.color} onChange={handleColorChange} />
          </Input.Item>
          <Input.Item label="Axis">
            <Select.Single
              onChange={handleAxisChange}
              value={selectedRule.axis}
              columns={[{ key: "name", name: "Axis" }]}
              data={AXIS_KEYS.map((a) => ({ name: a.toUpperCase(), key: a }))}
              entryRenderKey="name"
              allowNone={false}
            />
          </Input.Item>
          <Input.Item label="Line Width">
            <Input.Numeric
              variant="shadow"
              bounds={{ lower: 1, upper: 10 }}
              onChange={handleLineWidthChange}
              value={selectedRule.lineWidth}
            />
          </Input.Item>
          <Input.Item label="Line Dash">
            <Input.Numeric
              variant="shadow"
              bounds={{ lower: 0, upper: 50 }}
              onChange={handleLineDashChange}
              value={selectedRule.lineDash}
            />
          </Input.Item>
        </Align.Space>
      </Align.Space>
    );
  }

  const menuProps = Menu.useContextMenu();

  return (
    <Align.Space direction="x" style={{ height: "100%", width: "100%" }} empty>
      <Align.Space direction="y" empty>
        <Menu.ContextMenu
          menu={({ keys }) => {
            const onChange = (key: string): void => {
              switch (key) {
                case "remove":
                  dispatch(
                    removeRule({
                      key: layoutKey,
                      ruleKeys: keys,
                    }),
                  );
              }
            };

            return (
              <Menu.Menu level="small" onChange={onChange}>
                <Menu.Item itemKey="remove" startIcon={<Icon.Delete />}>
                  Remove Annotation
                </Menu.Item>
              </Menu.Menu>
            );
          }}
          {...menuProps}
        >
          <List.List<string, RuleState> data={vis.rules}>
            <Header.Header level="p">
              <Header.Title>Annotations</Header.Title>
              <Header.Actions>
                {[
                  {
                    key: "add",
                    title: "Add",
                    children: <Icon.Add />,
                    onClick: createRule,
                  },
                ]}
              </Header.Actions>
            </Header.Header>
            <List.Selector
              value={allSelected}
              allowNone={false}
              replaceOnSingle
              onChange={setAllSelected}
            >
              <List.Core.Virtual<string, RuleState>
                itemHeight={27}
                style={{ height: "100%", width: 200 }}
              >
                {({ onSelect, selected, translate, entry: { key, label } }) => (
                  <Button.Button
                    key={key}
                    id={key}
                    className={CSS(
                      Menu.CONTEXT_TARGET,
                      selected && Menu.CONTEXT_SELECTED,
                    )}
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
                )}
              </List.Core.Virtual>
            </List.Selector>
          </List.List>
        </Menu.ContextMenu>
      </Align.Space>
      <Divider.Divider direction="y" />
      {content}
    </Align.Space>
  );
};
