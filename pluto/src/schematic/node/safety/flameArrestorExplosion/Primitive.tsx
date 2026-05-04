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

const DIMENSIONS = { width: 60, height: 69 };

export const Primitive = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("flame-arrestor"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.333} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.667}
        top={50}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="2" y="2.5" width="56" height="64" rx="3" />
      <Path d="M30 2.5L30 66.5" />
      <Path d="M16 2.5L16 66.5" />
      <Path d="M2 34.5H30" />
      <Path d="M2 19H30" />
      <Path d="M2 50H30" />
    </InternalSVG>
  </Div>
);
