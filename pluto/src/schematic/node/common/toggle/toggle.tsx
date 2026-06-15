// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, memo, type ReactElement } from "react";
import { z } from "zod";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type ButtonProps } from "@/schematic/node/common/toggle/Button";
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
  BaseSymbol: FC<Omit<C, "label"> & ButtonProps>,
  overrides?: {
    grid?: Partial<Omit<Grid.GridProps, "editable">>;
  },
): FC<NodeProps<C>> => {
  // BaseSymbol's prop type is derived from C so callers are checked, but the node only
  // renders it with the shared button props; the config's symbol-specific fields reach
  // it at runtime via the rest spread.
  const Sym = BaseSymbol as FC<ButtonProps>;
  const Inner = ({
    nodeKey,
    onConfigChange,
    selected,
    config,
  }: NodeProps<ToggleConfig>): ReactElement => {
    const {
      control,
      source,
      sink,
      label,
      orientation = "left",
      onClickDelay,
      ...rest
    } = config;
    const { enabled, toggle } = Base.use({ aetherKey: nodeKey, source, sink });
    const scaleResize = Grid.useScaleResize(config, onConfigChange);
    return (
      <Grid.Grid
        editable={selected}
        nodeKey={nodeKey}
        orientation={orientation}
        onRotate={onConfigChange}
        {...overrides?.grid}
        {...scaleResize}
      >
        <Label.Label config={label} onChange={onConfigChange} />
        <Control.State config={control} onChange={onConfigChange} />
        <Sym
          enabled={enabled}
          onClick={toggle}
          onClickDelay={onClickDelay}
          orientation={orientation}
          {...rest}
        />
      </Grid.Grid>
    );
  };
  const M = memo(Inner) as unknown as FC<NodeProps<C>>;
  M.displayName = BaseSymbol.displayName;
  return M;
};

export const dummyToggleConfigZ = Label.labeledConfigZ.extend({
  enabled: z.boolean().optional(),
  clickable: z.boolean().optional(),
});
export type DummyToggleConfig = z.infer<typeof dummyToggleConfigZ>;

export const createDummyToggle = <C extends DummyToggleConfig>(
  Primitive: FC<Omit<C, "label"> & ButtonProps>,
): FC<NodeProps<C>> => {
  const Sym = Primitive as FC<ButtonProps>;
  const DummyToggle = ({
    nodeKey,
    onConfigChange,
    selected,
    config,
  }: NodeProps<DummyToggleConfig>): ReactElement => {
    const {
      label,
      orientation = "left",
      enabled = false,
      clickable = false,
      ...rest
    } = config;
    const handleToggleChange = () => {
      if (!clickable) return;
      onConfigChange({ enabled: !enabled });
    };
    const scaleResize = Grid.useScaleResize(config, onConfigChange);
    return (
      <Grid.Grid
        editable={selected}
        nodeKey={nodeKey}
        orientation={orientation}
        onRotate={onConfigChange}
        {...scaleResize}
      >
        <Label.Label config={label} onChange={onConfigChange} />
        <Sym
          orientation={orientation}
          enabled={enabled}
          onClick={handleToggleChange}
          {...rest}
        />
      </Grid.Grid>
    );
  };
  const M = memo(DummyToggle) as unknown as FC<NodeProps<C>>;
  M.displayName = Primitive.displayName;
  return M;
};
