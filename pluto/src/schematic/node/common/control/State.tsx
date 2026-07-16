// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, schematic } from "@synnaxlabs/client";
import { direction } from "@synnaxlabs/x";
import { useMemo } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Grid } from "@/schematic/node/common/grid";
import * as CommonTelem from "@/schematic/node/common/telem";
import { Control } from "@/telem/control";

export const stateConfigZ = schematic.controlStateConfigZ;
export type StateConfig = schematic.ControlStateConfig;

export interface StateProps extends StateConfig, Omit<Flex.BoxProps, "direction"> {
  chip?: Control.ChipProps;
  indicator?: Control.IndicatorProps;
}

export interface State {
  config?: StateConfig;
  channel?: channel.Key;
  onChange?: (next: { control: StateConfig }) => void;
}

interface InternalProps extends Flex.BoxProps {
  config: StateConfig;
  channel?: channel.Key;
}

const Internal = ({
  config: {
    show = true,
    showChip = true,
    showIndicator = true,
    authority,
    orientation = "bottom",
  },
  channel = 0,
  ...rest
}: InternalProps) => {
  const chip = useMemo(
    () => ({
      source: CommonTelem.chipStatusSource(channel),
      sink: CommonTelem.chipSink({ channel, authority }),
    }),
    [channel, authority],
  );
  const indicator = useMemo(
    () => ({ statusSource: CommonTelem.chipStatusSource(channel) }),
    [channel],
  );
  return (
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
};

export const State = Grid.createItem<State>(({ config, channel, onChange }) => {
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
      <Internal config={config} channel={channel} />
    </Grid.Item>
  );
});
State.displayName = "Control.State";
