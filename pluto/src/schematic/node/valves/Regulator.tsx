// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";

export interface Props extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 88, height: 80 };

export const Regulator = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div className={CSS.cls(className, CSS.B("regulator"))} {...rest}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={3.4091}
        top={66.25}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={97.7273}
        top={66.25}
        id="2"
      />
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={50.9091}
        top={12.5}
        id="3"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Primitive.Path d="M44.5 53L7.35453 34.2035C5.35901 33.1937 3 34.6438 3 36.8803V69.1197C3 71.3562 5.35901 72.8063 7.35453 71.7965L44.5 53ZM44.5 53L81.6455 34.2035C83.641 33.1937 86 34.6438 86 36.8803V69.1197C86 71.3562 83.641 72.8063 81.6455 71.7965L44.5 53Z" />
      <Primitive.Path d="M61 30C62.6569 30 64.0231 28.6494 63.7755 27.0111C63.141 22.8129 61.181 18.8968 58.1421 15.8579C54.3914 12.1071 49.3043 10 44 10C38.6957 10 33.6086 12.1071 29.8579 15.8579C26.819 18.8968 24.859 22.8129 24.2245 27.0111C23.9769 28.6494 25.3431 30 27 30L44.5 30H61Z" />
      <Primitive.Line x1="44.5" y1="53" x2="44.5" y2="30" strokeLinecap="round" />
      <Primitive.Path d="M44.5 10V8C44.5 6.34315 45.3431 5 47 5H80C81.6569 5 83 6.34315 83 8V24.4281C83 25.4126 82.517 26.3344 81.7076 26.8947L44.5 53" />
    </Primitive.SVG>
  </Primitive.Div>
);
