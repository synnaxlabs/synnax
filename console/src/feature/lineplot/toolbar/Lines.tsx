// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Channel, Color, LinePlot } from "@synnaxlabs/pluto";
import { type bounds, type color, type xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";

const EmptyContent = () => {
  const { onSelect } = Select.useContext<string>();
  return (
    <Empty.Action
      message="No lines plotted"
      action="Select channels in the data tab"
      onClick={(e) => {
        e.stopPropagation();
        onSelect("data");
      }}
    />
  );
};

interface LineProps extends Omit<List.ItemProps<string>, "onChange"> {}

const STROKE_WIDTH_BOUNDS: bounds.Bounds = { lower: 1, upper: 10 };
const STROKE_WIDTH_DRAG_SCALE: xy.XY = { x: 0.1, y: 0.1 };

const SelectAggregation = (
  props: Select.ButtonsProps<lineplot.Aggregation>,
): ReactElement => (
  <Select.Buttons {...props} shrink={0}>
    <Select.Item itemKey="min_max" size="small" tooltip="Draw the range of each group">
      Min/max
    </Select.Item>
    <Select.Item itemKey="average" size="small" tooltip="Draw the mean of each group">
      Average
    </Select.Item>
  </Select.Buttons>
);

const SelectDetail = (props: Select.ButtonsProps<lineplot.Detail>): ReactElement => (
  <Select.Buttons {...props} shrink={0}>
    <Select.Item itemKey="low" size="small">
      Low
    </Select.Item>
    <Select.Item itemKey="medium" size="small">
      Medium
    </Select.Item>
    <Select.Item itemKey="high" size="small">
      High
    </Select.Item>
  </Select.Buttons>
);

const Line = ({ itemKey, index }: LineProps): ReactElement | null => {
  const line = LinePlot.useLine({ lineKey: itemKey });
  const dispatch = LinePlot.useSingleDispatch();

  const handleLabelChange: Input.Control<string>["onChange"] = (label) =>
    dispatch(
      lineplot.setLineLabel({
        key: itemKey,
        label: label.length === 0 ? undefined : label,
      }),
    );

  const handleLabelReset = (): void =>
    dispatch(lineplot.setLineLabel({ key: itemKey }));

  const handleWidthChange = (strokeWidth: number) =>
    dispatch(lineplot.setLineStrokeWidth({ key: itemKey, strokeWidth }));

  const handleAggregationChange = (aggregation: lineplot.Aggregation) =>
    dispatch(lineplot.setLineAggregation({ key: itemKey, aggregation }));

  const handleDetailChange = (detail: lineplot.Detail) =>
    dispatch(lineplot.setLineDetail({ key: itemKey, detail }));

  const handleColorChange = (color: color.Color) =>
    dispatch(lineplot.setLineColor({ key: itemKey, color }));

  return (
    <List.Item itemKey={itemKey} index={index} key={itemKey} gap="large" preventClick>
      <Channel.AliasInput
        channel={line.yChannel}
        variant="shadow"
        value={line.label ?? ""}
        onChange={handleLabelChange}
        isDefault={line.isDefaultLabel}
        onReset={handleLabelReset}
        full="x"
      />
      <Input.Numeric
        value={line.strokeWidth}
        variant="shadow"
        startContent={<Icon.StrokeWidth />}
        onChange={handleWidthChange}
        dragScale={STROKE_WIDTH_DRAG_SCALE}
        bounds={STROKE_WIDTH_BOUNDS}
        shrink={false}
        tooltip="Stroke width"
      />
      <SelectAggregation value={line.aggregation} onChange={handleAggregationChange} />
      <SelectDetail value={line.detail} onChange={handleDetailChange} />
      <Color.Swatch value={line.color} onChange={handleColorChange} size="small" />
    </List.Item>
  );
};

const line = Component.renderProp(Line);

export const Lines = (): ReactElement => {
  const lineKeys = LinePlot.useLineKeys();
  return (
    <List.Frame data={lineKeys}>
      <List.Scroll full="y" className={CSS.BE("line-plot", "toolbar", "lines")}>
        <List.Items<string, lineplot.Line> emptyContent={<EmptyContent />}>
          {line}
        </List.Items>
      </List.Scroll>
    </List.Frame>
  );
};
