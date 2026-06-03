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
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { type Config } from "@/schematic/node/general/circle/config";

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
}

export const Circle = ({
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
    <Primitive.Div className={CSS(className, CSS.B("circle-shape"))}>
      <Handle.Boundary orientation="left" refreshDeps={radius}>
        <Handle.Handle
          location="top"
          orientation="left"
          left={50}
          top={(padding / height) * 100}
          id="1"
        />
        <Handle.Handle
          location="bottom"
          orientation="left"
          left={50}
          top={((height - padding) / height) * 100}
          id="2"
        />
        <Handle.Handle
          location="left"
          orientation="left"
          left={(padding / width) * 100}
          top={50}
          id="3"
        />
        <Handle.Handle
          location="right"
          orientation="left"
          left={((width - padding) / width) * 100}
          top={50}
          id="4"
        />
      </Handle.Boundary>
      <Primitive.SVG dimensions={{ width, height }} color={colorVal}>
        <Primitive.Circle
          cx={width / 2}
          cy={height / 2}
          r={radius}
          strokeWidth={strokeWidth ?? 2}
          fill={color.cssString(backgroundColor)}
        />
      </Primitive.SVG>
    </Primitive.Div>
  );
};
