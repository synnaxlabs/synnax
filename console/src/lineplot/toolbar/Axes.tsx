// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  compareArrayDeps,
  Direction,
  Flex,
  Icon,
  Input,
  Select,
  Tabs,
  type Text,
  useMemoCompare,
} from "@synnaxlabs/pluto";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { type AxisKey } from "@/lineplot/axis";
import { useSelect } from "@/lineplot/selectors";
import { type AxisState, setAxis, shouldDisplayAxis } from "@/lineplot/slice";

export interface AxesProps {
  layoutKey: string;
}

export const Axes = ({ layoutKey }: AxesProps): ReactElement => {
  const vis = useSelect(layoutKey);

  const shouldShow = Object.values(vis.axes.axes)
    .filter((a) => shouldDisplayAxis(a.key, vis))
    .map((a) => a.key);

  const tabs = useMemoCompare(
    () =>
      shouldShow.map((key) => ({
        tabKey: key,
        name: key.toUpperCase(),
      })),
    compareArrayDeps,
    [shouldShow] as [string[]],
  );

  const t = Tabs.useStatic({
    tabs,
  });

  return (
    <Tabs.Tabs {...t} size="small">
      {(p) => (
        <LinePlotAxisControls
          key={p.tabKey}
          axisKey={p.tabKey as AxisKey}
          layoutKey={layoutKey}
        />
      )}
    </Tabs.Tabs>
  );
};

export interface LinePlotAxisControlsProps {
  axisKey: AxisKey;
  layoutKey: string;
}

export interface AutoBoundButtonProps extends Omit<Button.ButtonProps, "children"> {
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

export const LinePlotAxisControls = ({
  axisKey,
  layoutKey,
}: LinePlotAxisControlsProps): ReactElement => {
  const dispatch = useDispatch();
  const axis = useSelect(layoutKey).axes.axes[axisKey];

  const handleChange = (axis: AxisState): void => {
    dispatch(setAxis({ key: layoutKey, axisKey, axis }));
  };

  const handleLabelChange: Input.Control<string>["onChange"] = (value: string) => {
    handleChange({ ...axis, label: value });
  };

  const handleLowerBoundChange: Input.Control<number>["onChange"] = (value: number) => {
    handleChange({
      ...axis,
      bounds: { ...axis.bounds, lower: value },
      autoBounds: { ...axis.autoBounds, lower: false },
    });
  };

  const handleLowerAutoBoundEnable = (): void => {
    handleChange({
      ...axis,
      autoBounds: { ...axis.autoBounds, lower: true },
    });
  };

  const handleUpperBoundChange: Input.Control<number>["onChange"] = (value: number) => {
    handleChange({
      ...axis,
      bounds: {
        ...axis.bounds,
        upper: value,
      },
      autoBounds: { ...axis.autoBounds, upper: false },
    });
  };

  const handleUpperAutoBoundEnable = (): void => {
    handleChange({
      ...axis,
      autoBounds: { ...axis.autoBounds, upper: true },
    });
  };

  const handleLabelDirectionChange: Input.Control<"x" | "y">["onChange"] = (value) => {
    handleChange({ ...axis, labelDirection: value });
  };

  const handleTickSpacingChange: Input.Control<number>["onChange"] = (value) => {
    handleChange({ ...axis, tickSpacing: value });
  };

  const handleLabelLevelChange: Input.Control<Text.Level>["onChange"] = (value) => {
    handleChange({ ...axis, labelLevel: value });
  };

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

const AXES_BOUNDS_DRAG_SCALE = {
  x: 0.1,
  y: 0.1,
};
