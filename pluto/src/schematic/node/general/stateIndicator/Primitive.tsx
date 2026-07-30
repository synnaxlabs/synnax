// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/stateIndicator/stateIndicator.css";

import { type CSSProperties, type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/stateIndicator/config";
import { symbolColorVar } from "@/schematic/symbolColor";
import { Text } from "@/text";

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
  matchedOptionKey?: string | null;
}

export const StateIndicator = ({
  className,
  orientation = "left",
  matchedOptionKey,
  options,
  color: colorVal,
  inlineSize,
}: RenderProps): ReactElement => {
  const matched = options.find((o) => o.key === matchedOptionKey);
  // The matched state's color drives the chassis; the symbol color is the
  // fallback so an unmatched indicator still reads as its symbol.
  const symbolColor = symbolColorVar(matched?.color) ?? symbolColorVar(colorVal);
  const label = matched != null ? matched.name || `Option ${matched.value}` : "Unknown";
  const style = useMemo<CSSProperties>(
    () => ({
      [CSS.var("symbol-color")]: symbolColor,
      minWidth: inlineSize,
    }),
    [symbolColor, inlineSize],
  );
  return (
    <Primitive.Div
      className={CSS(
        CSS.B("state-indicator"),
        CSS.B("symbol-colored"),
        symbolColor != null && CSS.M("colored"),
        className,
      )}
      style={style}
    >
      <Handle.Rectangle
        orientation={orientation}
        left={0}
        top={-2}
        right={100}
        bottom={102}
      />
      <div className={CSS.BE("state-indicator", "content")}>
        <Text.Text level="p" variant="code">
          {label}
        </Text.Text>
      </div>
    </Primitive.Div>
  );
};
