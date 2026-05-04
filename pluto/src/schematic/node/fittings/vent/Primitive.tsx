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
export interface Props extends SVGBasedPrimitiveProps, DivProps {}

const DIMENSIONS = { width: 22, height: 32 };

export const Primitive = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: Props): ReactElement => (
  <Div className={CSS(CSS.B("vent"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={22.7273}
        top={50}
        id="1"
      />
      <Handle location="right" orientation={orientation} left={80} top={50} id="2" />
    </HandleBoundary>
    <InternalSVG
      color={color}
      dimensions={DIMENSIONS}
      orientation={orientation}
      scale={scale}
    >
      <Path
        d="M5 3L16.6325 13.8016C17.9107 14.9885 17.9107 17.0115 16.6325 18.1984L5 29"
        strokeLinecap="round"
      />
    </InternalSVG>
  </Div>
);
