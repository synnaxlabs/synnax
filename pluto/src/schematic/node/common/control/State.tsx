// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { direction } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Grid } from "@/schematic/node/common/grid";
import { Control } from "@/telem/control";

export const chipConfigZ = schematic.chipConfigZ;
export type ChipConfig = schematic.ChipConfig;

export const indicatorConfigZ = schematic.indicatorConfigZ;
export type IndicatorConfig = schematic.IndicatorConfig;

export const stateConfigZ = schematic.controlStateConfigZ;
export type StateConfig = schematic.ControlStateConfig;

export interface StateProps extends StateConfig, Omit<Flex.BoxProps, "direction"> {
  chip?: Control.ChipProps;
  indicator?: Control.IndicatorProps;
}

export interface State {
  config?: StateConfig;
  onChange?: (next: { control: StateConfig }) => void;
}

interface InternalProps extends Flex.BoxProps {
  config: StateConfig;
}

const Internal = ({
  config: {
    show = true,
    showChip = true,
    showIndicator = true,
    chip,
    indicator,
    orientation = "bottom",
  },
  ...rest
}: InternalProps) => (
  <Flex.Box
    direction={direction.swap(orientation)}
    align="center"
    className={CSS(CSS.B("control-state"))}
    gap="small"
    {...rest}
  >
    {show && showChip && <Control.Chip size="small" {...chip} />}
    {show && showIndicator && <Control.Indicator {...indicator} />}
  </Flex.Box>
);

export const State = Grid.createItem<State>(({ config, onChange }) => {
  if (config == null) return null;
  const orientation = config.orientation ?? "bottom";
  return (
    <Grid.Item
      itemKey="control"
      location={orientation}
      onLocationChange={(loc) =>
        onChange?.({ control: { ...config, orientation: loc } })
      }
    >
      <Internal config={config} />
    </Grid.Item>
  );
});
State.displayName = "Control.State";
