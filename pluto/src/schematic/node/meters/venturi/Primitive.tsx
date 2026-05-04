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
const LABEL_POSITION = { x: 56, y: 29 };

export const Primitive = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flowmeter-Venturi"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={4} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={98.3333}
        top={50}
        id="2"
      />
      <Handle location="top" orientation={orientation} left={50} top={5.714} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={94.386}
        id="3"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2" width="67" height="31" rx="2" />
      <Path d="M8 33L26.5329 22.3" strokeLinecap="round" />
      <Path d="M8 2L26.5329 12.7" strokeLinecap="round" />
      <Path d="M56 33L26.5876 22.2948" strokeLinecap="round" />
      <Path d="M56 2L26.5876 12.7052" strokeLinecap="round" />
      <FlowmeterLabel position={LABEL_POSITION} color={colorVal} />
    </InternalSVG>
  </Div>
);
