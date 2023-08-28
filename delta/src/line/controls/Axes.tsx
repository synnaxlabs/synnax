// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Input, Select, Align } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { AxisKey } from "@/vis";

import { useSelectLinePlot } from "../store/selectors";
import { AxisState, setLinePlotAxis, shouldDisplayAxis } from "../store/slice";

export interface AxesProps {
  layoutKey: string;
}

export const Axes = ({ layoutKey }: AxesProps): ReactElement => {
  const vis = useSelectLinePlot(layoutKey);
  const dispatch = useDispatch();

  const handleAxisChange = (key: AxisKey) => (axis: AxisState) => {
    dispatch(setLinePlotAxis({ key: layoutKey, axisKey: key, axis }));
  };

  return (
    <Align.Space style={{ padding: "2rem", width: "100%" }}>
      {Object.entries(vis.axes)
        .filter(([key]) => shouldDisplayAxis(key as AxisKey, vis))
        .map(([key, axis]) => (
          <LinePlotAxisControls
            key={key}
            axis={axis}
            axisKey={key as AxisKey}
            onChange={handleAxisChange(key as AxisKey)}
          />
        ))}
    </Align.Space>
  );
};

export interface LinePlotAxisControlsProps {
  axisKey: AxisKey;
  axis: AxisState;
  onChange: (axis: AxisState) => void;
}

export const LinePlotAxisControls = ({
  axisKey,
  axis,
  onChange,
}: LinePlotAxisControlsProps): ReactElement => {
  const handleLabelChange: Input.Control<string>["onChange"] = (value: string) => {
    onChange({ ...axis, label: value });
  };

  const handleLowerBoundChange: Input.Control<number>["onChange"] = (value: number) => {
    onChange({ ...axis, bounds: { ...axis.bounds, lower: value } });
  };

  const handleUpperBoundChange: Input.Control<number>["onChange"] = (value: number) => {
    onChange({ ...axis, bounds: { ...axis.bounds, upper: value } });
  };

  const handleLabelDirectionChange: Input.Control<"x" | "y">["onChange"] = (value) => {
    onChange({ ...axis, labelDirection: value });
  };

  return (
    <Align.Space direction="x">
      <Input.Text
        value={axis.label}
        placeholder={axisKey.toUpperCase()}
        onChange={handleLabelChange}
      />
      <Input.Numeric
        value={axis.bounds.lower}
        onChange={handleLowerBoundChange}
        resetValue={0}
        dragScale={AXES_BOUNDS_DRAG_SCALE}
      />
      <Input.Numeric
        value={axis.bounds.upper}
        onChange={handleUpperBoundChange}
        resetValue={0}
        dragScale={AXES_BOUNDS_DRAG_SCALE}
      />
      <Select.Direction
        value={axis.labelDirection}
        onChange={handleLabelDirectionChange}
      />
    </Align.Space>
  );
};

const AXES_BOUNDS_DRAG_SCALE = {
  x: 0.1,
  y: 0.1,
};
