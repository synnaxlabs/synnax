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
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/vessels/cylinder/config";

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
  scale?: number;
  onResize?: (dimensions: dimensions.Dimensions) => void;
}

export const Cylinder = ({
  className,
  dimensions = Border.DEFAULT_DIMENSIONS,
  borderRadius = Border.DEFAULT_RADIUS,
  color: colorVal,
  backgroundColor,
  orientation = "left",
  scale,
}: RenderProps): ReactElement => {
  const detailedRadius = Border.parseRadius(borderRadius);
  const refreshDeps = useMemo(
    () => [dimensions, borderRadius, detailedRadius, orientation],
    [
      detailedRadius.bottomLeft,
      detailedRadius.bottomRight,
      detailedRadius.topLeft,
      detailedRadius.topRight,
      dimensions.height,
      dimensions.width,
      orientation,
    ],
  );
  const bgColor =
    backgroundColor == null ? undefined : color.cssString(backgroundColor);
  const transform = `scale(${dimensions.width / 66},${dimensions.height / 180})`;

  return (
    <Primitive.Div
      orientation={orientation}
      className={CSS(className, CSS.B("cylinder"))}
    >
      <Primitive.SVG
        dimensions={dimensions}
        orientation={orientation}
        scale={scale}
        color={colorVal}
      >
        <path
          d="M23 33.6712C11.9844 36.0332 3 42.4382 3 52.8862V174.568C3 176.225 4.34315 177.568 6 177.568H60C61.6569 177.568 63 176.225 63 174.568V52.8862C63 36.3342 40.4511 29.9292 23 33.6712ZM23 33.6712V13.3181C23 0.318109 42.9975 0.318123 42.9975 13.3181V33.6712"
          vectorEffect="non-scaling-stroke"
          strokeWidth="2"
          stroke="var(--pluto-symbol-display)"
          transform={transform}
          fill={bgColor}
        />
      </Primitive.SVG>
      <Handle.Boundary refreshDeps={refreshDeps} orientation={orientation}>
        <Handle.Handle
          location="top"
          orientation={orientation}
          left={50}
          top={2}
          id="1"
        />
        <Handle.Handle
          location="left"
          orientation={orientation}
          left={35}
          top={10}
          id="9"
        />
        <Handle.Handle
          location="right"
          orientation={orientation}
          left={65}
          top={10}
          id="10"
        />
        <Handle.Handle
          location="bottom"
          orientation={orientation}
          left={50}
          top={98.3333}
          id="2"
        />
        <Handle.Handle
          location="left"
          orientation={orientation}
          left={4}
          top={40}
          id="3"
        />
        <Handle.Handle
          location="right"
          orientation={orientation}
          left={96}
          top={40}
          id="4"
        />
        <Handle.Handle
          location="left"
          orientation={orientation}
          left={4}
          top={60}
          id="5"
        />
        <Handle.Handle
          location="right"
          orientation={orientation}
          left={96}
          top={60}
          id="6"
        />
        <Handle.Handle
          location="left"
          orientation={orientation}
          left={4}
          top={80}
          id="7"
        />
        <Handle.Handle
          location="right"
          orientation={orientation}
          left={96}
          top={80}
          id="8"
        />
      </Handle.Boundary>
    </Primitive.Div>
  );
};
