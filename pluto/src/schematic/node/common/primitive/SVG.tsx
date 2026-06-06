// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { dimensions, direction } from "@synnaxlabs/x";
import {
  type ComponentPropsWithoutRef,
  type CSSProperties,
  type ReactElement,
  useMemo,
} from "react";

import { CSS } from "@/css";
import { type SVGBasedProps } from "@/schematic/node/common/primitive/orientable";
import { symbolColorVar } from "@/schematic/symbolColor";

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
  dimensions: dimsProp,
  orientation = "left",
  children,
  className,
  color: colorVal,
  style,
  scale = 1,
  ...rest
}: SVGProps): ReactElement => {
  const dir = direction.construct(orientation);
  const dims = useMemo(
    () => (dir === "y" ? dimensions.swap(dimsProp) : dimsProp),
    [dir, dimsProp],
  );
  const pStyle = useMemo<CSSProperties>(
    () => ({
      ...style,
      aspectRatio: `${dims.width} / ${dims.height}`,
      width: dimensions.scale(dims, scale * BASE_SCALE).width,
      [CSS.var("symbol-color")]: symbolColorVar(colorVal),
    }),
    [style, dims, scale, colorVal],
  );

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
