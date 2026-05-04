// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, direction, location, type TimeSpan, type xy } from "@synnaxlabs/x";
import { type FC, memo, type ReactElement } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Grid, type GridItem, type GridProps } from "@/schematic/node/common/grid/Grid";
import { Label } from "@/schematic/node/common/label";
import { Control } from "@/telem/control";
import { Toggle } from "@/vis/toggle";

export interface StateMapping {
  key: string;
  name: string;
  value: number;
  color?: color.Crude;
}

export interface ControlStateProps extends Omit<Flex.BoxProps, "direction"> {
  show?: boolean;
  showChip?: boolean;
  showIndicator?: boolean;
  chip?: Control.ChipProps;
  indicator?: Control.IndicatorProps;
  orientation?: location.Location;
}

export type NodeProps<Config extends object = object> = {
  nodeKey: string;
  selected: boolean;
  onConfigChange: (data: Partial<Config>) => void;
  config: Config;
  position?: xy.XY;
  draggable?: boolean;
};

export type PreviewProps<P extends object = object> = P & {
  scale?: number;
};

export const controlStateGridItem = (props?: ControlStateProps): GridItem | null => {
  if (props == null) return null;
  const {
    show = true,
    showChip = true,
    showIndicator = true,
    chip,
    indicator,
    orientation = "bottom",
  } = props;
  return {
    key: "control",
    element: (
      <Flex.Box
        direction={direction.swap(orientation)}
        align="center"
        className={CSS(CSS.B("control-state"))}
        gap="small"
      >
        {show && showChip && <Control.Chip size="small" {...chip} />}
        {show && showIndicator && <Control.Indicator {...indicator} />}
      </Flex.Box>
    ),
    location: orientation,
  };
};

export interface LabeledConfig {
  label?: Label.Config;
  orientation?: location.Outer;
}

export interface ToggleConfig
  extends LabeledConfig, Pick<Toggle.UseProps, "source" | "sink"> {
  control?: ControlStateProps;
  onClickDelay?: number | TimeSpan;
}

export interface DummyToggleConfig extends LabeledConfig {
  enabled?: boolean;
  clickable?: boolean;
}

export const createToggle = <C extends ToggleConfig>(
  BaseSymbol: FC<any>,
  overrides?: {
    grid?: Partial<Omit<GridProps, "editable">>;
  },
): FC<NodeProps<C>> => {
  const C = ({
    nodeKey: symbolKey,
    onConfigChange: onChange,
    selected,
    config: data,
  }: NodeProps<C>): ReactElement => {
    const {
      control,
      source,
      sink,
      label,
      orientation = "left",
      onClickDelay,
      ...rest
    } = data;
    const { enabled, toggle } = Toggle.use({ aetherKey: symbolKey, source, sink });
    const gridItems: GridItem[] = [];
    const labelItem = Label.gridItem(label, onChange as never);
    if (labelItem != null) gridItems.push(labelItem);
    const controlItem = controlStateGridItem(control);
    if (controlItem != null) gridItems.push(controlItem);
    return (
      <Grid
        editable={selected}
        symbolKey={symbolKey}
        items={gridItems}
        onRotate={() =>
          onChange({
            orientation: location.rotate(orientation, "clockwise"),
          } as Partial<C>)
        }
        onLocationChange={(key, loc) => {
          if (key === "label")
            onChange({ label: { ...label, orientation: loc } } as Partial<C>);
          if (key === "control")
            onChange({ control: { ...control, orientation: loc } } as Partial<C>);
        }}
        {...overrides?.grid}
      >
        <BaseSymbol
          enabled={enabled}
          onClick={toggle}
          onClickDelay={onClickDelay}
          orientation={orientation}
          {...rest}
        />
      </Grid>
    );
  };
  const M = memo(C) as unknown as FC<NodeProps<C>>;
  (M as { displayName?: string }).displayName = BaseSymbol.displayName;
  return M;
};

interface LabeledOverrides {
  grid: Partial<Omit<GridProps, "editable">>;
}

export const createLabeled = <C extends LabeledConfig>(
  BaseSymbol: FC<any>,
  overrides?: LabeledOverrides,
): FC<NodeProps<C>> => {
  const C = ({
    nodeKey: symbolKey,
    onConfigChange: onChange,
    selected,
    config: data,
  }: NodeProps<C>): ReactElement => {
    const { label, orientation = "left", ...rest } = data;
    const gridItems: GridItem[] = [];
    const labelItem = Label.gridItem(label, onChange as never);
    if (labelItem != null) gridItems.push(labelItem);
    return (
      <Grid
        {...overrides?.grid}
        items={gridItems}
        editable={selected}
        symbolKey={symbolKey}
        onRotate={() =>
          onChange({
            orientation: location.rotate(orientation, "clockwise"),
          } as Partial<C>)
        }
        onLocationChange={(key, loc) => {
          if (key === "label")
            onChange({
              label: { ...label, orientation: loc },
            } as Partial<C>);
        }}
      >
        <BaseSymbol orientation={orientation} {...rest} />
      </Grid>
    );
  };
  const M = memo(C) as unknown as FC<NodeProps<C>>;
  (M as { displayName?: string }).displayName = BaseSymbol.displayName;
  return M;
};

export const createDummyToggle = <C extends DummyToggleConfig>(
  Primitive: FC<any>,
): FC<NodeProps<C>> => {
  const DummyToggle = ({
    nodeKey: symbolKey,
    onConfigChange: onChange,
    selected,
    config: data,
  }: NodeProps<C>): ReactElement => {
    const {
      label,
      orientation = "left",
      enabled = false,
      clickable = false,
      ...rest
    } = data;
    const gridItems: GridItem[] = [];
    const labelItem = Label.gridItem(label, onChange as never);
    if (labelItem != null) gridItems.push(labelItem);
    const handleToggleChange = () => {
      if (!clickable) return;
      onChange({ enabled: !enabled } as Partial<C>);
    };
    return (
      <Grid
        items={gridItems}
        editable={selected}
        symbolKey={symbolKey}
        onRotate={() =>
          onChange({
            orientation: location.rotate(orientation, "clockwise"),
          } as Partial<C>)
        }
        onLocationChange={(key, loc) => {
          if (key === "label")
            onChange({
              label: { ...label, orientation: loc },
            } as Partial<C>);
        }}
      >
        <Primitive
          orientation={orientation}
          enabled={enabled}
          onClick={handleToggleChange}
          {...rest}
        />
      </Grid>
    );
  };
  const M = memo(DummyToggle) as unknown as FC<NodeProps<C>>;
  (M as { displayName?: string }).displayName = Primitive.displayName;
  return M;
};
