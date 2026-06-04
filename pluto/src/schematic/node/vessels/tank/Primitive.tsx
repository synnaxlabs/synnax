// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/vessels/tank/tank.css";

import { color, type dimensions } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Border } from "@/schematic/node/common/border";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/vessels/tank/config";

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
  onResize?: (dimensions: dimensions.Dimensions) => void;
  boxBorderRadius?: number;
  strokeWidth?: number;
}

export const Tank = ({
  className,
  dimensions = Border.DEFAULT_DIMENSIONS,
  borderRadius = Border.DEFAULT_RADIUS,
  boxBorderRadius,
  color: colorVal,
  backgroundColor,
  strokeWidth = 2,
}: RenderProps): ReactElement => {
  const detailedRadius = Border.parseRadius(borderRadius);
  const hasCornerBoundaries = boxBorderRadius == null;
  const { width, height } = dimensions;
  const refreshDeps = useMemo(
    () => [dimensions, borderRadius, detailedRadius],
    [
      detailedRadius.bottomLeft,
      detailedRadius.bottomRight,
      detailedRadius.topLeft,
      detailedRadius.topRight,
      height,
      width,
    ],
  );
  const leftOffset = Border.pixelToPercent(1, width);
  const rightOffset = 100 - leftOffset;
  const topOffset = Border.pixelToPercent(1, height);
  const bottomOffset = 100 - topOffset;
  return (
    <Primitive.Div
      className={CSS(className, CSS.B("tank"), CSS.B("symbol-colored"))}
      style={{
        ...dimensions,
        borderRadius: boxBorderRadius ?? Border.cssRadius(detailedRadius),
        [CSS.var("symbol-color")]:
          colorVal != null ? color.rgbString(colorVal) : undefined,
        borderColor: "var(--pluto-symbol-display)",
        backgroundColor: color.cssString(backgroundColor),
        borderWidth: strokeWidth,
      }}
    >
      <Handle.Boundary refreshDeps={refreshDeps} orientation="left">
        <Handle.Handle
          location="top"
          orientation="left"
          left={50}
          top={topOffset}
          id="1"
        />
        {hasCornerBoundaries && (
          <>
            <Handle.Handle
              location="top"
              orientation="left"
              left={leftOffset}
              top={detailedRadius.topLeft.y}
              id="2"
            />
            <Handle.Handle
              location="top"
              orientation="left"
              left={rightOffset}
              top={detailedRadius.topRight.y}
              id="3"
            />
          </>
        )}
        <Handle.Handle
          location="bottom"
          orientation="left"
          left={50}
          top={bottomOffset}
          id="4"
        />
        {hasCornerBoundaries && (
          <>
            <Handle.Handle
              location="bottom"
              orientation="left"
              left={leftOffset}
              top={100 - detailedRadius.bottomLeft.y}
              id="5"
            />
            <Handle.Handle
              location="bottom"
              orientation="left"
              left={rightOffset}
              top={100 - detailedRadius.bottomRight.y}
              id="6"
            />
          </>
        )}
        <Handle.Handle
          location="left"
          orientation="left"
          left={leftOffset}
          top={50}
          id="7"
        />
        <Handle.Handle
          location="right"
          orientation="left"
          left={rightOffset}
          top={50}
          id="8"
        />
      </Handle.Boundary>
    </Primitive.Div>
  );
};
