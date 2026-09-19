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
  createContext,
  type CSSProperties,
  type ReactElement,
  use,
  useMemo,
} from "react";

import { CSS } from "@/css";
import { useUniqueKey } from "@/hooks/useUniqueKey";
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

/** True inside a delayed toggle, so the SVG draws the hold fill within its shapes. */
export const HoldFill = createContext(false);
HoldFill.displayName = "Primitive.HoldFill";

// Strokes overflow the view box, so the fill reaches past it by a stroke width.
const HOLD_FILL_MARGIN = 2;

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
  const holdFill = use(HoldFill);
  const id = useUniqueKey();
  const holdBox = {
    x: -HOLD_FILL_MARGIN,
    y: -HOLD_FILL_MARGIN,
    width: dimsProp.width + 2 * HOLD_FILL_MARGIN,
    height: dimsProp.height + 2 * HOLD_FILL_MARGIN,
  };
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
      [CSS.variable("symbol-color")]: symbolColorVar(colorVal),
    }),
    [style, dims, scale, colorVal],
  );

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox={dimensions.svgViewBox(dims)}
      className={CSS.cls(CSS.B("symbol-colored"), CSS.loc(orientation), className)}
      {...rest}
      style={pStyle}
    >
      <g>
        {holdFill ? (
          <>
            <g id={id}>{children}</g>
            <mask
              id={`${id}-mask`}
              className={CSS.BE("symbol-hold", "mask")}
              maskUnits="userSpaceOnUse"
              {...holdBox}
            >
              <use href={`#${id}`} />
            </mask>
            <rect
              className={CSS.BE("symbol-hold", "fill")}
              mask={`url(#${id}-mask)`}
              {...holdBox}
            />
          </>
        ) : (
          children
        )}
      </g>
    </svg>
  );
};
