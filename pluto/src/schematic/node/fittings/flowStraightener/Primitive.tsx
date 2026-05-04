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

const DIMENSIONS = { width: 72, height: 36 };

export const Primitive = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Div className={CSS(CSS.B("flow-straightener"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={97} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={93} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2.5" y="2.5" width="67" height="31" rx="2" />
      <Path d="M10.5 26.5H60.5" strokeLinecap="round" />
      <Path d="M10.5 9.5H60.5" strokeLinecap="round" />
    </InternalSVG>
  </Div>
);
