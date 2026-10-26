// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Primitive } from "@/schematic/node/common/primitive";
import { StringDisplay } from "@/schematic/node/general/stringDisplay/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { telem } from "@/telem/aether";
import { StringValue as BaseStringValue } from "@/vis/stringValue";

export const Symbol = ({
  nodeKey,
  position,
  onConfigChange,
  selected,
  config: {
    label,
    channel,
    stalenessTimeout,
    stalenessColor,
    strokeColor,
    textColor,
    level,
    inlineSize,
    orientation,
  },
}: NodeProps<schematic.StringDisplayNodeConfig>): ReactElement => {
  const content = Primitive.useCanvasBox<HTMLDivElement>(position);
  BaseStringValue.use({
    aetherKey: nodeKey,
    box: content.box,
    telem: telem.streamChannelStringValue({ channel: channel ?? 0 }),
    level,
    color: textColor,
    stalenessColor,
    stalenessTimeout,
  });
  const resize = Grid.useInlineResize(inlineSize, (inlineSize) =>
    onConfigChange({ inlineSize }),
  );
  return (
    <Grid.Grid editable={selected} nodeKey={nodeKey} allowRotate={false} {...resize}>
      <Label.Label config={label} onChange={onConfigChange} />
      <StringDisplay
        strokeColor={strokeColor}
        level={level}
        inlineSize={inlineSize}
        orientation={orientation}
        contentRef={content.ref}
      />
    </Grid.Grid>
  );
};
