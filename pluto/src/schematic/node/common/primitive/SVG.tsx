// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, dimensions, direction } from "@synnaxlabs/x";
import { type ComponentPropsWithoutRef, type ReactElement } from "react";

import { CSS } from "@/css";
import { type SVGBasedProps } from "@/schematic/node/common/primitive/orientable";

export interface SVGProps
  extends
    SVGBasedProps,
    Omit<
      ComponentPropsWithoutRef<"svg">,
      "direction" | "color" | "orientation" | "scale"
    > {
  dimensions: dimensions.Dimensions;
}

export const BASE_SCALE = 0.8;

export const SVG = ({
  dimensions: dims,
  orientation = "left",
  children,
  className,
  color: colorVal,
  style = {},
  scale = 1,
  ...rest
}: SVGProps): ReactElement => {
  const dir = direction.construct(orientation);
  dims = dir === "y" ? dimensions.swap(dims) : dims;
  let pStyle = {
    ...style,
    aspectRatio: `${dims.width} / ${dims.height}`,
    width: dimensions.scale(dims, scale * BASE_SCALE).width,
  };
  if (colorVal != null && !color.isZero(colorVal))
    pStyle = {
      ...pStyle,
      [CSS.var("symbol-color")]:
        `${color.rgbString(colorVal)}, ${color.aValue(colorVal)}`,
    };

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox={dimensions.svgViewBox(dims)}
      className={CSS(CSS.B("symbol-colored"), CSS.loc(orientation), className)}
      {...rest}
      style={pStyle}
    >
      <g>{children}</g>
    </svg>
  );
};
