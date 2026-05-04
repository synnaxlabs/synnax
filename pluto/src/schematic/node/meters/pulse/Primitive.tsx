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
import { FlowmeterLabel } from "@/schematic/node/common/symbol/flowmeter";
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

const DIMENSIONS = { width: 71, height: 36 };

export const Primitive = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-Pulse"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle location="right" orientation={orientation} left={98} top={50} id="2" />
      <Handle location="top" orientation={orientation} left={50} top={6} id="3" />
      <Handle location="bottom" orientation={orientation} left={50} top={91} id="4" />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Path d="M31 13.5H39" strokeLinecap="round" />
      <Path d="M23 21.5H31" strokeLinecap="round" />
      <Path d="M39 21.5H47" strokeLinecap="round" />
      <Path d="M39 13.5V21.5" strokeLinecap="round" />
      <Path d="M31 13.5V21.5" strokeLinecap="round" />
      <FlowmeterLabel color={colorVal} />
    </InternalSVG>
  </Div>
);
