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
import {
  Div,
  type DivProps,
  Handle,
  HandleBoundary,
  InternalSVG,
  Path,
  Rect,
  type SVGBasedPrimitiveProps,
} from "@/schematic/node/common/symbol/primitives";
export interface Props extends DivProps, SVGBasedPrimitiveProps {}

const DIMENSIONS = { width: 33, height: 69 };

export const Primitive = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("strainer"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={6.06} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={93.04} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="29" height="65" rx="1" />
      <Path d="M31 34.5L2.30611 2.33992" strokeDasharray="6 6" />
      <Path d="M31 34.5L2.30611 66.6601" strokeDasharray="6 6" />
    </InternalSVG>
  </Div>
);
