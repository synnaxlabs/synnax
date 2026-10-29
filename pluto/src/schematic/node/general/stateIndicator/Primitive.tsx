// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/stateIndicator/stateIndicator.css";

import { type schematic } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Text } from "@synnaxlabs/lyra/text";
import { color } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { SIZE_LEVELS } from "@/schematic/node/common/size";

interface RenderProps extends Partial<
  Pick<
    schematic.StateIndicatorNodeConfig,
    "strokeColor" | "orientation" | "inlineSize" | "size"
  >
> {
  options: schematic.StateIndicatorNodeConfig["options"];
  className?: string;
  matchedOptionKey?: string | null;
  /** Colors the label while the state channel is stale. */
  staleColor?: color.Crude;
}

export const StateIndicator = ({
  className,
  orientation = "left",
  matchedOptionKey,
  options,
  strokeColor,
  inlineSize,
  size = "medium",
  staleColor,
}: RenderProps): ReactElement => {
  const matched = options.find((o) => o.key === matchedOptionKey);
  // The matched state's color drives the chassis; the stroke color is the fallback so
  // an unmatched indicator still reads as its symbol.
  const symbolColor = color.rgbaString(matched?.color) ?? color.rgbaString(strokeColor);
  const label = matched != null ? matched.name || `Option ${matched.value}` : "Unknown";
  const style: CSSProperties = {
    [Primitive.SYMBOL_COLOR]: symbolColor,
    minWidth: inlineSize,
    height: Component.HEIGHTS[size],
  };
  return (
    <Primitive.Div
      className={CSS.cls(
        CSS.B("state-indicator"),
        CSS.symbolColored,
        CSS.B("symbol-readout"),
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
        <Text.Text
          level={SIZE_LEVELS[size]}
          color={color.cssString(staleColor)}
          variant="code"
        >
          {label}
        </Text.Text>
      </div>
    </Primitive.Div>
  );
};
