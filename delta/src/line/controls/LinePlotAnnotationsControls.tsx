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
  Divider,
  Header,
  Input,
  List,
  Status,
  componentRenderProp,
  Select,
  Align,
} from "@synnaxlabs/pluto";
import { nanoid } from "nanoid";
import { useDispatch } from "react-redux";

import { useSelectLinePlot } from "@/line/store/selectors";
import { AXIS_KEYS, AxisKey } from "@/vis";

import { RuleState, setLinePlotRule } from "../store/slice";

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

  const handleColorChange = (color: Color.Color): void => {
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
    <Align.Center direction="x">
      <Status.Text variant="disabled" hideIcon>
        No annotations added:
      </Status.Text>
      <Button.Button
        variant="outlined"
        onClick={(e) => {
          e.stopPropagation();
          createRule();
        }}
      >
        Create a new annotation
      </Button.Button>
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
          <Input.Item<string>
            label="Label"
            onChange={handleLabelChange}
            value={selectedRule.label}
            variant="shadow"
          />
          <Input.Item<string>
            label="Units"
            onChange={handleUnitsChange}
            value={selectedRule.units}
            variant="shadow"
          />
          <Input.Item<number>
            label="Position"
            onChange={handlePositionChange}
            value={selectedRule.position}
            variant="shadow"
          >
            {componentRenderProp(Input.Numeric)}
          </Input.Item>
          <Input.Item<Color.Crude, Color.Color, Color.SwatchProps>
            label="Color"
            onChange={handleColorChange}
            value={new Color.Color(selectedRule.color)}
          >
            {componentRenderProp(Color.Swatch)}
          </Input.Item>
          <Input.Item<AxisKey>
            label="Axis"
            onChange={handleAxisChange}
            value={selectedRule.axis}
          >
            {(props) => (
              <Select.Single
                columns={[{ key: "name", name: "Axis" }]}
                data={AXIS_KEYS.map((a) => ({ name: a.toUpperCase(), key: a }))}
                tagKey="name"
                allowClear={false}
                {...props}
              />
            )}
          </Input.Item>
          <Input.Item<number, number, Input.NumericProps>
            label="Line Width"
            onChange={handleLineWidthChange}
            value={selectedRule.lineWidth}
            bounds={{ lower: 1, upper: 10 }}
            variant="shadow"
          >
            {componentRenderProp(Input.Numeric)}
          </Input.Item>
          <Input.Item<number, number, Input.NumericProps>
            label="Line Dash"
            onChange={handleLineDashChange}
            value={selectedRule.lineDash}
            bounds={{ lower: 0, upper: 50 }}
            variant="shadow"
          >
            {componentRenderProp(Input.Numeric)}
          </Input.Item>
        </Align.Space>
      </Align.Space>
    );
  }

  return (
    <Align.Space direction="x" style={{ height: "100%", width: "100%" }} empty>
      <Align.Space direction="y" empty>
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
              <Button.Button
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
              </Button.Button>
            )}
          </List.Core.Virtual>
        </List.List>
      </Align.Space>
      <Divider.Divider direction="y" />
      {content}
    </Align.Space>
  );
};
