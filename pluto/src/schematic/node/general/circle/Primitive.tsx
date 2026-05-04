// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import {
  Circle,
  Div,
  Handle,
  HandleBoundary,
  InternalSVG,
} from "@/schematic/node/common/symbol/primitives";
import { type Config } from "@/schematic/node/general/circle/config";

interface RenderProps extends Config {
  className?: string;
}

export const Primitive = ({
  radius,
  color: colorVal,
  backgroundColor,
  className,
  strokeWidth,
}: RenderProps): ReactElement => {
  const padding = (strokeWidth ?? 2) + 1;
  const diameter = radius * 2;
  const width = diameter + 2 * padding;
  const height = diameter + 2 * padding;
  return (
    <Div className={CSS(className, CSS.B("circle-shape"))}>
      <HandleBoundary orientation="left" refreshDeps={radius}>
        <Handle
          location="top"
          orientation="left"
          left={50}
          top={(padding / height) * 100}
          id="1"
        />
        <Handle
          location="bottom"
          orientation="left"
          left={50}
          top={((height - padding) / height) * 100}
          id="2"
        />
        <Handle
          location="left"
          orientation="left"
          left={(padding / width) * 100}
          top={50}
          id="3"
        />
        <Handle
          location="right"
          orientation="left"
          left={((width - padding) / width) * 100}
          top={50}
          id="4"
        />
      </HandleBoundary>
      <InternalSVG dimensions={{ width, height }}>
        <Circle
          cx={width / 2}
          cy={height / 2}
          r={radius}
          stroke={color.cssString(colorVal)}
          strokeWidth={strokeWidth ?? 2}
          fill={color.cssString(backgroundColor)}
        />
      </InternalSVG>
    </Div>
  );
};
