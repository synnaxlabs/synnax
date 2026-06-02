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
  compareArrayDeps,
  Direction,
  Flex,
  Icon,
  Input,
  LinePlot as PLinePlot,
  Select,
  Tabs,
  useMemoCompare,
} from "@synnaxlabs/pluto";
import { type text } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

export interface AxesProps {
  layoutKey: string;
}

const shouldDisplay = (key: lineplot.AxisKey, channels: lineplot.Channels): boolean => {
  if (key === "x1" || key === "y1") return true;
  if (key === "x2") return channels.x2 !== 0;
  return channels[key].length > 0;
};

export const Axes = ({ layoutKey }: AxesProps): ReactElement => {
  const channels = PLinePlot.useSelectChannels({ key: layoutKey });
  const shouldShow = lineplot.AXIS_KEYS.filter((k) => shouldDisplay(k, channels));

  const tabs = useMemoCompare(
    () => shouldShow.map((key) => ({ tabKey: key, name: key.toUpperCase() })),
    compareArrayDeps,
    [shouldShow] as [string[]],
  );

  const t = Tabs.useStatic({ tabs });

  return (
    <Tabs.Tabs {...t} size="small">
      {(p) => (
        <LinePlotAxisControls
          key={p.tabKey}
          axisKey={p.tabKey as lineplot.AxisKey}
          layoutKey={layoutKey}
        />
      )}
    </Tabs.Tabs>
  );
};

export interface LinePlotAxisControlsProps {
  axisKey: lineplot.AxisKey;
  layoutKey: string;
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
  layoutKey,
}: LinePlotAxisControlsProps): ReactElement => {
  const { dispatch } = PLinePlot.useDispatch();
  const axis = PLinePlot.useSelectAxis({ key: layoutKey, axisKey });

  const update = useCallback(
    (next: lineplot.Axis): void => {
      dispatch({ key: layoutKey, actions: [lineplot.setAxis({ axis: next })] });
    },
    [dispatch, layoutKey],
  );

  const handleLabelChange: Input.Control<string>["onChange"] = (value) =>
    update({ ...axis, label: value });

  const handleLowerBoundChange: Input.Control<number>["onChange"] = (value) =>
    update({
      ...axis,
      bounds: { ...axis.bounds, lower: value },
      autoBounds: { ...axis.autoBounds, lower: false },
    });

  const handleLowerAutoBoundEnable = (): void =>
    update({ ...axis, autoBounds: { ...axis.autoBounds, lower: true } });

  const handleUpperBoundChange: Input.Control<number>["onChange"] = (value) =>
    update({
      ...axis,
      bounds: { ...axis.bounds, upper: value },
      autoBounds: { ...axis.autoBounds, upper: false },
    });

  const handleUpperAutoBoundEnable = (): void =>
    update({ ...axis, autoBounds: { ...axis.autoBounds, upper: true } });

  const handleLabelDirectionChange: Input.Control<"x" | "y">["onChange"] = (value) =>
    update({ ...axis, labelDirection: value });

  const handleTickSpacingChange: Input.Control<number>["onChange"] = (value) =>
    update({ ...axis, tickSpacing: value });

  const handleLabelLevelChange: Input.Control<text.Level>["onChange"] = (value) =>
    update({ ...axis, labelLevel: value });

  return (
    <Flex.Box y style={{ padding: "2rem" }} gap="small">
      <Flex.Box x>
        <Input.Item label="Lower Bound" y grow>
          <Input.Numeric
            value={axis.bounds.lower}
            onChange={handleLowerBoundChange}
            resetValue={0}
            dragScale={AXES_BOUNDS_DRAG_SCALE}
          >
            <AutoBoundButton
              enabled={axis.autoBounds.lower}
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
              enabled={axis.autoBounds.upper}
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
          <Input.Item label="Label Direction" style={{ minWidth: 90 }}>
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
