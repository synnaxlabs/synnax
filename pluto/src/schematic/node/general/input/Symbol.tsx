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
import { type ReactElement, useMemo } from "react";

import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { Telem } from "@/schematic/node/common/telem";
import { Input } from "@/schematic/node/general/input/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";
import { Input as InputTelem } from "@/vis/input";
import { Staleness } from "@/vis/staleness";
import { StringValue } from "@/vis/stringValue";

interface ReadbackProps {
  nodeKey: string;
  channel: channel.Key;
  size: Component.Size;
  stalenessColor?: color.Color;
  stalenessTimeout?: number;
}

const Readback = ({
  nodeKey,
  channel,
  size,
  stalenessColor,
  stalenessTimeout,
}: ReadbackProps): ReactElement => {
  const theme = Theming.use();
  const source = useMemo(() => telem.streamChannelStringValue({ channel }), [channel]);
  const { value, stale } = StringValue.use({
    aetherKey: `${nodeKey}-readback`,
    telem: source,
    stalenessTimeout,
  });
  return (
    <Primitive.Readout
      size={size}
      value={value}
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
    control,
    commandChannel,
    stateChannel,
    stalenessColor,
    stalenessTimeout,
    size,
    inlineSize,
    ...rest
  },
}: NodeProps<schematic.InputNodeConfig>): ReactElement => {
  const sink = useMemo(() => Telem.stringSink(commandChannel), [commandChannel]);
  const source = useMemo(
    () => telem.streamChannelStringValue({ channel: commandChannel ?? 0 }),
    [commandChannel],
  );
  const { set, value } = InputTelem.use({ aetherKey: nodeKey, sink, source });
  const resize = Grid.useInlineResize(inlineSize, (inlineSize) =>
    onConfigChange({ inlineSize }),
  );
  return (
    <Grid.Grid nodeKey={nodeKey} allowRotate={false} editable={selected} {...resize}>
      <Control.State
        config={control}
        channel={commandChannel}
        onChange={onConfigChange}
      />
      <Label.Label config={label} onChange={onConfigChange} />
      <Input
        onSend={set}
        value={value}
        size={size}
        inlineSize={inlineSize}
        readback={
          stateChannel != null &&
          stateChannel !== 0 && (
            <Readback
              nodeKey={nodeKey}
              channel={stateChannel}
              size={size}
              stalenessColor={stalenessColor}
              stalenessTimeout={stalenessTimeout}
            />
          )
        }
        {...rest}
      />
    </Grid.Grid>
  );
};
