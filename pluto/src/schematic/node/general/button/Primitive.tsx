// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color as colors } from "@synnaxlabs/x";
import { type CSSProperties, type MouseEventHandler, type ReactElement } from "react";

import { Button as Base } from "@/button";
import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/button/config";

interface ButtonProps extends Omit<Config, "variant"> {
  className?: string;
  onClick?: MouseEventHandler<HTMLButtonElement>;
  onMouseDown?: MouseEventHandler<HTMLButtonElement>;
  onMouseUp?: MouseEventHandler<HTMLButtonElement>;
}

// The button drives its background, border, and text off the theme-transformed
// symbol vars instead of the base button's concrete color, so it follows the
// console theme like every other symbol. The color is not forwarded to the base
// button, whose JS path would otherwise overwrite the contrast text color.
const themedStyle = (value?: colors.Crude): CSSProperties => ({
  [CSS.var("symbol-color")]:
    value != null && !colors.isZero(value)
      ? `${colors.rgbString(value)}, ${colors.aValue(value)}`
      : undefined,
  [CSS.var("bg")]: "var(--pluto-symbol-display)",
  [CSS.var("hover-bg")]: "oklch(from var(--pluto-symbol-display) l c h / 85%)",
  [CSS.var("active-bg")]: "oklch(from var(--pluto-symbol-display) l c h / 60%)",
  [CSS.var("border-color")]: "var(--pluto-symbol-display)",
  [CSS.var("hover-border-color")]:
    "oklch(from var(--pluto-symbol-display) l c h / 85%)",
  [CSS.var("active-border-color")]:
    "oklch(from var(--pluto-symbol-display) l c h / 70%)",
  [CSS.var("btn-text-color")]: "var(--pluto-symbol-contrast)",
});

export const Button = ({
  onClick,
  onMouseDown,
  onMouseUp,
  orientation = "left",
  label,
  color,
  size,
  level,
  onClickDelay: delay,
}: ButtonProps): ReactElement => (
  <Primitive.Div orientation={orientation}>
    <Base.Button
      variant="filled"
      className={CSS.B("symbol-colored")}
      style={themedStyle(color)}
      onClick={onClick}
      onMouseDown={onMouseDown}
      onMouseUp={onMouseUp}
      size={size}
      level={level}
      direction={label?.direction}
      onClickDelay={delay}
    >
      {label?.label ?? ""}
    </Base.Button>
    <Handle.Rectangle
      orientation={orientation}
      left={0}
      top={0}
      right={100}
      bottom={100}
    />
  </Primitive.Div>
);
