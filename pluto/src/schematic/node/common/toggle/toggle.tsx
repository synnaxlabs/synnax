// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x";
import { type FC, memo, type ReactElement } from "react";
import { z } from "zod";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type NodeProps } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { Toggle as Base } from "@/vis/toggle";

export const toggleConfigZ = Label.labeledConfigZ.extend({
  source: telem.booleanSourceSpecZ.optional(),
  sink: telem.booleanSinkSpecZ.optional(),
  control: Control.stateConfigZ.optional(),
  onClickDelay: z.number().optional(),
});
export type ToggleConfig = z.infer<typeof toggleConfigZ>;

export const ZERO_BOOLEAN_SOURCE = telem.sourcePipeline("boolean", {
  connections: [{ from: "valueStream", to: "threshold" }],
  segments: {
    valueStream: telem.streamChannelValue({ channel: 0 }),
    threshold: telem.withinBounds({ trueBound: { lower: 0.9, upper: 1.1 } }),
  },
  outlet: "threshold",
});

export const ZERO_BOOLEAN_SINK = telem.sinkPipeline("boolean", {
  connections: [{ from: "setpoint", to: "setter" }],
  segments: {
    setter: control.setChannelValue({ channel: 0 }),
    setpoint: telem.setpoint({ truthy: 1, falsy: 0 }),
  },
  inlet: "setpoint",
});

export const ZERO_TOGGLE_DEFAULTS: Partial<ToggleConfig> = {
  source: ZERO_BOOLEAN_SOURCE,
  sink: ZERO_BOOLEAN_SINK,
  control: { show: true },
  onClickDelay: 0,
};

export const ZERO_DUMMY_TOGGLE_DEFAULTS: Partial<DummyToggleConfig> = {
  enabled: false,
  clickable: false,
};

export const createToggle = <C extends ToggleConfig>(
  BaseSymbol: FC<any>,
  overrides?: {
    grid?: Partial<Omit<Grid.GridProps, "editable">>;
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
    const { enabled, toggle } = Base.use({ aetherKey: symbolKey, source, sink });
    const gridItems: Grid.Item[] = [];
    const labelItem = Label.gridItem(label, onChange as never);
    if (labelItem != null) gridItems.push(labelItem);
    const controlItem = Control.stateGridItem(control);
    if (controlItem != null) gridItems.push(controlItem);
    return (
      <Grid.Grid
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
      </Grid.Grid>
    );
  };
  const M = memo(C);
  M.displayName = BaseSymbol.displayName;
  return M;
};

export const dummyToggleConfigZ = Label.labeledConfigZ.extend({
  enabled: z.boolean().optional(),
  clickable: z.boolean().optional(),
});
export type DummyToggleConfig = z.infer<typeof dummyToggleConfigZ>;

export const createDummyToggle = <Config extends DummyToggleConfig>(
  Primitive: FC<any>,
): FC<NodeProps<Config>> => {
  const DummyToggle = ({
    nodeKey: symbolKey,
    onConfigChange: onChange,
    selected,
    config: data,
  }: NodeProps<Config>): ReactElement => {
    const {
      label,
      orientation = "left",
      enabled = false,
      clickable = false,
      ...rest
    } = data;
    const gridItems: Grid.Item[] = [];
    const labelItem = Label.gridItem(label, onChange as never);
    if (labelItem != null) gridItems.push(labelItem);
    const handleToggleChange = () => {
      if (!clickable) return;
      onChange({ enabled: !enabled } as Partial<Config>);
    };
    return (
      <Grid.Grid
        items={gridItems}
        editable={selected}
        symbolKey={symbolKey}
        onRotate={() =>
          onChange({
            orientation: location.rotate(orientation, "clockwise"),
          } as Partial<Config>)
        }
        onLocationChange={(key, loc) => {
          if (key === "label")
            onChange({
              label: { ...label, orientation: loc },
            } as Partial<Config>);
        }}
      >
        <Primitive
          orientation={orientation}
          enabled={enabled}
          onClick={handleToggleChange}
          {...rest}
        />
      </Grid.Grid>
    );
  };
  const M = memo(DummyToggle);
  M.displayName = Primitive.displayName;
  return M;
};
