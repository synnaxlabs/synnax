// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, schematic } from "@synnaxlabs/client";
import { CSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { direction } from "@synnaxlabs/x";
import { useMemo } from "react";

import { Grid } from "@/schematic/node/common/grid";
import { Telem } from "@/schematic/node/common/telem";
import { Control } from "@/telem/control";
import { control } from "@/telem/control/aether";

export const stateConfigZ = schematic.controlStateConfigZ;
export type StateConfig = schematic.ControlStateConfig;

/** reveal clears every hidden flag, so the control state shows once a command channel
 * is chosen. */
export const reveal = (config?: StateConfig): StateConfig =>
  stateConfigZ.parse({
    ...config,
    hidden: false,
    chipHidden: false,
    indicatorHidden: false,
  });

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
  const chip = useMemo(
    () => ({
      source: control.authoritySource({ channel }),
      sink: Telem.chipSink({ channel, authority }),
    }),
    [channel, authority],
  );
  const indicator = useMemo(
    () => ({ statusSource: control.authoritySource({ channel }) }),
    [channel],
  );
  return (
    <Flex.Box
      direction={direction.swap(orientation)}
      align="center"
      className={CSS.cls(CSS.B("control-state"))}
      gap="small"
      {...rest}
    >
      {!hidden && !chipHidden && <Control.Chip size="small" {...chip} />}
      {!hidden && !indicatorHidden && <Control.Indicator {...indicator} />}
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
