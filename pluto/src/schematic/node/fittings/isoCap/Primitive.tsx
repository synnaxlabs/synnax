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

const DIMENSIONS = { width: 36, height: 48 };

export const Primitive = ({
  className,
  orientation = "left",
  color,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Div className={CSS(CSS.B("cap"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={50} top={50} id="1" />
    </HandleBoundary>
    <InternalSVG
      color={color}
      dimensions={DIMENSIONS}
      orientation={orientation}
      scale={scale * 0.6}
    >
      <Path
        d="M3 3H30C31.6569 3 33 4.34315 33 6V42C33 43.6569 31.6569 45 30 45H3"
        strokeLinecap="round"
      />
    </InternalSVG>
  </Div>
);
