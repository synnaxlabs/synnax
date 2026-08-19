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

const DIMENSIONS = { width: 45, height: 43 };

export const Check = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Primitive.Div
    orientation={orientation}
    className={CSS.cls(CSS.B("check-valve"), className)}
    {...rest}
  >
    <Handle.Linear
      orientation={orientation}
      left={4.2222}
      right={94.2038}
      top={48.8372}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Line x1="42.3917" y1="2" x2="42.3917" y2="41" strokeLinecap="round" />
      <Primitive.Path d="M41.6607 21.8946C42.3917 21.5238 42.3906 20.4794 41.6589 20.1101L6.25889 2.2412C4.26237 1.23341 1.90481 2.68589 1.90704 4.92235L1.93925 37.1617C1.94148 39.3982 4.30194 40.846 6.29644 39.8342L41.6607 21.8946Z" />
    </Primitive.SVG>
  </Primitive.Div>
);
