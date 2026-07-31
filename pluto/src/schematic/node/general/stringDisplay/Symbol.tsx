// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Grid } from "@/schematic/node/common/grid";
import { Label } from "@/schematic/node/common/label";
import { type Config } from "@/schematic/node/general/stringDisplay/config";
import { StringDisplay } from "@/schematic/node/general/stringDisplay/Primitive";
import { type NodeProps } from "@/schematic/node/spec";
import { Theming } from "@/theming";
import { Value as BaseValue } from "@/vis/value";

const PRINTABLE_ASCII = Array.from({ length: 95 }, (_, i) =>
  String.fromCharCode(32 + i),
).join("");

// Glyph set for this symbol's text atlas. A string channel needs full printable
// ASCII; the default set is numeric and drops everything else.
const CHARACTERS = `${PRINTABLE_ASCII}°µ∞ᴇ`;

export const Symbol = ({
  nodeKey,
  position,
  onConfigChange,
  selected,
  config: {
    label,
    level = "p",
    textColor,
    color,
    telem: t,
    inlineSize = 100,
    stalenessColor,
    stalenessTimeout,
  },
}: NodeProps<Config>): ReactElement => {
  const font = Theming.useTypography(level);
  const valueBoxHeight = (font.lineHeight + 0.5) * font.baseSize + 2;
  const { width: oWidth } = BaseValue.use({
    aetherKey: nodeKey,
    color: textColor,
    level,
    box: box.construct(xy.translateY(position ?? xy.ZERO, 1), {
      height: valueBoxHeight,
      width: inlineSize,
    }),
    telem: t,
    minWidth: inlineSize,
    stalenessColor,
    stalenessTimeout,
    characters: CHARACTERS,
    numeric: false,
  });

  return (
    <Grid.Grid editable={selected} nodeKey={nodeKey} allowRotate={false}>
      <Label.Label config={label} onChange={onConfigChange} />
      <StringDisplay
        color={color}
        dimensions={{ height: valueBoxHeight, width: oWidth }}
        inlineSize={inlineSize}
      />
    </Grid.Grid>
  );
};
