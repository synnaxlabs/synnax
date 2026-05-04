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

const DIMENSIONS = { width: 66, height: 30 };

export const Primitive = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Div
    {...rest}
    className={CSS(CSS.B("static-mixer"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={1.5152} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={98.4848}
        top={50}
        id="2"
      />
      <Handle location="top" orientation={orientation} left={50} top={3.3333} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={96.6667}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="1" y="1" width="64" height="28" rx="2" ry="2" />
      <Path d="M17 10C23 10 27 20 33 20C39 20 43 10 49 10" strokeLinecap="round" />
      <Path d="M17 20C23 20 27 10 33 10C39 10 43 20 49 20" strokeLinecap="round" />
    </InternalSVG>
  </Div>
);
