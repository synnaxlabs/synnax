// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, schematic } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { direction, primitive } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { Grid } from "@/schematic/node/common/grid";
import { Telem } from "@/schematic/node/common/telem";
import { Control } from "@/telem/control";
import { control } from "@/telem/control/aether";

export const stateConfigZ = schematic.controlStateConfigZ;
export type StateConfig = schematic.ControlStateConfig;

export const STATE_CLASS = CSS.B("control-state");

const DEFAULT_CONFIG = stateConfigZ.parse({});

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
  config: { hidden, chipHidden, indicatorHidden, authority, orientation },
  channel = 0,
  ...rest
}: InternalProps) => {
  const chip = {
    source: control.authoritySource({ channel }),
    sink: Telem.chipSink({ channel, authority }),
  };
  const indicator = { statusSource: control.authoritySource({ channel }) };
  return (
    <Flex.Box
      direction={direction.swap(orientation)}
      align="center"
      className={STATE_CLASS}
      gap="small"
      {...rest}
    >
      {!hidden && !chipHidden && <Control.Chip size="small" {...chip} />}
      {!hidden && !indicatorHidden && <Control.Indicator {...indicator} />}
    </Flex.Box>
  );
};

export const State = Grid.createItem<State>(({ config, channel, onChange }) => {
  // The chip and the indicator report control of the channel, so with no channel
  // there is nothing to show.
  if (!primitive.isNonZero(channel)) return null;
  config ??= DEFAULT_CONFIG;
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
