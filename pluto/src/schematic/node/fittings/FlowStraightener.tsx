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
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";

export interface FlowStraightenerProps
  extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 72, height: 36 };

export const FlowStraightener = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: FlowStraightenerProps): ReactElement => (
  <Primitive.Div className={CSS.cls(CSS.B("flow-straightener"), className)} {...rest}>
    <Handle.Rectangle
      orientation={orientation}
      left={3}
      top={6}
      right={97}
      bottom={93}
    />
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Rect x="2.5" y="2.5" width="67" height="31" rx="2" />
      <Primitive.Path d="M10.5 26.5H60.5" strokeLinecap="round" />
      <Primitive.Path d="M10.5 9.5H60.5" strokeLinecap="round" />
    </Primitive.SVG>
  </Primitive.Div>
);
