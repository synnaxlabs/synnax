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
  Button,
  Direction,
  Flex,
  Icon,
  Input,
  LinePlot,
  Select,
  Tabs,
} from "@synnaxlabs/pluto";
import { type text } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { CSS } from "@/primitive/css";

export const Axes = (): ReactElement => {
  const axisKeys = LinePlot.useSelectAxisKeys();
  const tabs = useMemo(
    () => axisKeys.map((key) => ({ tabKey: key, name: key.toUpperCase() })),
    [axisKeys],
  );

  const t = Tabs.useStatic({ tabs });

  return (
    <Tabs.Tabs {...t} size="small">
      {(p) => (
        <LinePlotAxisControls key={p.tabKey} axisKey={p.tabKey as lineplot.AxisKey} />
      )}
    </Tabs.Tabs>
  );
};

export interface LinePlotAxisControlsProps {
  axisKey: lineplot.AxisKey;
}

interface AutoBoundButtonProps extends Omit<Button.ButtonProps, "children"> {
  enabled: boolean;
}

const AutoBoundButton = ({ enabled, ...rest }: AutoBoundButtonProps): ReactElement => (
  <Button.Button
    {...rest}
    variant="outlined"
    disabled={enabled}
    tooltip={
      enabled ? "Manually enter value to disable auto bounding" : "Enable auto bounding"
    }
  >
    <Icon.Auto />
  </Button.Button>
);

const AXES_BOUNDS_DRAG_SCALE = { x: 0.1, y: 0.1 };

export const LinePlotAxisControls = ({
  axisKey,
}: LinePlotAxisControlsProps): ReactElement => {
  const dispatch = LinePlot.useSingleDispatch();
  const axis = LinePlot.useSelectAxis({ axisKey });

  const handleLabelChange: Input.Control<string>["onChange"] = (value) =>
    dispatch(lineplot.setAxisLabel({ key: axisKey, label: value }));

  const handleLowerBoundChange: Input.Control<number>["onChange"] = (value) =>
    dispatch(
      lineplot.setAxisBounds({
        key: axisKey,
        bounds: { ...axis.bounds, lower: value },
        manualBounds: { ...axis.manualBounds, lower: true },
      }),
    );

  const handleLowerAutoBoundEnable = (): void =>
    dispatch(
      lineplot.setAxisBounds({
        key: axisKey,
        bounds: axis.bounds,
        manualBounds: { ...axis.manualBounds, lower: false },
      }),
    );

  const handleUpperBoundChange: Input.Control<number>["onChange"] = (value) =>
    dispatch(
      lineplot.setAxisBounds({
        key: axisKey,
        bounds: { ...axis.bounds, upper: value },
        manualBounds: { ...axis.manualBounds, upper: true },
      }),
    );

  const handleUpperAutoBoundEnable = (): void =>
    dispatch(
      lineplot.setAxisBounds({
        key: axisKey,
        bounds: axis.bounds,
        manualBounds: { ...axis.manualBounds, upper: false },
      }),
    );

  const handleLabelDirectionChange: Input.Control<"x" | "y">["onChange"] = (value) =>
    dispatch(lineplot.setAxisLabelDirection({ key: axisKey, labelDirection: value }));

  const handleTickSpacingChange: Input.Control<number>["onChange"] = (value) =>
    dispatch(lineplot.setAxisTickSpacing({ key: axisKey, tickSpacing: value }));

  const handleLabelLevelChange: Input.Control<text.Level>["onChange"] = (value) =>
    dispatch(lineplot.setAxisLabelLevel({ key: axisKey, labelLevel: value }));

  return (
    <Flex.Box y className={CSS.BE("line-plot", "toolbar", "axes")} gap="small">
      <Flex.Box x>
        <Input.Item label="Lower Bound" y grow>
          <Input.Numeric
            value={axis.bounds.lower}
            onChange={handleLowerBoundChange}
            resetValue={0}
            dragScale={AXES_BOUNDS_DRAG_SCALE}
          >
            <AutoBoundButton
              enabled={!axis.manualBounds.lower}
              onClick={handleLowerAutoBoundEnable}
            />
          </Input.Numeric>
        </Input.Item>
        <Input.Item label="Upper Bound" grow>
          <Input.Numeric
            value={axis.bounds.upper}
            onChange={handleUpperBoundChange}
            resetValue={0}
            dragScale={AXES_BOUNDS_DRAG_SCALE}
          >
            <AutoBoundButton
              enabled={!axis.manualBounds.upper}
              onClick={handleUpperAutoBoundEnable}
            />
          </Input.Numeric>
        </Input.Item>
        <Input.Item label="Tick Spacing" grow>
          <Input.Numeric
            resetValue={75}
            dragScale={AXES_BOUNDS_DRAG_SCALE}
            bounds={{ lower: 1, upper: 200 }}
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
            label="Label Direction"
            className={CSS.BE("line-plot", "toolbar", "axes-label-direction")}
          >
            <Direction.Select
              value={axis.labelDirection}
              onChange={handleLabelDirectionChange}
            />
          </Input.Item>
        )}
        <Input.Item label="Label Size">
          <Select.Text.Level
            value={axis.labelLevel}
            onChange={handleLabelLevelChange}
          />
        </Input.Item>
      </Flex.Box>
    </Flex.Box>
  );
};
