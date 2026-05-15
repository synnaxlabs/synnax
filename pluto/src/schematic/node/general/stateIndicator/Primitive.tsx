// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x/color";
import "@/schematic/node/general/stateIndicator/stateIndicator.css";

import { CSS } from "@synnaxlabs/lyra/css";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";

import { type ReactElement } from "react";

import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/stateIndicator/config";

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
  matchedOptionKey?: string | null;
}

export const Primitive = ({
  className,
  orientation = "left",
  matchedOptionKey,
  options,
  color: colorVal,
  inlineSize,
}: RenderProps): ReactElement => {
  const matched = options.find((o) => o.key === matchedOptionKey);
  const stateColor = matched?.color;
  const borderColor = colorVal != null ? color.cssString(colorVal) : undefined;
  const backgroundColor = stateColor != null ? color.cssString(stateColor) : undefined;
  const theme = Theming.use();
  const textColor =
    stateColor != null
      ? color.cssString(
          color.pickByContrast(stateColor, theme.colors.gray.l0, theme.colors.gray.l11),
        )
      : undefined;
  const label = matched != null ? matched.name || `Option ${matched.value}` : "Unknown";
  return (
    <Base.Div
      className={CSS(CSS.B("state-indicator"), className)}
      style={{ borderColor, backgroundColor, minWidth: inlineSize }}
    >
      <Handle.Boundary orientation={orientation}>
        <Handle.Handle location="left" orientation="left" left={0} top={50} id="1" />
        <Handle.Handle location="right" orientation="left" left={100} top={50} id="2" />
        <Handle.Handle location="top" orientation="left" left={50} top={-2} id="3" />
        <Handle.Handle
          location="bottom"
          orientation="left"
          left={50}
          top={102}
          id="4"
        />
      </Handle.Boundary>
      <div className={CSS.BE("state-indicator", "content")}>
        <Text.Text level="p" color={textColor} variant="code">
          {label}
        </Text.Text>
      </div>
    </Base.Div>
  );
};
