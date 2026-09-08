// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { direction, location } from "@synnaxlabs/x";
import { z } from "zod";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Grid } from "@/schematic/node/common/grid";
import { telem } from "@/telem/aether";
import { Control } from "@/telem/control";

export const chipConfigZ = z.object({
  source: telem.statusSourceSpecZ.optional(),
  sink: telem.booleanSinkSpecZ.optional(),
});
export type ChipConfig = z.infer<typeof chipConfigZ>;

export const indicatorConfigZ = z.object({
  statusSource: telem.statusSourceSpecZ.optional(),
  colorSource: telem.colorSourceSpecZ.optional(),
});
export type IndicatorConfig = z.infer<typeof indicatorConfigZ>;

export const stateConfigZ = z.object({
  show: z.boolean().optional(),
  showChip: z.boolean().optional(),
  showIndicator: z.boolean().optional(),
  chip: chipConfigZ.optional(),
  indicator: indicatorConfigZ.optional(),
  orientation: location.locationZ.optional(),
});
export type StateConfig = z.infer<typeof stateConfigZ>;

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
    className={CSS.cls(CSS.B("control-state"))}
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
