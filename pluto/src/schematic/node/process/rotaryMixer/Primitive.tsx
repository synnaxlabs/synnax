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

const DIMENSIONS = { width: 50, height: 33 };

export const Primitive = ({
  color,
  className,
  orientation = "left",
  scale,
  ...rest
}: Props): ReactElement => (
  <Toggle
    {...rest}
    className={CSS(CSS.B("rotary-mixer"), className)}
    orientation={orientation}
  >
    <HandleBoundary orientation={orientation}>
      <Handle location="left" orientation={orientation} left={2} top={48.4849} id="1" />
      <Handle
        location="right"
        orientation={orientation}
        left={97.5}
        top={48.4849}
        id="2"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Path d="M1 30V2C1 1.44772 1.44772 1 2 1H35.4545C35.7434 1 36.0181 1.12487 36.208 1.34247L48.4262 15.3425C48.7549 15.7192 48.7549 16.2808 48.4262 16.6575L36.208 30.6575C36.0181 30.8751 35.7434 31 35.4545 31H2C1.44772 31 1 30.5523 1 30Z" />
      <Line
        x1="32"
        y1="16"
        x2="40"
        y2="16"
        strokeLinecap="round"
        className={CSS(CSS.M("detail"), className)}
      />
      <Line
        x1="32"
        y1="16"
        x2="28"
        y2="22.9282"
        strokeLinecap="round"
        className={CSS(CSS.M("detail"), className)}
      />
      <Line
        x1="32"
        y1="16"
        x2="28"
        y2="9.0717"
        strokeLinecap="round"
        className={CSS(CSS.M("detail"), className)}
      />
    </InternalSVG>
  </Toggle>
);
