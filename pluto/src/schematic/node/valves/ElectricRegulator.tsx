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

const DIMENSIONS = { width: 87, height: 69 };

export const ElectricRegulator = ({
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
        left={2.29885}
        top={69.565}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={97.70115}
        top={69.565}
        id="2"
      />
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={23.1479}
        top={23.61}
        id="3"
      />
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={50}
        top={11.11}
        id="4"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={76.8521}
        top={23.61}
        id="5"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      orientation={orientation}
      color={color}
      scale={scale}
    >
      <Primitive.Path d="M43.5 49L6.35453 30.2035C4.35901 29.1937 2 30.6438 2 32.8803V65.1197C2 67.3562 4.35901 68.8063 6.35453 67.7965L43.5 49ZM43.5 49L80.6455 30.2035C82.641 29.1937 85 30.6438 85 32.8803V65.1197C85 67.3562 82.641 68.8063 80.6455 67.7965L43.5 49Z" />
      <Primitive.Rect x="21" y="6.5" width="44" height="18" rx="2" />
      <Primitive.Path d="M43 6.5V4C43 2.34315 44.3431 1 46 1H79C80.6569 1 82 2.34315 82 4V20.4281C82 21.4126 81.517 22.3344 80.7076 22.8947L43 49" />
      <Primitive.Line x1="43" y1="49" x2="43" y2="24.5" />
    </Primitive.SVG>
  </Primitive.Div>
);
