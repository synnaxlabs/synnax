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
export interface Props extends SVGBasedPrimitiveProps, DivProps {}

const DIMENSIONS = { width: 72, height: 36 };

export const Primitive = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Div className={CSS(CSS.B("orifice_plate"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.125} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.875}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      color={color}
      dimensions={DIMENSIONS}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2.5" y="2" width="67" height="31" rx="2" />
      <Path d="M24.5 2V7.5V13" strokeLinecap="round" />
      <Path d="M24.5 33V22" strokeLinecap="round" />
    </InternalSVG>
  </Div>
);
