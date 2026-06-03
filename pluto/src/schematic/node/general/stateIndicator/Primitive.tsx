// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/stateIndicator/stateIndicator.css";

import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { resolveColor } from "@/schematic/node/common/color";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/stateIndicator/config";
import { Text } from "@/text";
import { Theming } from "@/theming";

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
  const stateColor = matched?.color;
  const theme = Theming.use();
  const borderColor = color.cssString(resolveColor(colorVal, theme));
  const backgroundColor = stateColor != null ? color.cssString(stateColor) : undefined;
  const textColor =
    stateColor != null
      ? color.cssString(
          color.pickByContrast(stateColor, theme.colors.gray.l0, theme.colors.gray.l11),
        )
      : undefined;
  const label = matched != null ? matched.name || `Option ${matched.value}` : "Unknown";
  return (
    <Primitive.Div
      className={CSS(CSS.B("state-indicator"), className)}
      style={{ borderColor, backgroundColor, minWidth: inlineSize }}
    >
      <Handle.Rectangle
        orientation={orientation}
        left={0}
        top={-2}
        right={100}
        bottom={102}
      />
      <div className={CSS.BE("state-indicator", "content")}>
        <Text.Text level="p" color={textColor} variant="code">
          {label}
        </Text.Text>
      </div>
    </Primitive.Div>
  );
};
