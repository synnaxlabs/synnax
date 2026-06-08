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
  Channel,
  Color,
  Icon,
  Input,
  LinePlot,
  List,
  Select,
  Tabs,
  type telem,
} from "@synnaxlabs/pluto";
import { type bounds, type color, type xy } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { EmptyAction } from "@/components";
import { CSS } from "@/css";

export interface LinesProps {
  layoutKey: string;
}

export const Lines = ({ layoutKey }: LinesProps): ReactElement => {
  const lineKeys = LinePlot.useSelectLineKeys({ key: layoutKey });
  const { onSelect } = Tabs.useContext();

  const emptyContent = (
    <EmptyAction
      x
      message="No lines plotted. Select channels using the"
      action="data tab."
      onClick={(e) => {
        e.stopPropagation();
        onSelect?.("data");
      }}
    />
  );

  return (
    <List.Frame data={lineKeys}>
      <List.Items<string, lineplot.Line>
        full="y"
        className={CSS.BE("line-plot", "toolbar", "lines")}
        emptyContent={emptyContent}
      >
        {({ key, index, ...rest }) => (
          <Line key={key} layoutKey={layoutKey} index={index} {...rest} />
        )}
      </List.Items>
    </List.Frame>
  );
};

interface LineProps extends Omit<List.ItemProps<string>, "onChange"> {
  layoutKey: string;
  index: number;
}

const STROKE_WIDTH_BOUNDS: bounds.Bounds = { lower: 1, upper: 11 };
const DOWNSAMPLE_BOUNDS: bounds.Bounds = { lower: 1, upper: 1001 };
const STROKE_WIDTH_DRAG_SCALE: xy.XY = { x: 0.1, y: 0.1 };
const DOWNSAMPLE_DRAG_SCALE: xy.XY = { x: 0.1, y: 0.1 };

interface SelectDownsampleModeProps extends Omit<
  Select.ButtonsProps<telem.DownsampleMode>,
  "keys"
> {}

const KEYS: telem.DownsampleMode[] = ["average", "decimate"];

const SelectDownsampleMode = (props: SelectDownsampleModeProps): ReactElement => (
  <Select.Buttons {...props} keys={KEYS}>
    <Select.Button itemKey="average" size="small">
      Average
    </Select.Button>
    <Select.Button itemKey="decimate" size="small">
      Decimate
    </Select.Button>
  </Select.Buttons>
);

const Line = ({ itemKey, index, layoutKey }: LineProps): ReactElement | null => {
  const line = LinePlot.useSelectLine({ key: layoutKey, lineKey: itemKey });
  const { dispatch } = LinePlot.useDispatch();

  const apply = useCallback(
    (action: lineplot.Action): void => {
      dispatch({ key: layoutKey, actions: [action] });
    },
    [dispatch, layoutKey],
  );

  const handleLabelChange: Input.Control<string>["onChange"] = (value) =>
    apply(lineplot.setLineLabel({ key: itemKey, label: value }));

  const handleWidthChange: Input.Control<number>["onChange"] = (value) =>
    apply(lineplot.setLineStrokeWidth({ key: itemKey, strokeWidth: value }));

  const handleDownsampleChange: Input.Control<number>["onChange"] = (value) =>
    apply(lineplot.setLineDownsample({ key: itemKey, downsample: value }));

  const handleDownsampleModeChange: Select.ButtonsProps<telem.DownsampleMode>["onChange"] =
    (value: telem.DownsampleMode) =>
      apply(lineplot.setLineDownsampleMode({ key: itemKey, downsampleMode: value }));

  const handleColorChange: Input.Control<color.Color>["onChange"] = (value) =>
    apply(lineplot.setLineColor({ key: itemKey, color: value }));

  return (
    <List.Item itemKey={itemKey} index={index} key={itemKey} gap="large">
      <Channel.AliasInput
        channel={line.yChannel}
        variant="shadow"
        value={line.label ?? ""}
        onChange={handleLabelChange}
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
        tooltip="Stroke Width"
      />
      <Input.Numeric
        variant="shadow"
        startContent={<Icon.Downsample />}
        value={line.downsample}
        onChange={handleDownsampleChange}
        dragScale={DOWNSAMPLE_DRAG_SCALE}
        bounds={DOWNSAMPLE_BOUNDS}
        shrink={false}
        tooltip={
          line.downsampleMode === "average" ? "Averaging Window" : "Downsampling Factor"
        }
      />
      <SelectDownsampleMode
        value={line.downsampleMode}
        onChange={handleDownsampleModeChange}
      />
      <Color.Swatch value={line.color} onChange={handleColorChange} size="small" />
    </List.Item>
  );
};
