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
import { control as xControl, direction, primitive } from "@synnaxlabs/x";
import type { ReactNode } from "react";

import { CSS } from "@/css";
import { Grid } from "@/schematic/node/common/grid";
import { Control } from "@/telem/control";
import { control } from "@/telem/control/aether";

export const stateConfigZ = schematic.controlStateConfigZ;
export type StateConfig = schematic.ControlStateConfig;

export const STATE_CLASS = CSS.B("control-state");

export interface State {
  config: StateConfig;
  /** channels are the channels that the chip acquires and the indicator reports. */
  channels: channel.Key[];
  onChange?: (next: { control: StateConfig }) => void;
  /**
   * True when the Core deleted a channel of the symbol. The chip and the indicator
   * hide.
   */
  deleted?: boolean;
  /** Shown after the chip and the indicator, even with no channel. */
  children?: ReactNode;
}

interface InternalProps extends Flex.BoxProps {
  config: StateConfig;
  channels: channel.Key[];
  deleted: boolean;
}

const Internal = ({
  config: { hidden, chipHidden, indicatorHidden, authority, orientation },
  channels,
  deleted,
  children,
  ...rest
}: InternalProps) => {
  const controlShown = !hidden && !deleted && channels.length > 0;
  const chip = {
    source: control.authoritySource({ channels }),
    sink: control.acquireChannelControl({
      channels,
      authority: authority ?? xControl.ABSOLUTE_AUTHORITY,
    }),
  };
  const indicator = { statusSource: control.authoritySource({ channels }) };
  return (
    <Flex.Box
      direction={direction.swap(orientation)}
      align="center"
      className={STATE_CLASS}
      gap="small"
      {...rest}
    >
      {controlShown && !chipHidden && <Control.Chip size="small" {...chip} />}
      {controlShown && !indicatorHidden && <Control.Indicator {...indicator} />}
      {children}
    </Flex.Box>
  );
};

export const State = Grid.createItem<State>(
  ({ config, channels: all, onChange, deleted = false, children }) => {
    // The chip and the indicator report control of the channels, so with none, or with
    // a deleted one, only the children show.
    const channels = all.filter(primitive.isNonZero);
    if ((channels.length === 0 || deleted) && children == null) return null;
    return (
      <Grid.Item
        itemKey="control"
        location={config.orientation}
        onLocationChange={(loc) =>
          onChange?.({ control: { ...config, orientation: loc } })
        }
      >
        <Internal config={config} channels={channels} deleted={deleted}>
          {children}
        </Internal>
      </Grid.Item>
    );
  },
);
State.displayName = "Control.State";
