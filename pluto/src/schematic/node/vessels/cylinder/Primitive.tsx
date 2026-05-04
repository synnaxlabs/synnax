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
import { Border } from "@/schematic/node/common/border";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive as Base } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/vessels/cylinder/config";
import { Theming } from "@/theming";

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
  onResize?: (dimensions: dimensions.Dimensions) => void;
}

export const Primitive = ({
  className,
  dimensions = Border.DEFAULT_DIMENSIONS,
  borderRadius = Border.DEFAULT_RADIUS,
  color: colorVal,
  backgroundColor,
}: RenderProps): ReactElement => {
  const detailedRadius = Border.parseRadius(borderRadius);
  const t = Theming.use();
  const refreshDeps = useMemo(
    () => [dimensions, borderRadius, detailedRadius],
    [
      detailedRadius.bottomLeft,
      detailedRadius.bottomRight,
      detailedRadius.topLeft,
      detailedRadius.topRight,
      dimensions.height,
      dimensions.width,
    ],
  );
  const boardColor = color.cssString(colorVal ?? t.colors.gray.l11);
  const bgColor =
    backgroundColor == null ? undefined : color.cssString(backgroundColor);
  const widthScale = dimensions.width / 66;
  const heightScale = dimensions.height / 180;
  const transform = `scale(${widthScale},${heightScale})`;

  return (
    <Base.Div className={CSS(className, CSS.B("cylinder"))} style={{ ...dimensions }}>
      <svg
        width="100%"
        height="100%"
        viewBox={`0 0 ${dimensions.width} ${dimensions.height}`}
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        transform={transform}
      >
        <path
          d="M23 33.6712C11.9844 36.0332 3 42.4382 3 52.8862V174.568C3 176.225 4.34315 177.568 6 177.568H60C61.6569 177.568 63 176.225 63 174.568V52.8862C63 36.3342 40.4511 29.9292 23 33.6712ZM23 33.6712V13.3181C23 0.318109 42.9975 0.318123 42.9975 13.3181V33.6712"
          vectorEffect="non-scaling-stroke"
          strokeWidth="2"
          stroke={boardColor}
          transform={transform}
          fill={bgColor}
        />
      </svg>
      <Handle.Boundary refreshDeps={refreshDeps} orientation="left">
        <Handle.Handle location="top" orientation="left" left={50} top={2} id="1" />
        <Handle.Handle location="left" orientation="left" left={35} top={10} id="9" />
        <Handle.Handle location="right" orientation="left" left={65} top={10} id="10" />
        <Handle.Handle
          location="bottom"
          orientation="left"
          left={50}
          top={98.3333}
          id="2"
        />
        <Handle.Handle location="left" orientation="left" left={4} top={40} id="3" />
        <Handle.Handle location="right" orientation="left" left={96} top={40} id="4" />
        <Handle.Handle location="left" orientation="left" left={4} top={60} id="5" />
        <Handle.Handle location="right" orientation="left" left={96} top={60} id="6" />
        <Handle.Handle location="left" orientation="left" left={4} top={80} id="7" />
        <Handle.Handle location="right" orientation="left" left={96} top={80} id="8" />
      </Handle.Boundary>
    </Base.Div>
  );
};
