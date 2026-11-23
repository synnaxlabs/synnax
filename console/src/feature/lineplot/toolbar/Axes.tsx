// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Direction, LinePlot, Properties } from "@synnaxlabs/pluto";
import type { closedBounds, text } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { CSS } from "@/platform/css";

export const Axes = (): ReactElement => {
  const axisKeys = LinePlot.useAxisKeys();
  return (
    <Properties.Rail
      tabs={axisKeys.map((key) => ({
        key,
        name: LinePlot.axisLabel(key),
        icon: LinePlot.axisIcon(key),
      }))}
      initialValue={axisKeys[0]}
    >
      {axisKeys.map((key) => (
        <Tabs.Content key={key} itemKey={key}>
          <LinePlotAxisControls axisKey={key} />
        </Tabs.Content>
      ))}
    </Properties.Rail>
  );
};

export interface LinePlotAxisControlsProps {
  axisKey: lineplot.AxisKey;
}

const AXES_BOUNDS_DRAG_SCALE = { x: 0.1, y: 0.1 };

type Edge = keyof closedBounds.ClosedBounds;

/**
 * @returns the bounds after setting the edge manually, or undefined when the other edge
 * is manual and the pair would not hold a lower below its upper. An auto edge means
 * nothing to the plot, so it takes any value that keeps the pair valid.
 */
const setManualEdge = (
  { bounds, manualBounds }: lineplot.Axis,
  edge: Edge,
  value: number,
): closedBounds.ClosedBounds | undefined => {
  const next = {
    lower: bounds?.lower ?? value,
    upper: bounds?.upper ?? value,
    [edge]: value,
  };
  if (next.lower < next.upper) return next;
  if (edge === "lower" ? manualBounds.upper : manualBounds.lower) return undefined;
  return edge === "lower"
    ? { lower: value, upper: value + 1 }
    : { lower: value - 1, upper: value };
};

/** @returns the bounds after returning the given edge to auto. */
const setAutoEdge = (
  { bounds, manualBounds }: lineplot.Axis,
  edge: Edge,
): { bounds?: closedBounds.ClosedBounds; manualBounds: lineplot.ManualBounds } => {
  const nextManual = { ...manualBounds, [edge]: false };
  if (!nextManual.lower && !nextManual.upper) return { manualBounds: nextManual };
  return { bounds, manualBounds: nextManual };
};

interface BoundInputProps {
  value: number;
  manual: boolean;
  onChange: (value: number) => void;
  onAuto: () => void;
}

// An auto bound lives only in the plot, so the field shows no number while auto.
const BoundInput = ({
  value,
  manual,
  onChange,
  onAuto,
}: BoundInputProps): ReactElement => (
  <Input.Numeric
    value={manual ? value : undefined}
    placeholder="Enter a value to override"
    endContent={manual ? undefined : <Input.AutoBadge />}
    onChange={onChange}
    onClear={onAuto}
    resetValue={0}
    dragScale={AXES_BOUNDS_DRAG_SCALE}
  >
    {manual && <Input.AutoButton onClick={onAuto} />}
  </Input.Numeric>
);

const LinePlotAxisControls = ({ axisKey }: LinePlotAxisControlsProps): ReactElement => {
  const dispatch = LinePlot.useSingleDispatch();
  const axis = LinePlot.useAxis({ axisKey });

  const handleLabelChange: Input.Control<string>["onChange"] = (value) =>
    dispatch(lineplot.setAxisLabel({ key: axisKey, label: value }));

  const handleBoundChange =
    (edge: Edge): Input.Control<number>["onChange"] =>
    (value) => {
      const bounds = setManualEdge(axis, edge, value);
      // A manual edge that passes the other manual edge is refused.
      if (bounds == null) return;
      dispatch(
        lineplot.setAxisBounds({
          key: axisKey,
          bounds,
          manualBounds: { ...axis.manualBounds, [edge]: true },
        }),
      );
    };

  const handleAutoBoundEnable = (edge: Edge) => (): void =>
    dispatch(lineplot.setAxisBounds({ key: axisKey, ...setAutoEdge(axis, edge) }));

  const handleLabelDirectionChange: Input.Control<"x" | "y">["onChange"] = (value) =>
    dispatch(lineplot.setAxisLabelDirection({ key: axisKey, labelDirection: value }));

  const handleTickSpacingChange: Input.Control<number>["onChange"] = (value) =>
    dispatch(lineplot.setAxisTickSpacing({ key: axisKey, tickSpacing: value }));

  const handleLabelLevelChange: Input.Control<text.Level>["onChange"] = (value) =>
    dispatch(lineplot.setAxisLabelLevel({ key: axisKey, labelLevel: value }));

  return (
    <Flex.Box y className={CSS.BE("line-plot", "toolbar", "axes")} gap="small">
      <Flex.Box x>
        <Input.Item label="Lower bound" y grow>
          <BoundInput
            value={axis.bounds?.lower ?? 0}
            manual={axis.manualBounds.lower}
            onChange={handleBoundChange("lower")}
            onAuto={handleAutoBoundEnable("lower")}
          />
        </Input.Item>
        <Input.Item label="Upper bound" grow>
          <BoundInput
            value={axis.bounds?.upper ?? 0}
            manual={axis.manualBounds.upper}
            onChange={handleBoundChange("upper")}
            onAuto={handleAutoBoundEnable("upper")}
          />
        </Input.Item>
        <Input.Item label="Tick spacing" grow>
          <Input.Numeric
            resetValue={75}
            dragScale={AXES_BOUNDS_DRAG_SCALE}
            bounds={{ lower: 1, upper: 200 }}
            integer
            value={axis.tickSpacing}
            onChange={handleTickSpacingChange}
            endContent="px"
          />
        </Input.Item>
      </Flex.Box>
      <Flex.Box x>
        <Input.Item label="Label" grow>
          <Input.Text
            placeholder={axisKey.toUpperCase()}
            value={axis.label}
            onChange={handleLabelChange}
          />
        </Input.Item>
        {axis.key.startsWith("y") && (
          <Input.Item
            label="Label direction"
            className={CSS.BE("line-plot", "toolbar", "axes-label-direction")}
          >
            <Direction.Select
              value={axis.labelDirection}
              onChange={handleLabelDirectionChange}
            />
          </Input.Item>
        )}
        <Input.Item label="Label size">
          <Select.Text.Level
            value={axis.labelLevel}
            onChange={handleLabelLevelChange}
          />
        </Input.Item>
      </Flex.Box>
    </Flex.Box>
  );
};
