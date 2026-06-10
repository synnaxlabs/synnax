// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type FC, memo, type ReactElement, useMemo } from "react";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import * as CommonTelem from "@/schematic/node/common/telem";
import { type ButtonProps } from "@/schematic/node/common/toggle/Button";
import { type NodeProps } from "@/schematic/node/spec";
import { Toggle as Base } from "@/vis/toggle";

export const toggleConfigZ = schematic.toggleConfigZ;
export type ToggleConfig = schematic.ToggleConfig;

export const ZERO_TOGGLE_DEFAULTS: Partial<ToggleConfig> = {
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
    config: {
      control,
      stateChannel,
      commandChannel,
      label,
      orientation = "left",
      onClickDelay,
      ...rest
    },
  }: NodeProps<ToggleConfig>): ReactElement => {
    const source = useMemo(
      () => CommonTelem.booleanSource(stateChannel),
      [stateChannel],
    );
    const sink = useMemo(
      () => CommonTelem.booleanSink(commandChannel),
      [commandChannel],
    );
    const { enabled, toggle } = Base.use({ aetherKey: nodeKey, source, sink });
    return (
      <Grid.Grid
        editable={selected}
        nodeKey={nodeKey}
        orientation={orientation}
        onRotate={onConfigChange}
        {...overrides?.grid}
      >
        <Label.Label config={label} onChange={onConfigChange} />
        <Control.State
          config={control}
          channel={commandChannel}
          onChange={onConfigChange}
        />
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

export type DummyToggleConfig = Omit<
  schematic.DummyToggleSymbolConfig,
  "color" | "scale"
>;

export const createDummyToggle = <C extends DummyToggleConfig>(
  Primitive: FC<Omit<C, "label"> & ButtonProps>,
): FC<NodeProps<C>> => {
  const Sym = Primitive as FC<ButtonProps>;
  const DummyToggle = ({
    nodeKey,
    onConfigChange,
    selected,
    config: {
      label,
      orientation = "left",
      enabled = false,
      clickable = false,
      ...rest
    },
  }: NodeProps<DummyToggleConfig>): ReactElement => {
    const handleToggleChange = () => {
      if (!clickable) return;
      onConfigChange({ enabled: !enabled });
    };
    return (
      <Grid.Grid
        editable={selected}
        nodeKey={nodeKey}
        orientation={orientation}
        onRotate={onConfigChange}
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
