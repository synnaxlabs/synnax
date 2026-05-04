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
  Circle,
  Handle,
  HandleBoundary,
  InternalSVG,
  Line,
  Path,
  type SVGBasedPrimitiveProps,
  Toggle,
  type ToggleProps,
} from "@/schematic/node/common/symbol/primitives";
export interface Props extends ToggleProps, SVGBasedPrimitiveProps {}

const DIMENSIONS = { width: 64, height: 64 };

export const Primitive = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle
    {...rest}
    className={CSS(CSS.B("cavity-pump"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={3.125} top={50} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={96.875}
        top={50}
        id="2"
      />
      <Handle location="top" orientation={orientation} left={50} top={3.125} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={96.875}
        id="4"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Circle cx="32" cy="32" r="30" />
      <Line
        x1="32"
        y1="2"
        x2="62"
        y2="32"
        className={CSS(CSS.M("detail"), className)}
      />
      <Line
        x1="32"
        y1="62"
        x2="62"
        y2="32"
        className={CSS(CSS.M("detail"), className)}
      />
      <Path
        d="M 17 26 C 17 20.6667 23 20.6667 23 26 C 23 31.3333 29 31.3333 29 26 C 29 20.6667 35 20.6667 35 26"
        strokeLinecap="round"
        transform="translate(6, 6)"
        className={CSS(CSS.M("detail"), className)}
      />
    </InternalSVG>
  </Toggle>
);
