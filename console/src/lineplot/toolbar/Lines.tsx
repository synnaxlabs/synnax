// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Channel,
  Color,
  Icon,
  Input,
  List,
  Select,
  Tabs,
  type telem,
} from "@synnaxlabs/pluto";
import { type bounds, color, type xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { EmptyAction } from "@/components";
import { CSS } from "@/css";
import { useSelectLine, useSelectLineKeys } from "@/lineplot/selectors";
import { type LineState, setLine, typedLineKeyFromString } from "@/lineplot/slice";

export interface LinesProps {
  layoutKey: string;
}

export const Lines = ({ layoutKey }: LinesProps): ReactElement => {
  const lineKeys = useSelectLineKeys(layoutKey);
  const dispatch = useDispatch();

  const handleChange = (line: LineState): void => {
    dispatch(setLine({ key: layoutKey, line }));
  };

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
      <List.Items<string, LineState>
        full="y"
        className={CSS.BE("line-plot", "toolbar", "lines")}
        emptyContent={emptyContent}
      >
        {({ key, ...rest }) => (
          <Line key={key} layoutKey={layoutKey} onChange={handleChange} {...rest} />
        )}
      </List.Items>
    </List.Frame>
  );
};

interface LinePlotLineControlsProps extends Omit<List.ItemProps<string>, "onChange"> {
  layoutKey: string;
  onChange: (line: LineState) => void;
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

const Line = ({
  itemKey,
  onChange,
  layoutKey,
}: LinePlotLineControlsProps): ReactElement | null => {
  const line = useSelectLine(layoutKey, itemKey);
  if (line == null) return null;
  const handleLabelChange: Input.Control<string>["onChange"] = (value: string) => {
    onChange({ ...line, label: value });
  };

  const handleWidthChange: Input.Control<number>["onChange"] = (value: number) => {
    onChange({ ...line, strokeWidth: value });
  };

  const handleDownsampleChange: Input.Control<number>["onChange"] = (value: number) => {
    onChange({ ...line, downsample: value });
  };

  const handleDownsampleModeChange: Select.ButtonsProps<telem.DownsampleMode>["onChange"] =
    (value: telem.DownsampleMode) => {
      onChange({ ...line, downsampleMode: value });
    };

  const handleColorChange: Input.Control<color.Color>["onChange"] = (
    value: color.Color,
  ) => {
    onChange({ ...line, color: color.hex(value) });
  };

  const {
    channels: { y: yChannel },
  } = typedLineKeyFromString(line.key);

  return (
    <List.Item itemKey={itemKey} index={0} key={itemKey} gap="large">
      <Channel.AliasInput
        channel={yChannel}
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
        value={line.downsample ?? 1}
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
