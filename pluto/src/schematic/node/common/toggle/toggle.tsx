// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { type FC, memo, type ReactElement } from "react";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import type { Primitive } from "@/schematic/node/common/primitive";
import { Telem } from "@/schematic/node/common/telem";
import type { ButtonProps } from "@/schematic/node/common/toggle/Button";
import type { NodeProps } from "@/schematic/node/spec";
import { Staleness } from "@/vis/staleness";
import { Toggle as Base } from "@/vis/toggle";

export type ToggleConfig = schematic.ToggleConfig;

export const createToggle = <C extends ToggleConfig>(
  BaseSymbol: FC<Omit<C, "label"> & ButtonProps>,
  overrides?: {
    grid?: Partial<Omit<Grid.GridProps, "editable">>;
  },
): FC<NodeProps<C>> => {
  // BaseSymbol's prop type is derived from C so callers are checked, but the node only
  // renders it with the shared button props; the config's symbol-specific fields reach
  // it at runtime via the rest spread.
  const Sym = BaseSymbol as FC<ButtonProps & Primitive.SVGBasedProps>;
  const Inner = ({
    nodeKey,
    onConfigChange,
    selected,
    config,
  }: NodeProps<schematic.ToggleSymbolConfig>): ReactElement => {
    const {
      control,
      stateChannel,
      commandChannel,
      label,
      orientation = "left",
      onClickDelay = 0,
      stalenessTimeout,
      ...rest
    } = config;
    const source = Telem.booleanSource(stateChannel);
    const sink = Telem.booleanSink(commandChannel);
    const { enabled, toggle, stale } = Base.use({
      aetherKey: nodeKey,
      source,
      sink,
      stalenessTimeout,
    });
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
        <Control.State
          config={control}
          channels={[commandChannel]}
          onChange={onConfigChange}
        >
          {stale ? <Staleness.Badge /> : null}
        </Control.State>
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
