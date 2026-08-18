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

export interface StrainerConeProps
  extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 33, height: 69 };

export const StrainerCone = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: StrainerConeProps): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cx(CSS.B("strainer"), className)}>
    <Handle.Linear orientation={orientation} left={6.06} right={93.04} />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="2" y="2" width="29" height="65" rx="1" />
      <Primitive.Path d="M31 34.5L2.30611 2.33992" strokeDasharray="6 6" />
      <Primitive.Path d="M31 34.5L2.30611 66.6601" strokeDasharray="6 6" />
    </Primitive.SVG>
  </Primitive.Div>
);
