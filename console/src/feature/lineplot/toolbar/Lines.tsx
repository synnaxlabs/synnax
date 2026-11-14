// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/lineplot/toolbar/Lines.css";

import { lineplot } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Channel, LinePlot } from "@synnaxlabs/pluto";
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

interface SamplingOption<K extends string> {
  key: K;
  name: string;
  /** What the trigger shows, where the name alone does not say what it sets. */
  label: string;
  description: string;
  icon: Icon.ReactElement;
}

const AGGREGATIONS: SamplingOption<lineplot.Aggregation>[] = [
  {
    key: "min_max",
    name: "Min/max",
    label: "Min/max",
    description: "Keeps spikes visible.",
    icon: <Icon.Waveform />,
  },
  {
    key: "average",
    name: "Average",
    label: "Average",
    description: "Smooths out noise.",
    icon: <Icon.Wave.Sine />,
  },
];

const DETAILS: SamplingOption<lineplot.Detail>[] = [
  {
    key: "low",
    name: "Low",
    label: "Low detail",
    description: "Least detail. Loads fastest.",
    icon: <Icon.Meter.Low />,
  },
  {
    key: "medium",
    name: "Medium",
    label: "Medium detail",
    description: "Balances detail and speed.",
    icon: <Icon.Meter.Medium />,
  },
  {
    key: "high",
    name: "High",
    label: "High detail",
    description: "Most detail. Loads the most data.",
    icon: <Icon.Meter.High />,
  },
];

const SAMPLING_CLASS = CSS.BE("line-plot", "toolbar-sampling");
const SAMPLING_TRIGGER_PROPS = {
  variant: "shadow",
  className: CSS.BE("line-plot", "toolbar-sampling-trigger"),
} as const;

interface SelectSamplingProps<K extends string> extends Pick<
  Select.SingleSimpleProps<K>,
  "onChange" | "resourceName"
> {
  value: K;
  options: SamplingOption<K>[];
}

const SelectSampling = <K extends string>({
  value,
  options,
  resourceName,
  ...rest
}: SelectSamplingProps<K>): ReactElement => (
  <Select.Simple<K>
    {...rest}
    value={value}
    resourceName={resourceName}
    className={CSS.cls(
      SAMPLING_CLASS,
      CSS.BEM("line-plot", "toolbar-sampling", resourceName),
    )}
    allowNone={false}
    variant="floating"
    searchHidden
    icon={options.find((o) => o.key === value)?.icon}
    triggerProps={SAMPLING_TRIGGER_PROPS}
  >
    {options.map(({ key, name, label, description, icon }) => (
      <Select.Item<K> key={key} itemKey={key} label={label} textValue={name}>
        {icon}
        {name}
        <Select.ItemDescription>{description}</Select.ItemDescription>
      </Select.Item>
    ))}
  </Select.Simple>
);

const Line = ({ itemKey, index }: LineProps): ReactElement | null => {
  const line = LinePlot.useLine({ lineKey: itemKey });
  // A line whose x channel is not an index always reads decimated samples.
  const { data: xIsIndex = true } = Channel.useResultIsIndex(
    line.xChannel > 0 ? { key: line.xChannel } : null,
  );
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

  const handleColorChange = (color?: color.Color) =>
    dispatch(lineplot.setLineColor({ key: itemKey, color }));

  return (
    <List.Item itemKey={itemKey} index={index} key={itemKey} gap="large" preventClick>
      <Input.Text
        selectOnFocus
        variant="shadow"
        value={line.label}
        onChange={handleLabelChange}
        full="x"
      >
        {!line.isDefaultLabel && (
          <Button.Button
            onClick={handleLabelReset}
            tooltip="Reset to channel name"
            tooltipLocation={{ y: "top" }}
            variant="outlined"
          >
            <Icon.Sync />
          </Button.Button>
        )}
      </Input.Text>
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
      {xIsIndex && (
        <SelectSampling<lineplot.Aggregation>
          value={line.aggregation}
          options={AGGREGATIONS}
          onChange={handleAggregationChange}
          resourceName="aggregation"
        />
      )}
      <SelectSampling<lineplot.Detail>
        value={line.detail}
        options={DETAILS}
        onChange={handleDetailChange}
        resourceName="detail"
      />
      <Color.Swatch
        value={line.pickedColor}
        fallback={line.autoColor}
        onChange={handleColorChange}
        size="small"
        variant="text"
      />
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
