// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState } from "react";

import { Icon } from "@synnaxlabs/media";
import {
  Button,
  Color,
  ColorSwatch,
  Divider,
  Header,
  Input,
  List,
  Space,
  Status,
  componentRenderProp,
  Select,
  InputNumberProps,
  CrudeColor,
  ColorSwatchProps,
} from "@synnaxlabs/pluto";
import { CrudeColor } from "@synnaxlabs/pluto/dist/core/color/color";
import { nanoid } from "nanoid";
import { useDispatch } from "react-redux";

import { RuleState, setLinePlotRule } from "../store/slice";

import { useSelectLinePlot } from "@/line/store/selectors";
import { AXIS_KEYS, AxisKey } from "@/vis";

export interface LinePlotAnnotationsControlsProps {
  layoutKey: string;
}

export const LinePlotAnnotationsControls = ({
  layoutKey,
}: LinePlotAnnotationsControlsProps): ReactElement => {
  const vis = useSelectLinePlot(layoutKey);

  const dispatch = useDispatch();

  const [selected, setSelected] = useState<string>(vis?.rules[0]?.key ?? "");

  const handleUnitsChange = (unit: string): void => {
    dispatch(
      setLinePlotRule({
        key: layoutKey,
        rule: {
          key: selected,
          units: unit,
        },
      })
    );
  };

  const handleLabelChange = (label: string): void => {
    dispatch(
      setLinePlotRule({
        key: layoutKey,
        rule: {
          key: selected,
          label,
        },
      })
    );
  };

  const handlePositionChange = (position: number): void => {
    dispatch(
      setLinePlotRule({
        key: layoutKey,
        rule: {
          key: selected,
          position,
        },
      })
    );
  };

  const handleColorChange = (color: Color): void => {
    dispatch(
      setLinePlotRule({
        key: layoutKey,
        rule: {
          key: selected,
          color: color.hex,
        },
      })
    );
  };

  const handleAxisChange = (axis: AxisKey): void => {
    dispatch(
      setLinePlotRule({
        key: layoutKey,
        rule: {
          key: selected,
          axis,
        },
      })
    );
  };

  const handleLineWidthChange = (lineWidth: number): void => {
    dispatch(
      setLinePlotRule({
        key: layoutKey,
        rule: {
          key: selected,
          lineWidth,
        },
      })
    );
  };

  const handleLineDashChange = (lineDash: number): void => {
    dispatch(
      setLinePlotRule({
        key: layoutKey,
        rule: {
          key: selected,
          lineDash,
        },
      })
    );
  };

  const createRule = (): void => {
    const key = nanoid();
    dispatch(
      setLinePlotRule({
        key: layoutKey,
        rule: {
          key,
        },
      })
    );
    setSelected(key);
  };

  const selectedRule = vis.rules.find((rule) => rule.key === selected);

  const emptyContent = (
    <Space.Centered direction="x">
      <Status.Text variant="disabled" hideIcon>
        No annotations added:
      </Status.Text>
      <Button
        variant="outlined"
        onClick={(e) => {
          e.stopPropagation();
          createRule();
        }}
      >
        Create a new annotation
      </Button>
    </Space.Centered>
  );

  let content: ReactElement = emptyContent;

  if (selectedRule != null) {
    content = (
      <Space direction="y" style={{ flexGrow: "1" }} empty>
        <Header level="p">
          <Header.Title>{`Rule - ${selectedRule.label}`}</Header.Title>
        </Header>
        <Space direction="x" style={{ padding: "2rem" }} wrap>
          <Input.Item<string>
            label="Label"
            onChange={handleLabelChange}
            value={selectedRule.label}
          />
          <Input.Item<string>
            label="Units"
            onChange={handleUnitsChange}
            value={selectedRule.units}
          />
          <Input.Item<number>
            label="Position"
            onChange={handlePositionChange}
            value={selectedRule.position}
          >
            {componentRenderProp(Input.Numeric)}
          </Input.Item>
          <Input.Item<CrudeColor, Color, ColorSwatchProps>
            label="Color"
            onChange={handleColorChange}
            value={new Color(selectedRule.color)}
          >
            {componentRenderProp(ColorSwatch)}
          </Input.Item>
          <Input.Item<AxisKey>
            label="Axis"
            onChange={handleAxisChange}
            value={selectedRule.axis}
          >
            {(props) => (
              <Select
                columns={[{ key: "name", name: "Axis" }]}
                data={AXIS_KEYS.map((a) => ({ name: a.toUpperCase(), key: a }))}
                tagKey="name"
                allowClear={false}
                {...props}
              />
            )}
          </Input.Item>
          <Input.Item<number, number, InputNumberProps>
            label="Line Width"
            onChange={handleLineWidthChange}
            value={selectedRule.lineWidth}
            bounds={{ lower: 1, upper: 10 }}
          >
            {componentRenderProp(Input.Numeric)}
          </Input.Item>
          <Input.Item<number, number, InputNumberProps>
            label="Line Dash"
            onChange={handleLineDashChange}
            value={selectedRule.lineDash}
            bounds={{ lower: 0, upper: 50 }}
          >
            {componentRenderProp(Input.Numeric)}
          </Input.Item>
        </Space>
      </Space>
    );
  }

  return (
    <Space direction="x" style={{ height: "100%", width: "100%" }} empty>
      <Space direction="y" empty>
        <List<string, RuleState> data={vis.rules}>
          <Header level="p">
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
          </Header>
          <List.Selector
            value={[selected]}
            allowMultiple={false}
            allowNone={false}
            onChange={([v]) => {
              setSelected(v);
            }}
          />
          <List.Core.Virtual<string, RuleState>
            itemHeight={27}
            style={{ height: "100%", width: 200 }}
          >
            {({ onSelect, selected, style, entry: { key, label } }) => (
              <Button
                key={key}
                onClick={() => {
                  onSelect?.(key);
                }}
                style={{
                  ...style,
                  width: "100%",
                  backgroundColor: selected ? "var(--pluto-primary-z-40)" : "",
                  borderRadius: 0,
                }}
                variant="text"
              >
                {label}
              </Button>
            )}
          </List.Core.Virtual>
        </List>
      </Space>
      <Divider direction="y" />
      {content}
    </Space>
  );
};
