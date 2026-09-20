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

export interface StrainerProps extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 33, height: 69 };

export const Strainer = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: StrainerProps): ReactElement => (
  <Primitive.Div {...rest} className={CSS.cls(CSS.B("strainer"), className)}>
    <Handle.Linear orientation={orientation} left={6.06} right={93.04} />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="2" y="2" width="29" height="65" rx="1" />
      <Primitive.Path d="M2.293 2.29297L29.9383 66.7986" strokeDasharray="6 6" />
    </Primitive.SVG>
  </Primitive.Div>
);
