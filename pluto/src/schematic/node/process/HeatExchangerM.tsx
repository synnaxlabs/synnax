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

const DIMENSIONS = { width: 66, height: 66 };

export const HeatExchangerM = ({
  id,
  className,
  orientation = "right",
  color,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("heat-exchanger-M"), className)}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={89}
        top={27.27}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={89}
        top={72.73}
        id="2"
      />
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={50}
        top={4.545}
        id="3"
      />
      <Handle.Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={95.455}
        id="4"
      />
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={4.545}
        top={50}
        id="5"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path d="M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33L63 33Z" />
      <Primitive.Path d="M58.9698 18H23.56L38.5623 33.0023M38.5623 32.9977L23.56 48H58.9698" />
    </Primitive.SVG>
  </Primitive.Div>
);
