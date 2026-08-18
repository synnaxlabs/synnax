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

export const HeatExchangerGeneral = ({
  id,
  className,
  orientation = "right",
  color,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div
    {...rest}
    className={CSS.cx(CSS.B("heat-exchanger-general"), className)}
  >
    <Handle.Rectangle
      orientation={orientation}
      left={4.545}
      top={4.545}
      right={95.454}
      bottom={95.454}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path d="M3 33H18.3956L33.1284 18.1508M33.4276 47.8492L48.1604 33H63M33.278 48V18" />
      <Primitive.Path d="M63 33C63 36.9397 62.224 40.8407 60.7164 44.4805C59.2087 48.1203 56.999 51.4274 54.2132 54.2132C51.4274 56.999 48.1203 59.2087 44.4805 60.7164C40.8407 62.224 36.9397 63 33 63C29.0603 63 25.1593 62.224 21.5195 60.7164C17.8797 59.2087 14.5726 56.999 11.7868 54.2132C9.00104 51.4274 6.79125 48.1203 5.28361 44.4805C3.77597 40.8407 3 36.9397 3 33C3 29.0603 3.77597 25.1593 5.28362 21.5195C6.79126 17.8797 9.00104 14.5726 11.7868 11.7868C14.5726 9.00104 17.8797 6.79125 21.5195 5.28361C25.1593 3.77597 29.0603 3 33 3C36.9397 3 40.8407 3.77597 44.4805 5.28362C48.1203 6.79126 51.4274 9.00104 54.2132 11.7868C56.999 14.5726 59.2087 17.8797 60.7164 21.5195C62.224 25.1593 63 29.0603 63 33L63 33Z" />
    </Primitive.SVG>
  </Primitive.Div>
);
