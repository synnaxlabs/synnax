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
  Line,
  Rect,
  type SVGBasedPrimitiveProps,
} from "@/schematic/node/common/symbol/primitives";
export interface Props extends DivProps, SVGBasedPrimitiveProps {}

const DIMENSIONS = { width: 199, height: 48 };

export const Primitive = ({
  id,
  className,
  orientation = "right",
  color: colorVal,
  scale = 1,
  ...rest
}: Props): ReactElement => (
  <Div {...rest} className={CSS(CSS.B("heat-exchanger-M"), className)}>
    <HandleBoundary orientation={orientation}>
      <Handle location="top" orientation={orientation} left={9} top={6.25} id="1" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={24}
        top={93.75}
        id="2"
      />
      <Handle location="top" orientation={orientation} left={76} top={6.25} id="3" />
      <Handle
        location="bottom"
        orientation={orientation}
        left={91}
        top={93.75}
        id="4"
      />
      <Handle location="left" orientation={orientation} left={1.508} top={50} id="5" />
      <Handle
        location="right"
        orientation={orientation}
        left={98.492}
        top={50}
        id="6"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Rect x="3" y="3" width="193" height="42" rx="1" strokeWidth="2" />
      <Rect x="32.397" y="3" width="134.206" height="42" rx="1" strokeWidth="2" />
      <Rect x="32.397" y="13.5" width="134.206" height="21" rx="1" strokeWidth="2" />
      <Line x1="32.3769" y1="24" x2="166.623" y2="24" strokeWidth="2" />
    </InternalSVG>
  </Div>
);
