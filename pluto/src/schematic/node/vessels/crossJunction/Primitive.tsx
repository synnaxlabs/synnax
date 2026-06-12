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

export interface CrossJunctionProps
  extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 39, height: 39 };

export const CrossJunction = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  ...rest
}: CrossJunctionProps): ReactElement => (
  <Primitive.Div className={CSS(CSS.B("t-junction"), className)} {...rest}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="left"
        orientation={orientation}
        left={3.8462}
        top={50}
        id="1"
      />
      <Handle.Handle
        location="right"
        orientation={orientation}
        left={96.1539}
        top={50}
        id="2"
      />
      <Handle.Handle
        location="bottom"
        orientation={orientation}
        left={50.5}
        top={96.1539}
        id="3"
      />
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={50.5}
        top={3.8462}
        id="4"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path
        d="M22.5 35.5C22.5 36.6046 21.6046 37.5 20.5 37.5H18.5C17.3954 37.5 16.5 36.6046 16.5 35.5V24.5C16.5 23.3954 15.6046 22.5 14.5 22.5H3.5C2.39543 22.5 1.5 21.6046 1.5 20.5V18.5C1.5 17.3954 2.39543 16.5 3.5 16.5H14.5C15.6046 16.5 16.5 15.6046 16.5 14.5V3.5C16.5 2.39543 17.3954 1.5 18.5 1.5H20.5C21.6046 1.5 22.5 2.39543 22.5 3.5V14.5C22.5 15.6046 23.3954 16.5 24.5 16.5H35.5C36.6046 16.5 37.5 17.3954 37.5 18.5V20.5C37.5 21.6046 36.6046 22.5 35.5 22.5H24.5C23.3954 22.5 22.5 23.3954 22.5 24.5V35.5Z"
        fill="var(--pluto-symbol-display)"
        stroke="none"
      />
    </Primitive.SVG>
  </Primitive.Div>
);
