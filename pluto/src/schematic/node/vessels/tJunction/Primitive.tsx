// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
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

const DIMENSIONS = { width: 39, height: 21 };

export const Primitive = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  ...rest
}: Props): ReactElement => (
  <Div className={CSS(CSS.B("t-junction"), className)} {...rest}>
    <HandleBoundary orientation={orientation}>
      <Handle
        location="left"
        orientation={orientation}
        left={3.8462}
        top={22.3095}
        id="1"
      />
      <Handle
        location="right"
        orientation={orientation}
        left={96.1538}
        top={22.3095}
        id="2"
      />
      <Handle
        location="bottom"
        orientation={orientation}
        left={50}
        top={92.8571}
        id="3"
      />
    </HandleBoundary>
    <InternalSVG
      dimensions={DIMENSIONS}
      color={colorVal}
      orientation={orientation}
      scale={scale}
    >
      <Path
        d="M1.5 5.5V3.5C1.5 2.39543 2.39543 1.5 3.5 1.5H35.5C36.6046 1.5 37.5 2.39543 37.5 3.5V5.5C37.5 6.60457 36.6046 7.5 35.5 7.5H24.5C23.3954 7.5 22.5 8.39543 22.5 9.5V17.5C22.5 18.6046 21.6046 19.5 20.5 19.5H18.5C17.3954 19.5 16.5 18.6046 16.5 17.5V9.5C16.5 8.39543 15.6046 7.5 14.5 7.5H3.5C2.39543 7.5 1.5 6.60457 1.5 5.5Z"
        fill={color.cssString(colorVal)}
        stroke="none"
      />
    </InternalSVG>
  </Div>
);
