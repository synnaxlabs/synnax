// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import type { ReactElement } from "react";

import { Channel } from "@/channel";
import { Color } from "@/color";
import { Deleted } from "@/schematic/node/common/deleted";
import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { Primitive as BasePrimitive } from "@/schematic/node/common/primitive";
import { Stale } from "@/schematic/node/common/stale";
import { Primitive } from "@/schematic/node/general/stringDisplay/primitive";
import type { NodeProps } from "@/schematic/node/spec";
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
    strokeColor,
    textColor,
    level,
    inlineSize,
    orientation,
  },
}: NodeProps<schematic.StringDisplayNodeConfig>): ReactElement => {
  const resolve = Color.useResolve();
  const content = BasePrimitive.useCanvasBox<HTMLDivElement>(position);
  const { stale } = BaseStringValue.use({
    aetherKey: nodeKey,
    box: content.box,
    telem: telem.streamChannelStringValue({ channel }),
    level,
    color: resolve(textColor),
    stalenessTimeout,
  });
  const deleted = Channel.useDeleted(channel);
  const resize = Grid.useInlineResize(inlineSize, (inlineSize) =>
    onConfigChange({ inlineSize }),
  );
  return (
    <Grid.Grid editable={selected} nodeKey={nodeKey} allowRotate={false} {...resize}>
      <Label.Label config={label} onChange={onConfigChange} />
      {stale && <Stale.Indicator />}
      {deleted && <Deleted.Item />}
      <Primitive.StringDisplay
        strokeColor={resolve(strokeColor)}
        level={level}
        inlineSize={inlineSize}
        orientation={orientation}
        contentRef={content.ref}
      />
    </Grid.Grid>
  );
};
