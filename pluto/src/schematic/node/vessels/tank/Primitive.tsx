// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type dimensions } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import {
  cssBorderRadius,
  DEFAULT_BORDER_RADIUS,
  DEFAULT_DIMENSIONS,
  parseBorderRadius,
  pixelToPercent,
} from "@/schematic/node/common/symbol/border";
import { Div, Handle, HandleBoundary } from "@/schematic/node/common/symbol/primitives";
import { type Config } from "@/schematic/node/vessels/tank/config";
import { Theming } from "@/theming";

interface RenderProps extends Config {
  className?: string;
  onResize?: (dimensions: dimensions.Dimensions) => void;
  boxBorderRadius?: number;
  strokeWidth?: number;
}

export const Primitive = ({
  className,
  dimensions = DEFAULT_DIMENSIONS,
  borderRadius = DEFAULT_BORDER_RADIUS,
  boxBorderRadius,
  color: colorVal,
  backgroundColor,
  strokeWidth = 2,
}: RenderProps): ReactElement => {
  const detailedRadius = parseBorderRadius(borderRadius);
  const hasCornerBoundaries = boxBorderRadius == null;
  const t = Theming.use();
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
  const leftOffset = pixelToPercent(1, width);
  const rightOffset = 100 - leftOffset;
  const topOffset = pixelToPercent(1, height);
  const bottomOffset = 100 - topOffset;
  return (
    <Div
      className={CSS(className, CSS.B("tank"))}
      style={{
        ...dimensions,
        borderRadius: boxBorderRadius ?? cssBorderRadius(detailedRadius),
        borderColor: color.cssString(colorVal ?? t.colors.gray.l11),
        backgroundColor: color.cssString(backgroundColor),
        borderWidth: strokeWidth,
      }}
    >
      <HandleBoundary refreshDeps={refreshDeps} orientation="left">
        <Handle location="top" orientation="left" left={50} top={topOffset} id="1" />
        {hasCornerBoundaries && (
          <>
            <Handle
              location="top"
              orientation="left"
              left={leftOffset}
              top={detailedRadius.topLeft.y}
              id="2"
            />
            <Handle
              location="top"
              orientation="left"
              left={rightOffset}
              top={detailedRadius.topRight.y}
              id="3"
            />
          </>
        )}
        <Handle
          location="bottom"
          orientation="left"
          left={50}
          top={bottomOffset}
          id="4"
        />
        {hasCornerBoundaries && (
          <>
            <Handle
              location="bottom"
              orientation="left"
              left={leftOffset}
              top={100 - detailedRadius.bottomLeft.y}
              id="5"
            />
            <Handle
              location="bottom"
              orientation="left"
              left={rightOffset}
              top={100 - detailedRadius.bottomRight.y}
              id="6"
            />
          </>
        )}
        <Handle location="left" orientation="left" left={leftOffset} top={50} id="7" />
        <Handle
          location="right"
          orientation="left"
          left={rightOffset}
          top={50}
          id="8"
        />
      </HandleBoundary>
    </Div>
  );
};
