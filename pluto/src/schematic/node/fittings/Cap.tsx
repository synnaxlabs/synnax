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

export interface CapProps extends Primitive.SVGBasedProps, Primitive.DivProps {}

const DIMENSIONS = { width: 26, height: 48 };

export const Cap = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: CapProps): ReactElement => (
  <Primitive.Div className={CSS.cls(CSS.B("cap"), className)} {...rest}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={7.6923}
        top={50}
        id="1"
      />
    </Handle.Boundary>
    <Primitive.SVG
      color={color}
      dimensions={DIMENSIONS}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path d="M24 24C24 35.1852 15.2546 44.4725 3.87626 45.8297C2.90571 45.9455 2 45.1407 2 44V4C2 2.85926 2.90571 2.0545 3.87626 2.17027C15.2546 3.52755 24 12.8148 24 24Z" />
    </Primitive.SVG>
  </Primitive.Div>
);
