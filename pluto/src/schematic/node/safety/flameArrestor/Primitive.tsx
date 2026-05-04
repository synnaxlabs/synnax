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
  <Div {...rest} className={CSS(CSS.B("flame-arrestor"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={7.575} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={92.425}
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
      <Path
        d="M16.5 2.5L16.5 66.5M2.5 34.5H30.5M2.5 18.9848H30.5M2.5 50.0152H30.5M5.3 66.5H27.7C29.2464 66.5 30.5 65.1976 30.5 63.5909V5.40909C30.5 3.80244 29.2464 2.5 27.7 2.5H5.3C3.7536 2.5 2.5 3.80245 2.5 5.40909V63.5909C2.5 65.1976 3.7536 66.5 5.3 66.5Z"
        strokeLinecap="round"
      />
    </InternalSVG>
  </Div>
);
