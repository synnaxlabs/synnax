// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type schematic } from "@synnaxlabs/client";
import { type Component } from "@synnaxlabs/lyra/component";
import { Theming } from "@synnaxlabs/lyra/theming";
import { type color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { Telem } from "@/schematic/node/common/telem";
import { optionName, Select } from "@/schematic/node/general/select/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Setpoint as BaseSetpoint } from "@/vis/setpoint";
import { Staleness } from "@/vis/staleness";
import { StateIndicator } from "@/vis/stateIndicator";

interface ReadbackProps {
  nodeKey: string;
  channel: channel.Key;
  size: Component.Size;
  options: schematic.SelectNodeConfig["options"];
  stalenessColor?: color.Color;
  stalenessTimeout?: number;
}

const Readback = ({
  nodeKey,
  channel,
  size,
  options,
  stalenessColor,
  stalenessTimeout,
}: ReadbackProps): ReactElement => {
  const theme = Theming.use();
  const source = Telem.numberSource(channel);
  const { key, stale } = StateIndicator.use({
    aetherKey: `${nodeKey}-readback`,
    source,
    options,
    stalenessTimeout,
  });
  const matched = options.find((o) => o.key === key);
  return (
    <Primitive.Readout
      size={size}
      value={matched == null ? "-" : optionName(matched)}
      color={stale ? Staleness.resolveColor(stalenessColor, theme) : undefined}
    />
  );
};

export const Symbol = ({
  nodeKey,
  onConfigChange,
  selected,
  config: {
    label,
    orientation = "left",
    control,
    fillColor,
    commandChannel,
    stateChannel,
    stalenessColor,
    stalenessTimeout,
    options,
    size,
    disabled,
    inlineSize,
    onClickDelay,
  },
}: NodeProps<schematic.SelectNodeConfig>): ReactElement => {
  const sink = Telem.numberSink(commandChannel);
  const source = Telem.numberSource(commandChannel);
  const { set, value } = BaseSetpoint.use({ aetherKey: nodeKey, sink, source });
  const resize = Grid.useInlineResize(inlineSize, (inlineSize) =>
    onConfigChange({ inlineSize }),
  );
  const target = options.find((o) => o.value === value)?.key ?? null;
  return (
    <Grid.Grid nodeKey={nodeKey} allowRotate={false} editable={selected} {...resize}>
      <Control.State
        config={control}
        channel={commandChannel}
        onChange={onConfigChange}
      />
      <Label.Label config={label} onChange={onConfigChange} />
      <Select
        value={target}
        onSend={set}
        fillColor={fillColor}
        orientation={orientation}
        disabled={disabled}
        options={options}
        size={size}
        inlineSize={inlineSize}
        onClickDelay={onClickDelay}
        readback={
          stateChannel != null &&
          stateChannel !== 0 && (
            <Readback
              nodeKey={nodeKey}
              channel={stateChannel}
              size={size}
              options={options}
              stalenessColor={stalenessColor}
              stalenessTimeout={stalenessTimeout}
            />
          )
        }
      />
    </Grid.Grid>
  );
};
