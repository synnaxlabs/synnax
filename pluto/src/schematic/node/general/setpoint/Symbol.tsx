// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel, schematic } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { useResize } from "@synnaxlabs/lyra/hooks";
import { Theming } from "@synnaxlabs/lyra/theming";
import { box, type color, xy } from "@synnaxlabs/x";
import { type ReactElement, type RefCallback, useState } from "react";

import { Color } from "@/color";
import { Control } from "@/schematic/node/common/control";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { Telem } from "@/schematic/node/common/telem";
import { Setpoint } from "@/schematic/node/general/setpoint/Primitive";
import type { NodeProps } from "@/schematic/node/spec";
import { Setpoint as BaseSetpoint } from "@/vis/setpoint";
import { Staleness } from "@/vis/staleness";
import { Value as BaseValue } from "@/vis/value";

// READOUT_INSET is the readout's margin, border, and padding.
const READOUT_INSET = 10;

interface UseReadbackProps {
  nodeKey: string;
  position?: xy.XY;
  channel: channel.Key;
  size: Component.Size;
  stalenessColor?: color.Color;
  stalenessTimeout?: number;
  staleColorHidden: boolean;
}

interface UseReadbackReturn {
  stale: boolean;
  valueRef: RefCallback<HTMLParagraphElement>;
}

// useReadback draws the actual value on the canvas, so samples never re-render React.
// The offset moves the canvas text inset onto the readout's padding, under the field.
const useReadback = ({
  nodeKey,
  position,
  channel,
  size,
  stalenessColor,
  stalenessTimeout,
  staleColorHidden,
}: UseReadbackProps): UseReadbackReturn => {
  const theme = Theming.use();
  const level = Primitive.READOUT_LEVELS[size];
  const fontHeight = theme.typography[level].size * theme.sizes.base;
  const textInset = 6 + fontHeight * 0.75;
  const offset = xy.construct(READOUT_INSET - textInset, Component.HEIGHTS[size]);
  // offsetWidth ignores the diagram zoom, so it is in the same units as the box.
  const [cellWidth, setCellWidth] = useState(0);
  const valueRef = useResize<HTMLParagraphElement>((_, el) =>
    setCellWidth(el.offsetWidth),
  );
  const telem = BaseValue.stringSource({ channel });
  const { stale } = BaseValue.use({
    aetherKey: `${nodeKey}-readback`,
    level,
    color: theme.colors.gray.l10,
    // The box ends at the value cell's right edge, and the clip keeps a long value
    // inside it.
    box: box.construct(xy.translate(position ?? xy.ZERO, offset), {
      height: Primitive.readoutHeight(size),
      width: textInset + cellWidth,
    }),
    clip: true,
    telem,
    minWidth: 0,
    stalenessColor,
    stalenessTimeout,
    staleColorHidden,
  });
  return { stale, valueRef };
};

export const Symbol = ({
  nodeKey,
  position,
  onConfigChange,
  selected,
  config: {
    label,
    orientation = "left",
    control,
    units,
    commandChannel,
    stateChannel,
    stalenessColor,
    stalenessTimeout,
    fillColor,
    size,
    disabled,
    onClickDelay,
    inlineSize,
  },
}: NodeProps<schematic.SetpointNodeConfig>): ReactElement => {
  const resolve = Color.useResolve();
  const sink = Telem.numberSink(commandChannel);
  const source = Telem.numberSource(commandChannel);
  const { set, value } = BaseSetpoint.use({ aetherKey: nodeKey, sink, source });
  const badged = Staleness.useBadged();
  const { stale, valueRef } = useReadback({
    nodeKey,
    position,
    channel: stateChannel,
    size,
    stalenessColor,
    stalenessTimeout,
    staleColorHidden: badged,
  });
  const resize = Grid.useInlineResize(inlineSize, (inlineSize) =>
    onConfigChange({ inlineSize }),
  );
  return (
    <Grid.Grid nodeKey={nodeKey} allowRotate={false} editable={selected} {...resize}>
      <Control.State
        config={control}
        channel={commandChannel}
        onChange={onConfigChange}
      >
        {stale && badged ? <Staleness.Badge /> : null}
      </Control.State>
      <Label.Label config={label} onChange={onConfigChange} />
      <Setpoint
        onSend={set}
        value={value}
        units={units}
        fillColor={resolve(fillColor)}
        orientation={orientation}
        disabled={disabled}
        size={size}
        onClickDelay={onClickDelay}
        inlineSize={inlineSize}
        readback={
          stateChannel !== 0 && (
            <Primitive.Readout size={size} units={units} valueRef={valueRef} />
          )
        }
      />
    </Grid.Grid>
  );
};
